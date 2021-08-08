// Copyright 2020 gorse Project Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package ranking

import (
	"fmt"
	"sync"
	"time"

	"github.com/zhenghaoz/gorse/base"
	"github.com/zhenghaoz/gorse/model"
	"go.uber.org/zap"
)

// ParamsSearchResult contains the return of grid search.
type ParamsSearchResult struct {
	BestModel  Model
	BestScore  Score
	BestParams model.Params
	BestIndex  int
	Scores     []Score
	Params     []model.Params
}

func NewParamsSearchResult() *ParamsSearchResult {
	return &ParamsSearchResult{
		Scores: make([]Score, 0),
		Params: make([]model.Params, 0),
	}
}

func (r *ParamsSearchResult) AddScore(params model.Params, score Score) {
	r.Scores = append(r.Scores, score)
	r.Params = append(r.Params, params.Copy())
	if len(r.Scores) == 0 || score.NDCG > r.BestScore.NDCG {
		r.BestScore = score
		r.BestParams = params.Copy()
		r.BestIndex = len(r.Params) - 1
	}
}

// GridSearchCV finds the best parameters for a model.
func GridSearchCV(estimator Model, trainSet *DataSet, testSet *DataSet, paramGrid model.ParamsGrid,
	seed int64, fitConfig *FitConfig) ParamsSearchResult {
	// Retrieve parameter names and length
	paramNames := make([]model.ParamName, 0, len(paramGrid))
	count := 1
	for paramName, values := range paramGrid {
		paramNames = append(paramNames, paramName)
		count *= len(values)
	}
	// Construct DFS procedure
	results := ParamsSearchResult{
		Scores: make([]Score, 0, count),
		Params: make([]model.Params, 0, count),
	}
	var dfs func(deep int, params model.Params)
	progress := 0
	dfs = func(deep int, params model.Params) {
		if deep == len(paramNames) {
			progress++
			base.Logger().Info(fmt.Sprintf("grid search (%v/%v)", progress, count),
				zap.Any("params", params))
			// Cross validate
			estimator.Clear()
			estimator.SetParams(estimator.GetParams().Overwrite(params))
			score := estimator.Fit(trainSet, testSet, fitConfig)
			// Create GridSearch result
			results.Scores = append(results.Scores, score)
			results.Params = append(results.Params, params.Copy())
			if len(results.Scores) == 0 || score.NDCG > results.BestScore.NDCG {
				results.BestModel = Clone(estimator)
				results.BestScore = score
				results.BestParams = params.Copy()
				results.BestIndex = len(results.Params) - 1
			}
		} else {
			paramName := paramNames[deep]
			values := paramGrid[paramName]
			for _, val := range values {
				params[paramName] = val
				dfs(deep+1, params)
			}
		}
	}
	params := make(map[model.ParamName]interface{})
	dfs(0, params)
	return results
}

// RandomSearchCV searches hyper-parameters by random.
func RandomSearchCV(estimator Model, trainSet *DataSet, testSet *DataSet, paramGrid model.ParamsGrid,
	numTrials int, seed int64, fitConfig *FitConfig) ParamsSearchResult {
	// if the number of combination is less than number of trials, use grid search
	if paramGrid.NumCombinations() < numTrials {
		return GridSearchCV(estimator, trainSet, testSet, paramGrid, seed, fitConfig)
	}
	rng := base.NewRandomGenerator(seed)
	results := ParamsSearchResult{
		Scores: make([]Score, 0, numTrials),
		Params: make([]model.Params, 0, numTrials),
	}
	for i := 1; i <= numTrials; i++ {
		// Make parameters
		params := model.Params{}
		for paramName, values := range paramGrid {
			value := values[rng.Intn(len(values))]
			params[paramName] = value
		}
		// Cross validate
		base.Logger().Info(fmt.Sprintf("random search (%v/%v)", i, numTrials),
			zap.Any("params", params))
		estimator.Clear()
		estimator.SetParams(estimator.GetParams().Overwrite(params))
		score := estimator.Fit(trainSet, testSet, fitConfig)
		results.Scores = append(results.Scores, score)
		results.Params = append(results.Params, params.Copy())
		if len(results.Scores) == 0 || score.NDCG > results.BestScore.NDCG {
			results.BestModel = Clone(estimator)
			results.BestScore = score
			results.BestParams = params.Copy()
			results.BestIndex = len(results.Params) - 1
		}
	}
	return results
}

// ModelSearcher is a thread-safe personal ranking model searcher.
type ModelSearcher struct {
	// arguments
	numEpochs int
	numTrials int
	numJobs   int
	// results
	bestMutex      sync.Mutex
	bestModelName  string
	bestModel      Model
	bestScore      Score
	bestSimilarity string
}

// NewModelSearcher creates a thread-safe personal ranking model searcher.
func NewModelSearcher(nEpoch, nTrials, nJobs int) *ModelSearcher {
	return &ModelSearcher{
		numTrials:      nTrials,
		numEpochs:      nEpoch,
		numJobs:        nJobs,
		bestSimilarity: model.SimilarityCosine,
	}
}

// GetBestModel returns the optimal personal ranking model.
func (searcher *ModelSearcher) GetBestModel() (string, Model, Score) {
	searcher.bestMutex.Lock()
	defer searcher.bestMutex.Unlock()
	return searcher.bestModelName, searcher.bestModel, searcher.bestScore
}

// GetBestSimilarity returns the optimal similarity for neighborhood recommendations.
func (searcher *ModelSearcher) GetBestSimilarity() string {
	searcher.bestMutex.Lock()
	defer searcher.bestMutex.Unlock()
	return searcher.bestSimilarity
}

func (searcher *ModelSearcher) Fit(trainSet, valSet *DataSet) error {
	base.Logger().Info("ranking model search",
		zap.Int("n_users", trainSet.UserCount()),
		zap.Int("n_items", trainSet.ItemCount()))
	startTime := time.Now()
	models := []string{"bpr", "ccd", "knn"}
	for _, name := range models {
		m, err := NewModel(name, model.Params{model.NEpochs: searcher.numEpochs})
		if err != nil {
			return err
		}
		r := RandomSearchCV(m, trainSet, valSet, m.GetParamsGrid(), searcher.numTrials, 0,
			NewFitConfig().SetJobs(searcher.numJobs))
		searcher.bestMutex.Lock()
		if name == "knn" {
			searcher.bestSimilarity = r.BestModel.GetParams()[model.Similarity].(string)
		}
		if searcher.bestModel == nil || r.BestScore.NDCG > searcher.bestScore.NDCG {
			searcher.bestModelName = name
			searcher.bestModel = r.BestModel
			searcher.bestScore = r.BestScore
		}
		searcher.bestMutex.Unlock()
	}
	searchTime := time.Since(startTime)
	base.Logger().Info("complete ranking model search",
		zap.Float32("NDCG@10", searcher.bestScore.NDCG),
		zap.Float32("Precision@10", searcher.bestScore.Precision),
		zap.Float32("Recall@10", searcher.bestScore.Recall),
		zap.String("model", searcher.bestModelName),
		zap.Any("params", searcher.bestModel.GetParams()),
		zap.String("search_time", searchTime.String()))
	return nil
}

// Copyright 2021 gorse Project Authors
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
	"bufio"
	"bytes"
	"encoding/gob"
	"fmt"
	"time"

	"github.com/barkimedes/go-deepcopy"

	"github.com/chewxy/math32"
	"github.com/zhenghaoz/gorse/base"
	"github.com/zhenghaoz/gorse/floats"
	"github.com/zhenghaoz/gorse/model"
	"go.uber.org/zap"
	"gonum.org/v1/gonum/mat"
)

type Score struct {
	NDCG      float32
	Precision float32
	Recall    float32
}

type FitConfig struct {
	Jobs       int
	Verbose    int
	Candidates int
	TopK       int
}

func NewFitConfig() *FitConfig {
	return &FitConfig{
		Jobs:       1,
		Verbose:    10,
		Candidates: 100,
		TopK:       10,
	}
}

func (config *FitConfig) SetJobs(nJobs int) *FitConfig {
	config.Jobs = nJobs
	return config
}

func (config *FitConfig) LoadDefaultIfNil() *FitConfig {
	if config == nil {
		return NewFitConfig()
	}
	return config
}

type Model interface {
	model.Model
	// Fit a model with a train set and parameters.
	Fit(trainSet *DataSet, validateSet *DataSet, config *FitConfig) Score
	// GetItemIndex returns item index.
	GetItemIndex() base.Index
}

type MatrixFactorization interface {
	Model
	// Predict the rating given by a user (userId) to a item (itemId).
	Predict(userId, itemId string) float32
	// InternalPredict predicts rating given by a user index and a item index
	InternalPredict(userIndex, itemIndex int) float32
	// GetUserIndex returns user index.
	GetUserIndex() base.Index
}

type BaseMatrixFactorization struct {
	model.BaseModel
	UserIndex base.Index
	ItemIndex base.Index
}

func (model *BaseMatrixFactorization) Init(trainSet *DataSet) {
	model.UserIndex = trainSet.UserIndex
	model.ItemIndex = trainSet.ItemIndex
}

func (model *BaseMatrixFactorization) GetUserIndex() base.Index {
	return model.UserIndex
}

func (model *BaseMatrixFactorization) GetItemIndex() base.Index {
	return model.ItemIndex
}

func NewModel(name string, params model.Params) (Model, error) {
	switch name {
	case "als":
		return NewALS(params), nil
	case "bpr":
		return NewBPR(params), nil
	case "ccd":
		return NewCCD(params), nil
	case "knn":
		return NewKNN(params), nil
	}
	return nil, fmt.Errorf("unknown model %v", name)
}

// Clone a model with deep copy.
func Clone(m Model) Model {
	if temp, err := deepcopy.Anything(m); err != nil {
		panic(err)
	} else {
		copied := temp.(Model)
		copied.SetParams(copied.GetParams())
		return copied
	}
}

func EncodeModel(m Model) ([]byte, error) {
	buf := bytes.NewBuffer(nil)
	writer := bufio.NewWriter(buf)
	encoder := gob.NewEncoder(writer)
	if err := encoder.Encode(m); err != nil {
		return nil, err
	}
	if err := writer.Flush(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func DecodeModel(name string, buf []byte) (Model, error) {
	reader := bytes.NewReader(buf)
	decoder := gob.NewDecoder(reader)
	switch name {
	case "als":
		var als ALS
		if err := decoder.Decode(&als); err != nil {
			return nil, err
		}
		return &als, nil
	case "bpr":
		var bpr BPR
		if err := decoder.Decode(&bpr); err != nil {
			return nil, err
		}
		return &bpr, nil
	case "ccd":
		var ccd CCD
		if err := decoder.Decode(&ccd); err != nil {
			return nil, err
		}
		return &ccd, nil
	case "knn":
		var knn KNN
		if err := decoder.Decode(&knn); err != nil {
			return nil, err
		}
		return &knn, nil
	}
	return nil, fmt.Errorf("unknown model %v", name)
}

// BPR means Bayesian Personal Ranking, is a pairwise learning algorithm for matrix factorization
// model with implicit feedback. The pairwise ranking between item i and j for user u is estimated
// by:
//
//   p(i >_u j) = \sigma( p_u^T (q_i - q_j) )
//
// Hyper-parameters:
//	 Reg 		- The regularization parameter of the cost function that is
// 				  optimized. Default is 0.01.
//	 Lr 		- The learning rate of SGD. Default is 0.05.
//	 nFactors	- The number of latent factors. Default is 10.
//	 NEpochs	- The number of iteration of the SGD procedure. Default is 100.
//	 InitMean	- The mean of initial random latent factors. Default is 0.
//	 InitStdDev	- The standard deviation of initial random latent factors. Default is 0.001.
type BPR struct {
	BaseMatrixFactorization
	// Model parameters
	UserFactor [][]float32 // p_u
	ItemFactor [][]float32 // q_i
	// Hyper parameters
	nFactors   int
	nEpochs    int
	lr         float32
	reg        float32
	initMean   float32
	initStdDev float32
}

// NewBPR creates a BPR model.
func NewBPR(params model.Params) *BPR {
	bpr := new(BPR)
	bpr.SetParams(params)
	return bpr
}

// SetParams sets hyper-parameters of the BPR model.
func (bpr *BPR) SetParams(params model.Params) {
	bpr.BaseMatrixFactorization.SetParams(params)
	// Setup hyper-parameters
	bpr.nFactors = bpr.Params.GetInt(model.NFactors, 10)
	bpr.nEpochs = bpr.Params.GetInt(model.NEpochs, 100)
	bpr.lr = bpr.Params.GetFloat32(model.Lr, 0.05)
	bpr.reg = bpr.Params.GetFloat32(model.Reg, 0.01)
	bpr.initMean = bpr.Params.GetFloat32(model.InitMean, 0)
	bpr.initStdDev = bpr.Params.GetFloat32(model.InitStdDev, 0.001)
}

func (bpr *BPR) GetParamsGrid() model.ParamsGrid {
	return model.ParamsGrid{
		model.NFactors:   []interface{}{8, 16, 32, 64},
		model.Lr:         []interface{}{0.001, 0.005, 0.01, 0.05, 0.1},
		model.Reg:        []interface{}{0.001, 0.005, 0.01, 0.05, 0.1},
		model.InitMean:   []interface{}{0},
		model.InitStdDev: []interface{}{0.001, 0.005, 0.01, 0.05, 0.1},
	}
}

// Predict by the BPR model.
func (bpr *BPR) Predict(userId, itemId string) float32 {
	// Convert sparse Names to dense Names
	userIndex := bpr.UserIndex.ToNumber(userId)
	itemIndex := bpr.ItemIndex.ToNumber(itemId)
	if userIndex == base.NotId {
		base.Logger().Warn("unknown user", zap.String("user_id", userId))
	}
	if itemIndex == base.NotId {
		base.Logger().Warn("unknown item", zap.String("item_id", itemId))
	}
	return bpr.InternalPredict(userIndex, itemIndex)
}

func (bpr *BPR) InternalPredict(userIndex, itemIndex int) float32 {
	ret := float32(0.0)
	// + q_i^Tp_u
	if itemIndex != base.NotId && userIndex != base.NotId {
		ret += floats.Dot(bpr.UserFactor[userIndex], bpr.ItemFactor[itemIndex])
	} else {
		base.Logger().Warn("unknown user or item")
	}
	return ret
}

// Fit the BPR model.
func (bpr *BPR) Fit(trainSet, valSet *DataSet, config *FitConfig) Score {
	config = config.LoadDefaultIfNil()
	base.Logger().Info("fit bpr",
		zap.Int("train_set_size", trainSet.Count()),
		zap.Int("test_set_size", valSet.Count()),
		zap.Any("params", bpr.GetParams()),
		zap.Any("config", config))
	bpr.Init(trainSet)
	// Create buffers
	temp := base.NewMatrix32(config.Jobs, bpr.nFactors)
	userFactor := base.NewMatrix32(config.Jobs, bpr.nFactors)
	positiveItemFactor := base.NewMatrix32(config.Jobs, bpr.nFactors)
	negativeItemFactor := base.NewMatrix32(config.Jobs, bpr.nFactors)
	rng := make([]base.RandomGenerator, config.Jobs)
	for i := 0; i < config.Jobs; i++ {
		rng[i] = base.NewRandomGenerator(bpr.GetRandomGenerator().Int63())
	}
	// Convert array to hashmap
	userFeedback := make([]map[int]interface{}, trainSet.UserCount())
	for u := range userFeedback {
		userFeedback[u] = make(map[int]interface{})
		for _, i := range trainSet.UserFeedback[u] {
			userFeedback[u][i] = nil
		}
	}
	snapshots := SnapshotManger{}
	evalStart := time.Now()
	scores := Evaluate(bpr, valSet, trainSet, config.TopK, config.Candidates, config.Jobs, NDCG, Precision, Recall)
	evalTime := time.Since(evalStart)
	base.Logger().Debug(fmt.Sprintf("fit bpr %v/%v", 0, bpr.nEpochs),
		zap.String("eval_time", evalTime.String()),
		zap.Float32(fmt.Sprintf("NDCG@%v", config.TopK), scores[0]),
		zap.Float32(fmt.Sprintf("Precision@%v", config.TopK), scores[1]),
		zap.Float32(fmt.Sprintf("Recall@%v", config.TopK), scores[2]))
	snapshots.AddSnapshot(Score{NDCG: scores[0], Precision: scores[1], Recall: scores[2]}, bpr.UserFactor, bpr.ItemFactor)
	// Training
	for epoch := 1; epoch <= bpr.nEpochs; epoch++ {
		fitStart := time.Now()
		// Training epoch
		cost := float32(0.0)
		_ = base.Parallel(trainSet.Count(), config.Jobs, func(workerId, _ int) error {
			// Select a user
			var userIndex, ratingCount int
			for {
				userIndex = rng[workerId].Intn(trainSet.UserCount())
				ratingCount = len(trainSet.UserFeedback[userIndex])
				if ratingCount > 0 {
					break
				}
			}
			posIndex := trainSet.UserFeedback[userIndex][rng[workerId].Intn(ratingCount)]
			// Select a negative sample
			negIndex := -1
			for {
				temp := rng[workerId].Intn(trainSet.ItemCount())
				if _, exist := userFeedback[userIndex][temp]; !exist {
					negIndex = temp
					break
				}
			}
			diff := bpr.InternalPredict(userIndex, posIndex) - bpr.InternalPredict(userIndex, negIndex)
			cost += math32.Log(1 + math32.Exp(-diff))
			grad := math32.Exp(-diff) / (1.0 + math32.Exp(-diff))
			// Pairwise update
			copy(userFactor[workerId], bpr.UserFactor[userIndex])
			copy(positiveItemFactor[workerId], bpr.ItemFactor[posIndex])
			copy(negativeItemFactor[workerId], bpr.ItemFactor[negIndex])
			// Update positive item latent factor: +w_u
			floats.MulConstTo(userFactor[workerId], grad, temp[workerId])
			floats.MulConstAddTo(positiveItemFactor[workerId], -bpr.reg, temp[workerId])
			floats.MulConstAddTo(temp[workerId], bpr.lr, bpr.ItemFactor[posIndex])
			// Update negative item latent factor: -w_u
			floats.MulConstTo(userFactor[workerId], -grad, temp[workerId])
			floats.MulConstAddTo(negativeItemFactor[workerId], -bpr.reg, temp[workerId])
			floats.MulConstAddTo(temp[workerId], bpr.lr, bpr.ItemFactor[negIndex])
			// Update user latent factor: h_i-h_j
			floats.SubTo(positiveItemFactor[workerId], negativeItemFactor[workerId], temp[workerId])
			floats.MulConst(temp[workerId], grad)
			floats.MulConstAddTo(userFactor[workerId], -bpr.reg, temp[workerId])
			floats.MulConstAddTo(temp[workerId], bpr.lr, bpr.UserFactor[userIndex])
			return nil
		})
		fitTime := time.Since(fitStart)
		// Cross validation
		if epoch%config.Verbose == 0 || epoch == bpr.nEpochs {
			evalStart = time.Now()
			scores = Evaluate(bpr, valSet, trainSet, config.TopK, config.Candidates, config.Jobs, NDCG, Precision, Recall)
			evalTime = time.Since(evalStart)
			base.Logger().Debug(fmt.Sprintf("fit bpr %v/%v", epoch, bpr.nEpochs),
				zap.String("fit_time", fitTime.String()),
				zap.String("eval_time", evalTime.String()),
				zap.Float32(fmt.Sprintf("NDCG@%v", config.TopK), scores[0]),
				zap.Float32(fmt.Sprintf("Precision@%v", config.TopK), scores[1]),
				zap.Float32(fmt.Sprintf("Recall@%v", config.TopK), scores[2]))
			snapshots.AddSnapshot(Score{NDCG: scores[0], Precision: scores[1], Recall: scores[2]}, bpr.UserFactor, bpr.ItemFactor)
		}
	}
	// restore best snapshot
	bpr.UserFactor = snapshots.BestWeights[0].([][]float32)
	bpr.ItemFactor = snapshots.BestWeights[1].([][]float32)
	base.Logger().Info("fit bpr complete",
		zap.Float32(fmt.Sprintf("NDCG@%v", config.TopK), snapshots.BestScore.NDCG),
		zap.Float32(fmt.Sprintf("Precision@%v", config.TopK), snapshots.BestScore.Precision),
		zap.Float32(fmt.Sprintf("Recall@%v", config.TopK), snapshots.BestScore.Recall))
	return snapshots.BestScore
}

func (bpr *BPR) Clear() {
	bpr.UserIndex = nil
	bpr.ItemIndex = nil
	bpr.UserFactor = nil
	bpr.ItemFactor = nil
}

func (bpr *BPR) Invalid() bool {
	return bpr == nil ||
		bpr.UserIndex == nil ||
		bpr.ItemIndex == nil ||
		bpr.UserFactor == nil ||
		bpr.ItemFactor == nil
}

func (bpr *BPR) Init(trainSet *DataSet) {
	// Initialize parameters
	newUserFactor := bpr.GetRandomGenerator().NormalMatrix(trainSet.UserCount(), bpr.nFactors, bpr.initMean, bpr.initStdDev)
	newItemFactor := bpr.GetRandomGenerator().NormalMatrix(trainSet.ItemCount(), bpr.nFactors, bpr.initMean, bpr.initStdDev)
	// Relocate parameters
	if bpr.UserIndex != nil {
		for _, userId := range trainSet.UserIndex.GetNames() {
			oldIndex := bpr.UserIndex.ToNumber(userId)
			newIndex := trainSet.UserIndex.ToNumber(userId)
			if oldIndex != base.NotId {
				newUserFactor[newIndex] = bpr.UserFactor[oldIndex]
			}
		}
	}
	if bpr.ItemIndex != nil {
		for _, itemId := range trainSet.ItemIndex.GetNames() {
			oldIndex := bpr.ItemIndex.ToNumber(itemId)
			newIndex := trainSet.ItemIndex.ToNumber(itemId)
			if oldIndex != base.NotId {
				newItemFactor[newIndex] = bpr.ItemFactor[oldIndex]
			}
		}
	}
	// Initialize base
	bpr.UserFactor = newUserFactor
	bpr.ItemFactor = newItemFactor
	bpr.BaseMatrixFactorization.Init(trainSet)
}

// ALS [7] is the Weighted Regularized Matrix Factorization, which exploits
// unique properties of implicit feedback datasets. It treats the data as
// indication of positive and negative preference associated with vastly
// varying confidence levels. This leads to a factor model which is especially
// tailored for implicit feedback recommenders. Authors also proposed a
// scalable optimization procedure, which scales linearly with the data size.
// Hyper-parameters:
//   NFactors   - The number of latent factors. Default is 10.
//   NEpochs    - The number of training epochs. Default is 50.
//   InitMean   - The mean of initial latent factors. Default is 0.
//   InitStdDev - The standard deviation of initial latent factors. Default is 0.1.
//   Reg        - The strength of regularization.
type ALS struct {
	BaseMatrixFactorization
	// Model parameters
	UserFactor *mat.Dense // p_u
	ItemFactor *mat.Dense // q_i
	// Hyper parameters
	nFactors   int
	nEpochs    int
	reg        float64
	initMean   float64
	initStdDev float64
	weight     float64
}

// NewALS creates a ALS model.
func NewALS(params model.Params) *ALS {
	als := new(ALS)
	als.SetParams(params)
	return als
}

// SetParams sets hyper-parameters for the ALS model.
func (als *ALS) SetParams(params model.Params) {
	als.BaseMatrixFactorization.SetParams(params)
	als.nFactors = als.Params.GetInt(model.NFactors, 15)
	als.nEpochs = als.Params.GetInt(model.NEpochs, 50)
	als.initMean = float64(als.Params.GetFloat32(model.InitMean, 0))
	als.initStdDev = float64(als.Params.GetFloat32(model.InitStdDev, 0.1))
	als.reg = float64(als.Params.GetFloat32(model.Reg, 0.06))
	als.weight = float64(als.Params.GetFloat32(model.Alpha, 0.001))
}

func (als *ALS) GetParamsGrid() model.ParamsGrid {
	return model.ParamsGrid{
		model.NFactors:   []interface{}{8, 16, 32, 64},
		model.InitMean:   []interface{}{0},
		model.InitStdDev: []interface{}{0.001, 0.005, 0.01, 0.05, 0.1},
		model.Reg:        []interface{}{0.001, 0.005, 0.01, 0.05, 0.1},
		model.Alpha:      []interface{}{0.001, 0.005, 0.01, 0.05, 0.1},
	}
}

// Predict by the ALS model.
func (als *ALS) Predict(userId, itemId string) float32 {
	userIndex := als.UserIndex.ToNumber(userId)
	itemIndex := als.ItemIndex.ToNumber(itemId)
	if userIndex == base.NotId {
		base.Logger().Info("unknown user", zap.String("user_id", userId))
		return 0
	}
	if itemIndex == base.NotId {
		base.Logger().Info("unknown item", zap.String("item_id", itemId))
		return 0
	}
	return als.InternalPredict(userIndex, itemIndex)
}

func (als *ALS) InternalPredict(userIndex, itemIndex int) float32 {
	ret := float32(0.0)
	if itemIndex != base.NotId && userIndex != base.NotId {
		ret = float32(mat.Dot(als.UserFactor.RowView(userIndex),
			als.ItemFactor.RowView(itemIndex)))
	} else {
		base.Logger().Warn("unknown user or item")
	}
	return ret
}

// Fit the ALS model.
func (als *ALS) Fit(trainSet, valSet *DataSet, config *FitConfig) Score {
	config = config.LoadDefaultIfNil()
	base.Logger().Info("fit als",
		zap.Int("train_set_size", trainSet.Count()),
		zap.Int("test_set_size", valSet.Count()),
		zap.Any("params", als.GetParams()),
		zap.Any("config", config))
	als.Init(trainSet)
	// Create temporary matrix
	temp1 := make([]*mat.Dense, config.Jobs)
	temp2 := make([]*mat.VecDense, config.Jobs)
	a := make([]*mat.Dense, config.Jobs)
	for i := 0; i < config.Jobs; i++ {
		temp1[i] = mat.NewDense(als.nFactors, als.nFactors, nil)
		temp2[i] = mat.NewVecDense(als.nFactors, nil)
		a[i] = mat.NewDense(als.nFactors, als.nFactors, nil)
	}
	c := mat.NewDense(als.nFactors, als.nFactors, nil)
	// Create regularization matrix
	regs := make([]float64, als.nFactors)
	for i := range regs {
		regs[i] = als.reg
	}
	regI := mat.NewDiagDense(als.nFactors, regs)
	snapshots := SnapshotManger{}
	evalStart := time.Now()
	scores := Evaluate(als, valSet, trainSet, config.TopK, config.Candidates, config.Jobs, NDCG, Precision, Recall)
	evalTime := time.Since(evalStart)
	base.Logger().Debug(fmt.Sprintf("fit als %v/%v", 0, als.nEpochs),
		zap.String("eval_time", evalTime.String()),
		zap.Float32(fmt.Sprintf("NDCG@%v", config.TopK), scores[0]),
		zap.Float32(fmt.Sprintf("Precision@%v", config.TopK), scores[1]),
		zap.Float32(fmt.Sprintf("Recall@%v", config.TopK), scores[2]))
	userFactorCopy := mat.NewDense(trainSet.UserCount(), als.nFactors, nil)
	itemFactorCopy := mat.NewDense(trainSet.ItemCount(), als.nFactors, nil)
	userFactorCopy.Copy(als.UserFactor)
	itemFactorCopy.Copy(als.ItemFactor)
	snapshots.AddSnapshotNoCopy(Score{NDCG: scores[0], Precision: scores[1], Recall: scores[2]}, userFactorCopy, itemFactorCopy)
	for ep := 1; ep <= als.nEpochs; ep++ {
		fitStart := time.Now()
		// Recompute all user factors: x_u = (Y^T C^userIndex Y + \lambda reg)^{-1} Y^T C^userIndex p(userIndex)
		// Y^T Y
		c.Mul(als.ItemFactor.T(), als.ItemFactor)
		c.Scale(als.weight, c)
		err := base.Parallel(trainSet.UserCount(), config.Jobs, func(workerId, userIndex int) error {
			a[workerId].Copy(c)
			b := mat.NewVecDense(als.nFactors, nil)
			for _, itemIndex := range trainSet.UserFeedback[userIndex] {
				// Y^T (C^u-I) Y
				temp1[workerId].Outer(1, als.ItemFactor.RowView(itemIndex), als.ItemFactor.RowView(itemIndex))
				a[workerId].Add(a[workerId], temp1[workerId])
				// Y^T C^u p(u)
				temp2[workerId].ScaleVec(1+als.weight, als.ItemFactor.RowView(itemIndex))
				b.AddVec(b, temp2[workerId])
			}
			a[workerId].Add(a[workerId], regI)
			err := temp1[workerId].Inverse(a[workerId])
			temp2[workerId].MulVec(temp1[workerId], b)
			als.UserFactor.SetRow(userIndex, temp2[workerId].RawVector().Data)
			return err
		})
		if err != nil {
			base.Logger().Error("failed to inverse matrix", zap.Error(err))
		}
		// Recompute all item factors: y_i = (X^T C^i X + \lambda reg)^{-1} X^T C^i p(i)
		// X^T X
		c.Mul(als.UserFactor.T(), als.UserFactor)
		c.Scale(als.weight, c)
		err = base.Parallel(trainSet.ItemCount(), config.Jobs, func(workerId, itemIndex int) error {
			a[workerId].Copy(c)
			b := mat.NewVecDense(als.nFactors, nil)
			for _, index := range trainSet.ItemFeedback[itemIndex] {
				// X^T (C^i-I) X
				temp1[workerId].Outer(1, als.UserFactor.RowView(index), als.UserFactor.RowView(index))
				a[workerId].Add(a[workerId], temp1[workerId])
				// X^T C^i p(i)
				temp2[workerId].ScaleVec(1+als.weight, als.UserFactor.RowView(index))
				b.AddVec(b, temp2[workerId])
			}
			a[workerId].Add(a[workerId], regI)
			err = temp1[workerId].Inverse(a[workerId])
			temp2[workerId].MulVec(temp1[workerId], b)
			als.ItemFactor.SetRow(itemIndex, temp2[workerId].RawVector().Data)
			return err
		})
		if err != nil {
			base.Logger().Error("failed to inverse matrix", zap.Error(err))
		}
		fitTime := time.Since(fitStart)
		// Cross validation
		if ep%config.Verbose == 0 || ep == als.nEpochs {
			evalStart = time.Now()
			scores = Evaluate(als, valSet, trainSet, config.TopK, config.Candidates, config.Jobs, NDCG, Precision, Recall)
			evalTime = time.Since(evalStart)
			base.Logger().Debug(fmt.Sprintf("fit als %v/%v", ep, als.nEpochs),
				zap.String("fit_time", fitTime.String()),
				zap.String("eval_time", evalTime.String()),
				zap.Float32(fmt.Sprintf("NDCG@%v", config.TopK), scores[0]),
				zap.Float32(fmt.Sprintf("Precision@%v", config.TopK), scores[1]),
				zap.Float32(fmt.Sprintf("Recall@%v", config.TopK), scores[2]))
			userFactorCopy = mat.NewDense(trainSet.UserCount(), als.nFactors, nil)
			itemFactorCopy = mat.NewDense(trainSet.ItemCount(), als.nFactors, nil)
			userFactorCopy.Copy(als.UserFactor)
			itemFactorCopy.Copy(als.ItemFactor)
			snapshots.AddSnapshotNoCopy(Score{NDCG: scores[0], Precision: scores[1], Recall: scores[2]}, userFactorCopy, itemFactorCopy)
		}
	}
	// restore best snapshot
	als.UserFactor = snapshots.BestWeights[0].(*mat.Dense)
	als.ItemFactor = snapshots.BestWeights[1].(*mat.Dense)
	base.Logger().Info("fit als complete",
		zap.Float32(fmt.Sprintf("NDCG@%v", config.TopK), snapshots.BestScore.NDCG),
		zap.Float32(fmt.Sprintf("Precision@%v", config.TopK), snapshots.BestScore.Precision),
		zap.Float32(fmt.Sprintf("Recall@%v", config.TopK), snapshots.BestScore.Recall))
	return snapshots.BestScore
}

func (als *ALS) Clear() {
	als.UserIndex = nil
	als.ItemIndex = nil
	als.ItemFactor = nil
	als.UserFactor = nil
}

func (als *ALS) Invalid() bool {
	return als == nil ||
		als.ItemIndex == nil ||
		als.UserIndex == nil ||
		als.ItemFactor == nil ||
		als.UserFactor == nil
}

func (als *ALS) Init(trainSet *DataSet) {
	// Initialize
	newUserFactor := mat.NewDense(trainSet.UserCount(), als.nFactors,
		als.GetRandomGenerator().NormalVector64(trainSet.UserCount()*als.nFactors, als.initMean, als.initStdDev))
	newItemFactor := mat.NewDense(trainSet.ItemCount(), als.nFactors,
		als.GetRandomGenerator().NormalVector64(trainSet.ItemCount()*als.nFactors, als.initMean, als.initStdDev))
	// Relocate parameters
	if als.UserIndex != nil {
		for _, userId := range trainSet.UserIndex.GetNames() {
			oldIndex := als.UserIndex.ToNumber(userId)
			newIndex := trainSet.UserIndex.ToNumber(userId)
			if oldIndex != base.NotId {
				newUserFactor.SetRow(newIndex, als.UserFactor.RawRowView(oldIndex))
			}
		}
	}
	if als.ItemIndex != nil {
		for _, itemId := range trainSet.ItemIndex.GetNames() {
			oldIndex := als.ItemIndex.ToNumber(itemId)
			newIndex := trainSet.ItemIndex.ToNumber(itemId)
			if oldIndex != base.NotId {
				newItemFactor.SetRow(newIndex, als.ItemFactor.RawRowView(oldIndex))
			}
		}
	}
	// Initialize base
	als.UserFactor = newUserFactor
	als.ItemFactor = newItemFactor
	als.BaseMatrixFactorization.Init(trainSet)
}

type CCD struct {
	BaseMatrixFactorization
	// Model parameters
	UserFactor [][]float32
	ItemFactor [][]float32
	// Hyper parameters
	nFactors   int
	nEpochs    int
	reg        float32
	initMean   float32
	initStdDev float32
	weight     float32
}

// NewCCD creates a eALS model.
func NewCCD(params model.Params) *CCD {
	fast := new(CCD)
	fast.SetParams(params)
	return fast
}

// SetParams sets hyper-parameters for the ALS model.
func (ccd *CCD) SetParams(params model.Params) {
	ccd.BaseMatrixFactorization.SetParams(params)
	ccd.nFactors = ccd.Params.GetInt(model.NFactors, 15)
	ccd.nEpochs = ccd.Params.GetInt(model.NEpochs, 50)
	ccd.initMean = ccd.Params.GetFloat32(model.InitMean, 0)
	ccd.initStdDev = ccd.Params.GetFloat32(model.InitStdDev, 0.1)
	ccd.reg = ccd.Params.GetFloat32(model.Reg, 0.06)
	ccd.weight = ccd.Params.GetFloat32(model.Alpha, 0.001)
}

func (ccd *CCD) GetParamsGrid() model.ParamsGrid {
	return model.ParamsGrid{
		model.NFactors:   []interface{}{8, 16, 32, 64},
		model.InitMean:   []interface{}{0},
		model.InitStdDev: []interface{}{0.001, 0.005, 0.01, 0.05, 0.1},
		model.Reg:        []interface{}{0.001, 0.005, 0.01, 0.05, 0.1},
		model.Alpha:      []interface{}{0.001, 0.005, 0.01, 0.05, 0.1},
	}
}

// Predict by the ALS model.
func (ccd *CCD) Predict(userId, itemId string) float32 {
	userIndex := ccd.UserIndex.ToNumber(userId)
	itemIndex := ccd.ItemIndex.ToNumber(itemId)
	if userIndex == base.NotId {
		base.Logger().Info("unknown user:", zap.String("user_id", userId))
		return 0
	}
	if itemIndex == base.NotId {
		base.Logger().Info("unknown item:", zap.String("item_id", itemId))
		return 0
	}
	return ccd.InternalPredict(userIndex, itemIndex)
}

func (ccd *CCD) InternalPredict(userIndex, itemIndex int) float32 {
	ret := float32(0.0)
	if itemIndex != base.NotId && userIndex != base.NotId {
		ret = floats.Dot(ccd.UserFactor[userIndex], ccd.ItemFactor[itemIndex])
	} else {
		base.Logger().Warn("unknown user or item")
	}
	return ret
}

func (ccd *CCD) Clear() {
	ccd.UserIndex = nil
	ccd.ItemIndex = nil
	ccd.ItemFactor = nil
	ccd.UserFactor = nil
}

func (ccd *CCD) Invalid() bool {
	return ccd == nil ||
		ccd.UserIndex == nil ||
		ccd.ItemIndex == nil ||
		ccd.ItemFactor == nil ||
		ccd.UserFactor == nil
}

func (ccd *CCD) Init(trainSet *DataSet) {
	// Initialize
	newUserFactor := ccd.GetRandomGenerator().NormalMatrix(trainSet.UserCount(), ccd.nFactors, ccd.initMean, ccd.initStdDev)
	newItemFactor := ccd.GetRandomGenerator().NormalMatrix(trainSet.ItemCount(), ccd.nFactors, ccd.initMean, ccd.initStdDev)
	// Relocate parameters
	if ccd.UserIndex != nil {
		for _, userId := range trainSet.UserIndex.GetNames() {
			oldIndex := ccd.UserIndex.ToNumber(userId)
			newIndex := trainSet.UserIndex.ToNumber(userId)
			if oldIndex != base.NotId {
				newUserFactor[newIndex] = ccd.UserFactor[oldIndex]
			}
		}
	}
	if ccd.ItemIndex != nil {
		for _, itemId := range trainSet.ItemIndex.GetNames() {
			oldIndex := ccd.ItemIndex.ToNumber(itemId)
			newIndex := trainSet.ItemIndex.ToNumber(itemId)
			if oldIndex != base.NotId {
				newItemFactor[newIndex] = ccd.ItemFactor[oldIndex]
			}
		}
	}
	// Initialize base
	ccd.UserFactor = newUserFactor
	ccd.ItemFactor = newItemFactor
	ccd.BaseMatrixFactorization.Init(trainSet)
}

func (ccd *CCD) Fit(trainSet, valSet *DataSet, config *FitConfig) Score {
	config = config.LoadDefaultIfNil()
	base.Logger().Info("fit ccd",
		zap.Int("train_set_size", trainSet.Count()),
		zap.Int("test_set_size", valSet.Count()),
		zap.Any("params", ccd.GetParams()),
		zap.Any("config", config))
	ccd.Init(trainSet)
	// Create temporary matrix
	s := base.NewMatrix32(ccd.nFactors, ccd.nFactors)
	userPredictions := make([][]float32, config.Jobs)
	itemPredictions := make([][]float32, config.Jobs)
	userRes := make([][]float32, config.Jobs)
	itemRes := make([][]float32, config.Jobs)
	for i := 0; i < config.Jobs; i++ {
		userPredictions[i] = make([]float32, trainSet.ItemCount())
		itemPredictions[i] = make([]float32, trainSet.UserCount())
		userRes[i] = make([]float32, trainSet.ItemCount())
		itemRes[i] = make([]float32, trainSet.UserCount())
	}
	// evaluate initial model
	snapshots := SnapshotManger{}
	evalStart := time.Now()
	scores := Evaluate(ccd, valSet, trainSet, config.TopK, config.Candidates, config.Jobs, NDCG, Precision, Recall)
	evalTime := time.Since(evalStart)
	base.Logger().Debug(fmt.Sprintf("fit ccd %v/%v", 0, ccd.nEpochs),
		zap.String("eval_time", evalTime.String()),
		zap.Float32(fmt.Sprintf("NDCG@%v", config.TopK), scores[0]),
		zap.Float32(fmt.Sprintf("Precision@%v", config.TopK), scores[1]),
		zap.Float32(fmt.Sprintf("Recall@%v", config.TopK), scores[2]))
	snapshots.AddSnapshot(Score{NDCG: scores[0], Precision: scores[1], Recall: scores[2]}, ccd.UserFactor, ccd.ItemFactor)
	for ep := 1; ep <= ccd.nEpochs; ep++ {
		fitStart := time.Now()
		// Update user factors
		// S^q <- \sum^N_{itemIndex=1} c_i q_i q_i^T
		floats.MatZero(s)
		for i := 0; i < ccd.nFactors; i++ {
			for j := 0; j < ccd.nFactors; j++ {
				for itemIndex := 0; itemIndex < trainSet.ItemCount(); itemIndex++ {
					s[i][j] += ccd.ItemFactor[itemIndex][i] * ccd.ItemFactor[itemIndex][j]
				}
			}
		}
		_ = base.Parallel(trainSet.UserCount(), config.Jobs, func(workerId, userIndex int) error {
			userFeedback := trainSet.UserFeedback[userIndex]
			for _, i := range userFeedback {
				userPredictions[workerId][i] = ccd.InternalPredict(userIndex, i)
			}
			for f := 0; f < ccd.nFactors; f++ {
				// for itemIndex \in R_u do   \hat_{r}^f_{ui} <- \hat_{r}_{ui} - p_{uf]q_{if}
				for _, i := range userFeedback {
					userRes[workerId][i] = userPredictions[workerId][i] - ccd.UserFactor[userIndex][f]*ccd.ItemFactor[i][f]
				}
				// p_{uf} <-
				a, b, c := float32(0), float32(0), float32(0)
				for _, i := range userFeedback {
					a += (1 - (1-ccd.weight)*userRes[workerId][i]) * ccd.ItemFactor[i][f]
					c += (1 - ccd.weight) * ccd.ItemFactor[i][f] * ccd.ItemFactor[i][f]
				}
				for k := 0; k < ccd.nFactors; k++ {
					if k != f {
						b += ccd.weight * ccd.UserFactor[userIndex][k] * s[k][f]
					}
				}
				ccd.UserFactor[userIndex][f] = (a - b) / (c + ccd.weight*s[f][f] + ccd.reg)
				// for itemIndex \in R_u do   \hat_{r}_{ui} <- \hat_{r}^f_{ui} - p_{uf]q_{if}
				for _, i := range userFeedback {
					userPredictions[workerId][i] = userRes[workerId][i] + ccd.UserFactor[userIndex][f]*ccd.ItemFactor[i][f]
				}
			}
			return nil
		})
		// Update item factors
		// S^p <- P^T P
		floats.MatZero(s)
		for i := 0; i < ccd.nFactors; i++ {
			for j := 0; j < ccd.nFactors; j++ {
				for userIndex := 0; userIndex < trainSet.UserCount(); userIndex++ {
					s[i][j] += ccd.UserFactor[userIndex][i] * ccd.UserFactor[userIndex][j]
				}
			}
		}
		_ = base.Parallel(trainSet.ItemCount(), config.Jobs, func(workerId, itemIndex int) error {
			itemFeedback := trainSet.ItemFeedback[itemIndex]
			for _, u := range itemFeedback {
				itemPredictions[workerId][u] = ccd.InternalPredict(u, itemIndex)
			}
			for f := 0; f < ccd.nFactors; f++ {
				// for itemIndex \in R_u do   \hat_{r}^f_{ui} <- \hat_{r}_{ui} - p_{uf]q_{if}
				for _, u := range itemFeedback {
					itemRes[workerId][u] = itemPredictions[workerId][u] - ccd.UserFactor[u][f]*ccd.ItemFactor[itemIndex][f]
				}
				// q_{if} <-
				a, b, c := float32(0), float32(0), float32(0)
				for _, u := range itemFeedback {
					a += (1 - (1-ccd.weight)*itemRes[workerId][u]) * ccd.UserFactor[u][f]
					c += (1 - ccd.weight) * ccd.UserFactor[u][f] * ccd.UserFactor[u][f]
				}
				for k := 0; k < ccd.nFactors; k++ {
					if k != f {
						b += ccd.weight * ccd.ItemFactor[itemIndex][k] * s[k][f]
					}
				}
				ccd.ItemFactor[itemIndex][f] = (a - b) / (c + ccd.weight*s[f][f] + ccd.reg)
				// for itemIndex \in R_u do   \hat_{r}_{ui} <- \hat_{r}^f_{ui} - p_{uf]q_{if}
				for _, u := range itemFeedback {
					itemPredictions[workerId][u] = itemRes[workerId][u] + ccd.UserFactor[u][f]*ccd.ItemFactor[itemIndex][f]
				}
			}
			return nil
		})
		fitTime := time.Since(fitStart)
		// Cross validation
		if ep%config.Verbose == 0 || ep == ccd.nEpochs {
			evalStart = time.Now()
			scores = Evaluate(ccd, valSet, trainSet, config.TopK, config.Candidates, config.Jobs, NDCG, Precision, Recall)
			evalTime = time.Since(evalStart)
			base.Logger().Debug(fmt.Sprintf("fit ccd %v/%v", ep, ccd.nEpochs),
				zap.String("fit_time", fitTime.String()),
				zap.String("eval_time", evalTime.String()),
				zap.Float32(fmt.Sprintf("NDCG@%v", config.TopK), scores[0]),
				zap.Float32(fmt.Sprintf("Precision@%v", config.TopK), scores[1]),
				zap.Float32(fmt.Sprintf("Recall@%v", config.TopK), scores[2]))
			snapshots.AddSnapshot(Score{NDCG: scores[0], Precision: scores[1], Recall: scores[2]}, ccd.UserFactor, ccd.ItemFactor)
		}
	}
	// restore best snapshot
	ccd.UserFactor = snapshots.BestWeights[0].([][]float32)
	ccd.ItemFactor = snapshots.BestWeights[1].([][]float32)
	base.Logger().Info("fit ccd complete",
		zap.Float32(fmt.Sprintf("NDCG@%v", config.TopK), snapshots.BestScore.NDCG),
		zap.Float32(fmt.Sprintf("Precision@%v", config.TopK), snapshots.BestScore.Precision),
		zap.Float32(fmt.Sprintf("Recall@%v", config.TopK), snapshots.BestScore.Recall))
	return snapshots.BestScore
}

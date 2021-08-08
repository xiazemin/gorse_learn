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

package click

import (
	"bufio"
	"bytes"
	"encoding/gob"
	"fmt"
	"github.com/barkimedes/go-deepcopy"
	"time"

	"github.com/chewxy/math32"
	"github.com/zhenghaoz/gorse/base"
	"github.com/zhenghaoz/gorse/floats"
	"github.com/zhenghaoz/gorse/model"
	"go.uber.org/zap"
)

type Score struct {
	Task      FMTask
	RMSE      float32
	Precision float32
}

func (score Score) GetName() string {
	switch score.Task {
	case FMRegression:
		return "RMSE"
	case FMClassification:
		return "Precision"
	default:
		return "NaN"
	}
}

func (score Score) GetValue() float32 {
	switch score.Task {
	case FMRegression:
		return score.RMSE
	case FMClassification:
		return score.Precision
	default:
		return math32.NaN()
	}
}

func (score Score) BetterThan(s Score) bool {
	if s.Task == "" && score.Task != "" {
		return true
	} else if s.Task != "" && score.Task == "" {
		return false
	}
	if score.Task != s.Task {
		panic("task type doesn't match")
	}
	switch score.Task {
	case FMRegression:
		return score.RMSE < s.RMSE
	case FMClassification:
		return score.Precision > s.Precision
	default:
		return true
	}
}

type FitConfig struct {
	Jobs    int
	Verbose int
}

func NewFitConfig() *FitConfig {
	return &FitConfig{
		Jobs:    1,
		Verbose: 10,
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

type FactorizationMachine interface {
	model.Model
	Predict(userId, itemId string, itemLabels []string) float32
	InternalPredict(x []int) float32
	Fit(trainSet *Dataset, testSet *Dataset, config *FitConfig) Score
}

type BaseFactorizationMachine struct {
	model.BaseModel
	Index UnifiedIndex
}

func (b *BaseFactorizationMachine) Init(trainSet *Dataset) {
	b.Index = trainSet.Index
}

type FMTask string

const (
	FMClassification FMTask = "c"
	FMRegression     FMTask = "r"
)

type FM struct {
	BaseFactorizationMachine
	// Model parameters
	V         [][]float32
	W         []float32
	B         float32
	MinTarget float32
	MaxTarget float32
	Task      FMTask
	// Hyper parameters
	nFactors   int
	nEpochs    int
	lr         float32
	reg        float32
	initMean   float32
	initStdDev float32
	// Special options
	useFeature bool
}

func (fm *FM) GetParamsGrid() model.ParamsGrid {
	return model.ParamsGrid{
		model.NFactors:   []interface{}{8, 16, 32, 64, 128},
		model.Lr:         []interface{}{0.001, 0.005, 0.01, 0.05, 0.1},
		model.Reg:        []interface{}{0.001, 0.005, 0.01, 0.05, 0.1},
		model.InitMean:   []interface{}{0},
		model.InitStdDev: []interface{}{0.001, 0.005, 0.01, 0.05, 0.1},
	}
}

func NewFM(task FMTask, params model.Params) *FM {
	fm := new(FM)
	fm.Task = task
	fm.SetParams(params)
	return fm
}

func (fm *FM) SetParams(params model.Params) {
	fm.BaseFactorizationMachine.SetParams(params)
	// Setup hyper-parameters
	fm.nFactors = fm.Params.GetInt(model.NFactors, 128)
	fm.nEpochs = fm.Params.GetInt(model.NEpochs, 200)
	fm.lr = fm.Params.GetFloat32(model.Lr, 0.01)
	fm.reg = fm.Params.GetFloat32(model.Reg, 0.0)
	fm.initMean = fm.Params.GetFloat32(model.InitMean, 0)
	fm.initStdDev = fm.Params.GetFloat32(model.InitStdDev, 0.01)
	fm.useFeature = fm.Params.GetBool(model.UseFeature, true)
}

func (fm *FM) Predict(userId, itemId string, itemLabels []string) float32 {
	x := make([]int, 0)
	if userIndex := fm.Index.EncodeUser(userId); userIndex != base.NotId {
		x = append(x, userIndex)
	}
	if itemIndex := fm.Index.EncodeItem(itemId); itemIndex != base.NotId {
		x = append(x, itemIndex)
	}
	for _, itemLabel := range itemLabels {
		if itemLabelIndex := fm.Index.EncodeItemLabel(itemLabel); itemLabelIndex != base.NotId {
			x = append(x, itemLabelIndex)
		}
	}
	return fm.InternalPredict(x)
}

func (fm *FM) internalPredict(x []int) float32 {
	if !fm.useFeature {
		// The input vector must be formatted as [user_id, item_id, ...]
		x = x[:2]
	}
	// w_0
	pred := fm.B
	// \sum^n_{i=1} w_i x_i
	for _, i := range x {
		pred += fm.W[i]
	}
	// \sum^n_{i=1}\sum^n_{j=i+1} <v_i,v_j> x_i x_j
	sum := float32(0)
	for f := 0; f < fm.nFactors; f++ {
		a, b := float32(0), float32(0)
		for _, i := range x {
			// 1) \sum^n_{i=1} v^2_{i,f} x^2_i
			a += fm.V[i][f]
			// 2) \sum^n_{i=1} v^2_{i,f} x^2_i
			b += fm.V[i][f] * fm.V[i][f]
		}
		// 3) (\sum^n_{i=1} v^2_{i,f} x^2_i)^2 - \sum^n_{i=1} v^2_{i,f} x^2_i
		sum += a*a - b
	}
	pred += sum / 2
	return pred
}

func (fm *FM) InternalPredict(x []int) float32 {
	pred := fm.internalPredict(x)
	if fm.Task == FMRegression {
		if pred < fm.MinTarget {
			pred = fm.MinTarget
		} else if pred > fm.MaxTarget {
			pred = fm.MaxTarget
		}
	}
	return pred
}

func (fm *FM) Fit(trainSet, testSet *Dataset, config *FitConfig) Score {
	config = config.LoadDefaultIfNil()
	base.Logger().Info("fit FM",
		zap.Int("train_size", trainSet.Count()),
		zap.Int("test_size", testSet.Count()),
		zap.String("task", string(fm.Task)),
		zap.Any("params", fm.GetParams()),
		zap.Any("config", config))
	fm.Init(trainSet)
	temp := base.NewMatrix32(config.Jobs, fm.nFactors)
	vGrad := base.NewMatrix32(config.Jobs, fm.nFactors)

	snapshots := SnapshotManger{}
	evalStart := time.Now()
	var score Score
	switch fm.Task {
	case FMRegression:
		score = EvaluateRegression(fm, testSet)
	case FMClassification:
		score = EvaluateClassification(fm, testSet)
	default:
		base.Logger().Fatal("unknown task", zap.String("task", string(fm.Task)))
	}
	evalTime := time.Since(evalStart)
	base.Logger().Debug(fmt.Sprintf("fit fm %v/%v", 0, fm.nEpochs),
		zap.String("eval_time", evalTime.String()),
		zap.Float32(score.GetName(), score.GetValue()))
	snapshots.AddSnapshot(score, fm.V, fm.W, fm.B)

	for epoch := 1; epoch <= fm.nEpochs; epoch++ {
		for _, target := range trainSet.Target {
			fm.MinTarget = math32.Min(fm.MinTarget, target)
			fm.MaxTarget = math32.Max(fm.MaxTarget, target)
		}
		fitStart := time.Now()
		cost := float32(0)
		_ = base.BatchParallel(trainSet.Count(), config.Jobs, 128, func(workerId, beginJobId, endJobId int) error {
			for i := beginJobId; i < endJobId; i++ {
				labels, target := trainSet.Get(i)
				if !fm.useFeature {
					// The input vector must be formatted as [user_id, item_id, ...]
					labels = labels[:2]
				}
				prediction := fm.internalPredict(labels)
				var grad float32
				switch fm.Task {
				case FMRegression:
					grad = prediction - target
					cost += grad * grad / 2
				case FMClassification:
					grad = -target * (1 - 1/(1+math32.Exp(-target*prediction)))
					cost += (1 + target) * math32.Log(1+math32.Exp(-prediction)) / 2
					cost += (1 - target) * math32.Log(1+math32.Exp(prediction)) / 2
				default:
					base.Logger().Fatal("unknown task", zap.String("task", string(fm.Task)))
				}
				// \sum^n_{j=1}v_j,fx_j
				floats.Zero(temp[workerId])
				for _, j := range labels {
					floats.Add(temp[workerId], fm.V[j])
				}
				// Update w_0
				fm.B -= fm.lr * grad
				for _, i := range labels {
					// Update w_i
					fm.W[i] -= fm.lr * grad
					// Update v_{i,f}
					floats.SubTo(temp[workerId], fm.V[i], vGrad[workerId])
					floats.MulConst(vGrad[workerId], grad)
					floats.MulConstAddTo(fm.V[i], fm.reg, vGrad[workerId])
					floats.MulConstAddTo(vGrad[workerId], -fm.lr, fm.V[i])
				}
			}
			return nil
		})
		fitTime := time.Since(fitStart)
		// Cross validation
		if epoch%config.Verbose == 0 || epoch == fm.nEpochs {
			evalStart := time.Now()
			var score Score
			switch fm.Task {
			case FMRegression:
				score = EvaluateRegression(fm, testSet)
			case FMClassification:
				score = EvaluateClassification(fm, testSet)
			default:
				base.Logger().Fatal("unknown task", zap.String("task", string(fm.Task)))
			}
			evalTime := time.Since(evalStart)
			base.Logger().Debug(fmt.Sprintf("fit fm %v/%v", epoch, fm.nEpochs),
				zap.String("fit_time", fitTime.String()),
				zap.String("eval_time", evalTime.String()),
				zap.Float32("loss", cost),
				zap.Float32(score.GetName(), score.GetValue()))
			// check NaN
			if math32.IsNaN(cost) || math32.IsNaN(score.GetValue()) {
				base.Logger().Warn("model diverged", zap.Float32("lr", fm.lr))
				break
			}
			snapshots.AddSnapshot(score, fm.V, fm.W, fm.B)
		}
	}
	// restore best snapshot
	fm.V = snapshots.BestWeights[0].([][]float32)
	fm.W = snapshots.BestWeights[1].([]float32)
	fm.B = snapshots.BestWeights[2].(float32)
	base.Logger().Info("fit fm complete",
		zap.Float32(snapshots.BestScore.GetName(), snapshots.BestScore.GetValue()))
	return snapshots.BestScore
}

func (fm *FM) Clear() {
	fm.B = 0.0
	fm.V = nil
	fm.W = nil
	fm.Index = nil
}

func (fm *FM) Invalid() bool {
	return fm == nil ||
		fm.V == nil ||
		fm.W == nil ||
		fm.Index == nil
}

func (fm *FM) Init(trainSet *Dataset) {
	newV := fm.GetRandomGenerator().NormalMatrix(trainSet.Index.Len(), fm.nFactors, fm.initMean, fm.initStdDev)
	newW := make([]float32, trainSet.Index.Len())
	// Relocate parameters
	if fm.Index != nil {
		// users
		for _, userId := range trainSet.Index.GetUsers() {
			oldIndex := fm.Index.EncodeUser(userId)
			newIndex := trainSet.Index.EncodeUser(userId)
			if oldIndex != base.NotId {
				newW[newIndex] = fm.W[oldIndex]
				newV[newIndex] = fm.V[oldIndex]
			}
		}
		// items
		for _, itemId := range trainSet.Index.GetItems() {
			oldIndex := fm.Index.EncodeItem(itemId)
			newIndex := trainSet.Index.EncodeItem(itemId)
			if oldIndex != base.NotId {
				newW[newIndex] = fm.W[oldIndex]
				newV[newIndex] = fm.V[oldIndex]
			}
		}
		// labels
		for _, label := range trainSet.Index.GetContextLabels() {
			oldIndex := fm.Index.EncodeContextLabel(label)
			newIndex := trainSet.Index.EncodeContextLabel(label)
			if oldIndex != base.NotId {
				newW[newIndex] = fm.W[oldIndex]
				newV[newIndex] = fm.V[oldIndex]
			}
		}
	}
	fm.MinTarget = math32.Inf(1)
	fm.MaxTarget = math32.Inf(-1)
	fm.V = newV
	fm.W = newW
	fm.BaseFactorizationMachine.Init(trainSet)
}

func EncodeModel(m FactorizationMachine) ([]byte, error) {
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

func DecodeModel(buf []byte) (FactorizationMachine, error) {
	reader := bytes.NewReader(buf)
	decoder := gob.NewDecoder(reader)
	var fm FM
	if err := decoder.Decode(&fm); err != nil {
		return nil, err
	}
	return &fm, nil
}

// Clone a model with deep copy.
func Clone(m FactorizationMachine) FactorizationMachine {
	if temp, err := deepcopy.Anything(m); err != nil {
		panic(err)
	} else {
		copied := temp.(FactorizationMachine)
		copied.SetParams(copied.GetParams())
		return copied
	}
}

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
package base

import (
	"github.com/scylladb/go-set"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestParallel(t *testing.T) {
	a := make([]int, 10000)
	for i := range a {
		a[i] = i
	}
	b := make([]int, len(a))
	workerIds := make([]int, len(a))
	// multiple threads
	_ = Parallel(len(a), 4, func(workerId, jobId int) error {
		b[jobId] = a[jobId]
		workerIds[jobId] = workerId
		time.Sleep(time.Microsecond)
		return nil
	})
	workersSet := set.NewIntSet(workerIds...)
	assert.Equal(t, a, b)
	assert.GreaterOrEqual(t, 4, workersSet.Size())
	assert.Less(t, 1, workersSet.Size())
	// single thread
	_ = Parallel(len(a), 1, func(workerId, jobId int) error {
		b[jobId] = a[jobId]
		workerIds[jobId] = workerId
		return nil
	})
	workersSet = set.NewIntSet(workerIds...)
	assert.Equal(t, a, b)
	assert.Equal(t, 1, workersSet.Size())
}

func TestBatchParallel(t *testing.T) {
	a := make([]int, 10000)
	for i := range a {
		a[i] = i
	}
	b := make([]int, len(a))
	workerIds := make([]int, len(a))
	// multiple threads
	_ = BatchParallel(len(a), 4, 10, func(workerId, beginJobId, endJobId int) error {
		for jobId := beginJobId; jobId < endJobId; jobId++ {
			b[jobId] = a[jobId]
			workerIds[jobId] = workerId
		}
		time.Sleep(time.Microsecond)
		return nil
	})
	workersSet := set.NewIntSet(workerIds...)
	assert.Equal(t, a, b)
	assert.GreaterOrEqual(t, 4, workersSet.Size())
	assert.Less(t, 1, workersSet.Size())
	// single thread
	_ = Parallel(len(a), 1, func(workerId, jobId int) error {
		b[jobId] = a[jobId]
		workerIds[jobId] = workerId
		return nil
	})
	workersSet = set.NewIntSet(workerIds...)
	assert.Equal(t, a, b)
	assert.Equal(t, 1, workersSet.Size())
}

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
package cache

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func testMeta(t *testing.T, db Database) {
	// Set meta string
	err := db.SetString("meta", "1", "2")
	assert.Nil(t, err)
	// Get meta string
	value, err := db.GetString("meta", "1")
	assert.Nil(t, err)
	assert.Equal(t, "2", value)
	// Get meta not existed
	value, err = db.GetString("meta", "NULL")
	assert.ErrorIs(t, err, ErrObjectNotExist)
	assert.Equal(t, "", value)
	// Set meta int
	err = db.SetInt("meta", "1", 2)
	assert.Nil(t, err)
	// Get meta int
	valInt, err := db.GetInt("meta", "1")
	assert.Nil(t, err)
	assert.Equal(t, 2, valInt)
	// increase meta int
	err = db.IncrInt("meta", "1")
	assert.Nil(t, err)
	valInt, err = db.GetInt("meta", "1")
	assert.Nil(t, err)
	assert.Equal(t, 3, valInt)
	// set meta time
	err = db.SetTime("meta", "1", time.Date(1996, 4, 8, 0, 0, 0, 0, time.UTC))
	assert.Nil(t, err)
	// get meta time
	valTime, err := db.GetTime("meta", "1")
	assert.Nil(t, err)
	assert.Equal(t, 1996, valTime.Year())
	assert.Equal(t, time.Month(4), valTime.Month())
	assert.Equal(t, 8, valTime.Day())
}

func testScores(t *testing.T, db Database) {
	// Put items
	items := []ScoredItem{
		{"0", 0},
		{"1", 1.1},
		{"2", 1.2},
		{"3", 1.3},
		{"4", 1.4},
	}
	err := db.SetScores("list", "0", items)
	assert.Nil(t, err)
	// Get items
	totalItems, err := db.GetScores("list", "0", 0, -1)
	assert.Nil(t, err)
	assert.Equal(t, items, totalItems)
	// Get n items
	headItems, err := db.GetScores("list", "0", 0, 2)
	assert.Nil(t, err)
	assert.Equal(t, items[:3], headItems)
	// Get n items with offset
	offsetItems, err := db.GetScores("list", "0", 1, 3)
	assert.Nil(t, err)
	assert.Equal(t, items[1:4], offsetItems)
	// Get empty
	noItems, err := db.GetScores("list", "1", 0, 0)
	assert.Nil(t, err)
	assert.Equal(t, 0, len(noItems))
	// test overwrite
	overwriteItems := []ScoredItem{
		{"10", 10.0},
		{"11", 10.1},
		{"12", 10.2},
		{"13", 10.3},
		{"14", 10.4},
	}
	err = db.SetScores("list", "0", overwriteItems)
	assert.Nil(t, err)
	totalItems, err = db.GetScores("list", "0", 0, -1)
	assert.Nil(t, err)
	assert.Equal(t, overwriteItems, totalItems)
}

func testList(t *testing.T, db Database) {
	// append
	items := []string{"0", "1", "2", "3", "4"}
	err := db.AppendList("list", "0", items...)
	assert.Nil(t, err)
	totalItems, err := db.GetList("list", "0")
	assert.Nil(t, err)
	assert.Equal(t, items, totalItems)
	// append
	appendItems := []string{"10", "11", "12", "13", "14"}
	err = db.AppendList("list", "0", appendItems...)
	assert.Nil(t, err)
	totalItems, err = db.GetList("list", "0")
	assert.Nil(t, err)
	assert.Equal(t, append(items, appendItems...), totalItems)
	// clear
	err = db.ClearList("list", "0")
	assert.Nil(t, err)
	totalItems, err = db.GetList("list", "0")
	assert.Nil(t, err)
	assert.Empty(t, totalItems)
}

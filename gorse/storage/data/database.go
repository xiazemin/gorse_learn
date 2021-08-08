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

package data

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/go-redis/redis/v8"
	"github.com/pkg/errors"
	"github.com/zhenghaoz/gorse/base"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/x/mongo/driver/connstring"
	"strings"
	"time"
)

var (
	ErrUserNotExist = errors.New("user not exist")
	ErrItemNotExist = errors.New("item not exist")
	ErrUnsupported  = fmt.Errorf("unsupported interface")
)

// Item stores meta data about item.
type Item struct {
	ItemId    string
	Timestamp time.Time
	Labels    []string
	Comment   string
}

// User stores meta data about user.
type User struct {
	UserId    string
	Labels    []string
	Subscribe []string
	Comment   string
}

// FeedbackKey identifies feedback.
type FeedbackKey struct {
	FeedbackType string
	UserId       string
	ItemId       string
}

// Feedback stores feedback.
type Feedback struct {
	FeedbackKey
	Timestamp time.Time
	Comment   string
}

// Measurement stores a statistical value.
type Measurement struct {
	Name      string
	Timestamp time.Time
	Value     float32
	Comment   string
}

type Database interface {
	Init() error
	Close() error
	InsertItem(item Item) error
	BatchInsertItem(items []Item) error
	DeleteItem(itemId string) error
	GetItem(itemId string) (Item, error)
	GetItems(cursor string, n int, timeLimit *time.Time) (string, []Item, error)
	GetItemFeedback(itemId string, feedbackTypes ...string) ([]Feedback, error)
	InsertUser(user User) error
	DeleteUser(userId string) error
	GetUser(userId string) (User, error)
	GetUsers(cursor string, n int) (string, []User, error)
	GetUserFeedback(userId string, feedbackTypes ...string) ([]Feedback, error)
	GetUserItemFeedback(userId, itemId string, feedbackTypes ...string) ([]Feedback, error)
	DeleteUserItemFeedback(userId, itemId string, feedbackTypes ...string) (int, error)
	InsertFeedback(feedback Feedback, insertUser, insertItem bool) error
	BatchInsertFeedback(feedback []Feedback, insertUser, insertItem bool) error
	GetFeedback(cursor string, n int, timeLimit *time.Time, feedbackTypes ...string) (string, []Feedback, error)
	InsertMeasurement(measurement Measurement) error
	GetMeasurements(name string, n int) ([]Measurement, error)
	GetClickThroughRate(date time.Time, positiveTypes []string, readType string) (float64, error)
	CountActiveUsers(date time.Time) (int, error)
}

const mySQLPrefix = "mysql://"
const mongoPredix = "mongodb://"
const redisPrefix = "redis://"

// Open a connection to a database.
func Open(path string) (Database, error) {
	var err error
	if strings.HasPrefix(path, mySQLPrefix) {
		name := path[len(mySQLPrefix):]
		database := new(SQLDatabase)
		if database.db, err = sql.Open("mysql", name); err != nil {
			return nil, err
		}
		return database, nil
	} else if strings.HasPrefix(path, mongoPredix) {
		// connect to database
		database := new(MongoDB)
		if database.client, err = mongo.Connect(context.Background(), options.Client().ApplyURI(path)); err != nil {
			return nil, err
		}
		// parse DSN and extract database name
		if cs, err := connstring.ParseAndValidate(path); err != nil {
			return nil, err
		} else {
			database.dbName = cs.Database
		}
		return database, nil
	} else if strings.HasPrefix(path, redisPrefix) {
		addr := path[len(redisPrefix):]
		database := new(Redis)
		database.client = redis.NewClient(&redis.Options{Addr: addr})
		base.Logger().Warn("redis is used for testing only")
		return database, nil
	}
	return nil, errors.Errorf("Unknown database: %s", path)
}

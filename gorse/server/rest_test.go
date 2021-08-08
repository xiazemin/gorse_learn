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
package server

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/emicklei/go-restful/v3"
	"github.com/steinfletcher/apitest"
	"github.com/stretchr/testify/assert"
	"github.com/zhenghaoz/gorse/config"
	"github.com/zhenghaoz/gorse/storage/cache"
	"github.com/zhenghaoz/gorse/storage/data"
)

const apiKey = "test_api_key"

type mockServer struct {
	dataStoreServer  *miniredis.Miniredis
	cacheStoreServer *miniredis.Miniredis
	handler          *restful.Container
	RestServer
}

func newMockServer(t *testing.T) *mockServer {
	s := new(mockServer)
	// create mock redis server
	var err error
	s.dataStoreServer, err = miniredis.Run()
	assert.Nil(t, err)
	s.cacheStoreServer, err = miniredis.Run()
	assert.Nil(t, err)
	// open database
	s.DataClient, err = data.Open("redis://" + s.dataStoreServer.Addr())
	assert.Nil(t, err)
	s.CacheClient, err = cache.Open("redis://" + s.cacheStoreServer.Addr())
	assert.Nil(t, err)
	// configuration
	s.GorseConfig = (*config.Config)(nil).LoadDefaultIfNil()
	s.GorseConfig.Server.APIKey = apiKey
	s.WebService = new(restful.WebService)
	s.CreateWebService()
	// create handler
	s.handler = restful.NewContainer()
	s.handler.Add(s.WebService)
	return s
}

func (s *mockServer) Close(t *testing.T) {
	err := s.DataClient.Close()
	assert.Nil(t, err)
	err = s.CacheClient.Close()
	assert.Nil(t, err)
	s.dataStoreServer.Close()
	s.cacheStoreServer.Close()
}

func marshal(t *testing.T, v interface{}) string {
	s, err := json.Marshal(v)
	assert.Nil(t, err)
	return string(s)
}

func TestServer_Users(t *testing.T) {
	s := newMockServer(t)
	defer s.Close(t)
	users := []data.User{
		{UserId: "0"},
		{UserId: "1"},
		{UserId: "2"},
		{UserId: "3"},
		{UserId: "4"},
	}
	apitest.New().
		Handler(s.handler).
		Post("/api/user").
		Header("X-API-Key", apiKey).
		JSON(users[0]).
		Expect(t).
		Status(http.StatusOK).
		Body(`{"RowAffected":1}`).
		End()
	apitest.New().
		Handler(s.handler).
		Get("/api/user/0").
		Header("X-API-Key", apiKey).
		Expect(t).
		Status(http.StatusOK).
		Body(marshal(t, users[0])).
		End()
	apitest.New().
		Handler(s.handler).
		Post("/api/users").
		Header("X-API-Key", apiKey).
		JSON(users[1:]).
		Expect(t).
		Status(http.StatusOK).
		Body(`{"RowAffected":4}`).
		End()
	apitest.New().
		Handler(s.handler).
		Get("/api/users").
		Header("X-API-Key", apiKey).
		QueryParams(map[string]string{
			"cursor": "",
			"n":      "100",
		}).
		Expect(t).
		Status(http.StatusOK).
		Body(marshal(t, UserIterator{
			Cursor: "",
			Users:  users,
		})).
		End()
	apitest.New().
		Handler(s.handler).
		Delete("/api/user/0").
		Header("X-API-Key", apiKey).
		Expect(t).
		Status(http.StatusOK).
		Body(`{"RowAffected": 1}`).
		End()
	apitest.New().
		Handler(s.handler).
		Get("/api/user/0").
		Header("X-API-Key", apiKey).
		Expect(t).
		Status(http.StatusNotFound).
		End()
}

func TestServer_Items(t *testing.T) {
	s := newMockServer(t)
	defer s.Close(t)
	// Items
	items := []data.Item{
		{
			ItemId:    "0",
			Timestamp: time.Date(1996, 3, 15, 0, 0, 0, 0, time.UTC),
			Labels:    []string{"a"},
			Comment:   "comment_0",
		},
		{
			ItemId:    "2",
			Timestamp: time.Date(1996, 3, 15, 0, 0, 0, 0, time.UTC),
			Labels:    []string{"a"},
			Comment:   "comment_2",
		},
		{
			ItemId:    "4",
			Timestamp: time.Date(1996, 3, 15, 0, 0, 0, 0, time.UTC),
			Labels:    []string{"a", "b"},
			Comment:   "comment_4",
		},
		{
			ItemId:    "6",
			Timestamp: time.Date(1996, 3, 15, 0, 0, 0, 0, time.UTC),
			Labels:    []string{"b"},
			Comment:   "comment_6",
		},
		{
			ItemId:    "8",
			Timestamp: time.Date(1996, 3, 15, 0, 0, 0, 0, time.UTC),
			Labels:    []string{"b"},
			Comment:   "comment_8",
		},
	}
	// insert items
	apitest.New().
		Handler(s.handler).
		Post("/api/item").
		Header("X-API-Key", apiKey).
		JSON(items[0]).
		Expect(t).
		Status(http.StatusOK).
		Body(`{"RowAffected": 1}`).
		End()
	// batch insert items
	apitest.New().
		Handler(s.handler).
		Post("/api/items").
		Header("X-API-Key", apiKey).
		JSON(items[1:]).
		Expect(t).
		Status(http.StatusOK).
		Body(`{"RowAffected": 4}`).
		End()
	//get items
	apitest.New().
		Handler(s.handler).
		Get("/api/items").
		Header("X-API-Key", apiKey).
		QueryParams(map[string]string{
			"cursor": "",
			"n":      "100",
		}).
		Expect(t).
		Status(http.StatusOK).
		Body(marshal(t, ItemIterator{
			Cursor: "",
			Items:  items,
		})).
		End()
	// delete item
	apitest.New().
		Handler(s.handler).
		Delete("/api/item/0").
		Header("X-API-Key", apiKey).
		Expect(t).
		Status(http.StatusOK).
		Body(`{"RowAffected": 1}`).
		End()
	// get item
	apitest.New().
		Handler(s.handler).
		Get("/api/item/0").
		Header("X-API-Key", apiKey).
		Expect(t).
		Status(http.StatusNotFound).
		End()
}

func TestServer_Feedback(t *testing.T) {
	s := newMockServer(t)
	defer s.Close(t)
	// Insert ret
	feedback := []data.Feedback{
		{FeedbackKey: data.FeedbackKey{FeedbackType: "click", UserId: "0", ItemId: "0"}},
		{FeedbackKey: data.FeedbackKey{FeedbackType: "click", UserId: "1", ItemId: "2"}},
		{FeedbackKey: data.FeedbackKey{FeedbackType: "click", UserId: "2", ItemId: "4"}},
		{FeedbackKey: data.FeedbackKey{FeedbackType: "click", UserId: "3", ItemId: "6"}},
		{FeedbackKey: data.FeedbackKey{FeedbackType: "click", UserId: "4", ItemId: "8"}},
	}
	//BatchInsertFeedback
	apitest.New().
		Handler(s.handler).
		Post("/api/feedback").
		Header("X-API-Key", apiKey).
		JSON(feedback).
		Expect(t).
		Status(http.StatusOK).
		Body(`{"RowAffected": 5}`).
		End()
	//Get Feedback
	apitest.New().
		Handler(s.handler).
		Get("/api/feedback").
		Header("X-API-Key", apiKey).
		QueryParams(map[string]string{
			"cursor": "",
			"n":      "100",
		}).
		Expect(t).
		Body(marshal(t, FeedbackIterator{
			Cursor:   "",
			Feedback: feedback,
		})).
		Status(http.StatusOK).
		End()
	//Get Items
	apitest.New().
		Handler(s.handler).
		Get("/api/items").
		Header("X-API-Key", apiKey).
		Expect(t).
		Status(http.StatusOK).
		Body(marshal(t, ItemIterator{
			Cursor: "",
			Items: []data.Item{
				{ItemId: "0"},
				{ItemId: "2"},
				{ItemId: "4"},
				{ItemId: "6"},
				{ItemId: "8"},
			},
		})).
		End()
	apitest.New().
		Handler(s.handler).
		Get("/api/users").
		Header("X-API-Key", apiKey).
		Expect(t).
		Status(http.StatusOK).
		Body(marshal(t, UserIterator{
			Cursor: "",
			Users: []data.User{
				{UserId: "0"},
				{UserId: "1"},
				{UserId: "2"},
				{UserId: "3"},
				{UserId: "4"}},
		})).
		End()
	apitest.New().
		Handler(s.handler).
		Get("/api/user/2/feedback/click").
		Header("X-API-Key", apiKey).
		Expect(t).
		Status(http.StatusOK).
		Body(`[{"FeedbackType":"click", "UserId": "2", "ItemId": "4", "Timestamp":"0001-01-01T00:00:00Z","Comment":""}]`).
		End()
	apitest.New().
		Handler(s.handler).
		Get("/api/item/4/feedback/click").
		Header("X-API-Key", apiKey).
		Expect(t).
		Status(http.StatusOK).
		Body(`[{"FeedbackType":"click", "UserId": "2", "ItemId": "4", "Timestamp":"0001-01-01T00:00:00Z","Comment":""}]`).
		End()
}

func TestServer_List(t *testing.T) {
	s := newMockServer(t)
	defer s.Close(t)
	type ListOperator struct {
		Prefix string
		Label  string
		Get    string
	}
	operators := []ListOperator{
		{cache.RecommendItems, "0", "/api/intermediate/recommend/0"},
		//{cache.SubscribeItems, "0", "/subscribe/0"},
		{cache.LatestItems, "", "/api/latest/"},
		{cache.LatestItems, "0", "/api/latest/0"},
		{cache.PopularItems, "", "/api/popular/"},
		{cache.PopularItems, "0", "/api/popular/0"},
		{cache.SimilarItems, "0", "/api/neighbors/0"},
	}

	for _, operator := range operators {
		t.Logf("test RESTful API: %v", operator.Get)
		// Put items
		items := []cache.ScoredItem{
			{"0", 100},
			{"1", 99},
			{"2", 98},
			{"3", 97},
			{"4", 96},
		}
		err := s.CacheClient.SetScores(operator.Prefix, operator.Label, items)
		assert.Nil(t, err)
		apitest.New().
			Handler(s.handler).
			Get(operator.Get).
			Header("X-API-Key", apiKey).
			Expect(t).
			Status(http.StatusOK).
			Body(marshal(t, items)).
			End()
		apitest.New().
			Handler(s.handler).
			Get(operator.Get).
			Header("X-API-Key", apiKey).
			QueryParams(map[string]string{
				"offset": "0",
				"n":      "3"}).
			Expect(t).
			Status(http.StatusOK).
			Body(marshal(t, []cache.ScoredItem{items[0], items[1], items[2]})).
			End()
		apitest.New().
			Handler(s.handler).
			Get(operator.Get).
			Header("X-API-Key", apiKey).
			QueryParams(map[string]string{
				"offset": "1",
				"n":      "3"}).
			Expect(t).
			Status(http.StatusOK).
			Body(marshal(t, []cache.ScoredItem{items[1], items[2], items[3]})).
			End()
		apitest.New().
			Handler(s.handler).
			Get(operator.Get).
			Header("X-API-Key", apiKey).
			QueryParams(map[string]string{
				"offset": "0",
				"n":      "0"}).
			Expect(t).
			Status(http.StatusOK).
			Body(marshal(t, items)).
			End()
	}
}

func TestServer_DeleteFeedback(t *testing.T) {
	s := newMockServer(t)
	defer s.Close(t)
	// Insert feedback
	feedback := []data.Feedback{
		{FeedbackKey: data.FeedbackKey{FeedbackType: "type1", UserId: "2", ItemId: "3"}},
		{FeedbackKey: data.FeedbackKey{FeedbackType: "type2", UserId: "2", ItemId: "3"}},
		{FeedbackKey: data.FeedbackKey{FeedbackType: "type3", UserId: "2", ItemId: "3"}},
		{FeedbackKey: data.FeedbackKey{FeedbackType: "type1", UserId: "1", ItemId: "6"}},
		{FeedbackKey: data.FeedbackKey{FeedbackType: "type1", UserId: "4", ItemId: "8"}},
	}
	apitest.New().
		Handler(s.handler).
		Post("/api/feedback").
		Header("X-API-Key", apiKey).
		JSON(feedback).
		Expect(t).
		Status(http.StatusOK).
		Body(`{"RowAffected": 5}`).
		End()
	// Get Feedback
	apitest.New().
		Handler(s.handler).
		Get("/api/feedback/2/3").
		Header("X-API-Key", apiKey).
		Expect(t).
		Body(marshal(t, []data.Feedback{feedback[0], feedback[1], feedback[2]})).
		Status(http.StatusOK).
		End()
	// Get typed feedback
	apitest.New().
		Handler(s.handler).
		Get("/api/feedback/type2/2/3").
		Header("X-API-Key", apiKey).
		Expect(t).
		Status(http.StatusOK).
		Body(marshal(t, feedback[1])).
		End()
	// delete feedback
	apitest.New().
		Handler(s.handler).
		Delete("/api/feedback/2/3").
		Header("X-API-Key", apiKey).
		Expect(t).
		Body(`{"RowAffected": 3}`).
		Status(http.StatusOK).
		End()
	// delete typed feedback
	apitest.New().
		Handler(s.handler).
		Delete("/api/feedback/type1/4/8").
		Header("X-API-Key", apiKey).
		Expect(t).
		Status(http.StatusOK).
		Body(`{"RowAffected": 1}`).
		End()
}

func TestServer_Measurement(t *testing.T) {
	s := newMockServer(t)
	defer s.Close(t)
	measurements := []data.Measurement{
		{"Test_NDCG", time.Date(2000, 1, 1, 1, 1, 1, 0, time.UTC), 0, "a"},
		{"Test_NDCG", time.Date(2001, 1, 1, 1, 1, 1, 0, time.UTC), 1, "b"},
		{"Test_NDCG", time.Date(2002, 1, 1, 1, 1, 1, 0, time.UTC), 2, "c"},
		{"Test_NDCG", time.Date(2003, 1, 1, 1, 1, 1, 0, time.UTC), 3, "d"},
		{"Test_NDCG", time.Date(2004, 1, 1, 1, 1, 1, 0, time.UTC), 4, "e"},
		{"Test_Recall", time.Date(2000, 1, 1, 1, 1, 1, 0, time.UTC), 1, "f"},
	}
	for _, measurement := range measurements {
		err := s.DataClient.InsertMeasurement(measurement)
		assert.Nil(t, err)
	}
	apitest.New().
		Handler(s.handler).
		Get("/api/measurements/Test_NDCG").
		Query("n", "3").
		Header("X-API-Key", apiKey).
		Expect(t).
		Status(http.StatusOK).
		Body(marshal(t, []data.Measurement{
			measurements[4],
			measurements[3],
			measurements[2],
		})).
		End()
}

func TestServer_GetRecommends(t *testing.T) {
	s := newMockServer(t)
	defer s.Close(t)
	// insert recommendation
	err := s.CacheClient.SetScores(cache.RecommendItems, "0",
		[]cache.ScoredItem{
			{"1", 99},
			{"2", 98},
			{"3", 97},
			{"4", 96},
			{"5", 95},
			{"6", 94},
			{"7", 93},
			{"8", 92},
		})
	assert.Nil(t, err)
	// insert feedback
	feedback := []data.Feedback{
		{FeedbackKey: data.FeedbackKey{FeedbackType: "a", UserId: "0", ItemId: "2"}},
		{FeedbackKey: data.FeedbackKey{FeedbackType: "a", UserId: "0", ItemId: "4"}},
	}
	apitest.New().
		Handler(s.handler).
		Post("/api/feedback").
		Header("X-API-Key", apiKey).
		JSON(feedback).
		Expect(t).
		Status(http.StatusOK).
		Body(`{"RowAffected": 2}`).
		End()
	apitest.New().
		Handler(s.handler).
		Get("/api/recommend/0").
		Header("X-API-Key", apiKey).
		QueryParams(map[string]string{
			"n": "3",
		}).
		Expect(t).
		Status(http.StatusOK).
		Body(marshal(t, []string{"1", "3", "5"})).
		End()
	apitest.New().
		Handler(s.handler).
		Get("/api/recommend/0").
		Header("X-API-Key", apiKey).
		QueryParams(map[string]string{
			"n":          "3",
			"write-back": "read",
		}).
		Expect(t).
		Status(http.StatusOK).
		Body(marshal(t, []string{"1", "3", "5"})).
		End()
	apitest.New().
		Handler(s.handler).
		Get("/api/recommend/0").
		Header("X-API-Key", apiKey).
		QueryParams(map[string]string{
			"n": "3",
		}).
		Expect(t).
		Status(http.StatusOK).
		Body(marshal(t, []string{"6", "7", "8"})).
		End()
}

func TestServer_GetRecommends_Fallback_Similar(t *testing.T) {
	s := newMockServer(t)
	defer s.Close(t)
	// insert recommendation
	err := s.CacheClient.SetScores(cache.RecommendItems, "0",
		[]cache.ScoredItem{{"1", 99}, {"2", 98}, {"3", 97}, {"4", 96}})
	assert.Nil(t, err)
	// insert feedback
	feedback := []data.Feedback{
		{FeedbackKey: data.FeedbackKey{FeedbackType: "a", UserId: "0", ItemId: "1"}},
		{FeedbackKey: data.FeedbackKey{FeedbackType: "a", UserId: "0", ItemId: "2"}},
		{FeedbackKey: data.FeedbackKey{FeedbackType: "a", UserId: "0", ItemId: "3"}},
		{FeedbackKey: data.FeedbackKey{FeedbackType: "a", UserId: "0", ItemId: "4"}},
	}
	apitest.New().
		Handler(s.handler).
		Post("/api/feedback").
		Header("X-API-Key", apiKey).
		JSON(feedback).
		Expect(t).
		Status(http.StatusOK).
		Body(`{"RowAffected": 4}`).
		End()
	// insert similar items
	err = s.CacheClient.SetScores(cache.SimilarItems, "1", []cache.ScoredItem{
		{"2", 100000},
		{"9", 1},
	})
	assert.Nil(t, err)
	err = s.CacheClient.SetScores(cache.SimilarItems, "2", []cache.ScoredItem{
		{"3", 100000},
		{"8", 1},
		{"9", 1},
	})
	assert.Nil(t, err)
	err = s.CacheClient.SetScores(cache.SimilarItems, "3", []cache.ScoredItem{
		{"4", 100000},
		{"7", 1},
		{"8", 1},
		{"9", 1},
	})
	assert.Nil(t, err)
	err = s.CacheClient.SetScores(cache.SimilarItems, "4", []cache.ScoredItem{
		{"1", 100000},
		{"6", 1},
		{"7", 1},
		{"8", 1},
		{"9", 1},
	})
	assert.Nil(t, err)
	// test fallback
	s.GorseConfig.Recommend.FallbackRecommend = "popular"
	apitest.New().
		Handler(s.handler).
		Get("/api/recommend/0").
		Header("X-API-Key", apiKey).
		QueryParams(map[string]string{
			"n": "3",
		}).
		Expect(t).
		Status(http.StatusOK).
		Body(marshal(t, []string{"9", "8", "7"})).
		End()
}

func TestServer_GetRecommends_Fallback_NonPersonalized(t *testing.T) {
	s := newMockServer(t)
	defer s.Close(t)
	// insert recommendation
	err := s.CacheClient.SetScores(cache.RecommendItems, "0",
		[]cache.ScoredItem{{"1", 99}, {"2", 98}, {"3", 97}, {"4", 96}})
	assert.Nil(t, err)
	// insert latest
	err = s.CacheClient.SetScores(cache.LatestItems, "",
		[]cache.ScoredItem{{"5", 95}, {"6", 94}, {"7", 93}, {"8", 92}})
	assert.Nil(t, err)
	// insert popular
	err = s.CacheClient.SetScores(cache.PopularItems, "",
		[]cache.ScoredItem{{"9", 91}, {"10", 90}, {"11", 89}, {"12", 88}})
	assert.Nil(t, err)
	// test popular fallback
	s.GorseConfig.Recommend.FallbackRecommend = "popular"
	apitest.New().
		Handler(s.handler).
		Get("/api/recommend/0").
		Header("X-API-Key", apiKey).
		QueryParams(map[string]string{
			"n": "8",
		}).
		Expect(t).
		Status(http.StatusOK).
		Body(marshal(t, []string{"1", "2", "3", "4", "9", "10", "11", "12"})).
		End()
	// test latest fallback
	s.GorseConfig.Recommend.FallbackRecommend = "latest"
	apitest.New().
		Handler(s.handler).
		Get("/api/recommend/0").
		Header("X-API-Key", apiKey).
		QueryParams(map[string]string{
			"n": "8",
		}).
		Expect(t).
		Status(http.StatusOK).
		Body(marshal(t, []string{"1", "2", "3", "4", "5", "6", "7", "8"})).
		End()
	// test wrong fallback
	s.GorseConfig.Recommend.FallbackRecommend = ""
	apitest.New().
		Handler(s.handler).
		Get("/api/recommend/0").
		Header("X-API-Key", apiKey).
		QueryParams(map[string]string{
			"n": "8",
		}).
		Expect(t).
		Status(http.StatusInternalServerError).
		End()
}

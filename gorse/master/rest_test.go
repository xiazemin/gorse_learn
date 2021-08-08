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

package master

import (
	"bytes"
	"encoding/json"
	"github.com/alicebob/miniredis/v2"
	"github.com/emicklei/go-restful/v3"
	"github.com/steinfletcher/apitest"
	"github.com/stretchr/testify/assert"
	"github.com/zhenghaoz/gorse/config"
	"github.com/zhenghaoz/gorse/model/click"
	"github.com/zhenghaoz/gorse/model/ranking"
	"github.com/zhenghaoz/gorse/server"
	"github.com/zhenghaoz/gorse/storage/cache"
	"github.com/zhenghaoz/gorse/storage/data"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

type mockServer struct {
	dataStoreServer  *miniredis.Miniredis
	cacheStoreServer *miniredis.Miniredis
	handler          *restful.Container
	Master
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
	// create server
	s.GorseConfig = (*config.Config)(nil).LoadDefaultIfNil()
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

func TestMaster_ExportItems(t *testing.T) {
	s := newMockServer(t)
	defer s.Close(t)
	// insert items
	items := []data.Item{
		{"1", time.Date(2020, 1, 1, 1, 1, 1, 1, time.UTC), []string{"a", "b"}, "o,n,e"},
		{"2", time.Date(2021, 1, 1, 1, 1, 1, 1, time.UTC), []string{"b", "c"}, "t\r\nw\r\no"},
		{"3", time.Date(2022, 1, 1, 1, 1, 1, 1, time.UTC), []string{"c", "d"}, "\"three\""},
	}
	err := s.DataClient.BatchInsertItem(items)
	assert.Nil(t, err)
	// send request
	req := httptest.NewRequest("GET", "https://example.com/", nil)
	w := httptest.NewRecorder()
	s.importExportItems(w, req)
	assert.Equal(t, http.StatusOK, w.Result().StatusCode)
	assert.Equal(t, "text/csv", w.Header().Get("Content-Type"))
	assert.Equal(t, "attachment;filename=items.csv", w.Header().Get("Content-Disposition"))
	assert.Equal(t, "item_id,time_stamp,labels,description\r\n"+
		"1,2020-01-01 01:01:01.000000001 +0000 UTC,a|b,\"o,n,e\"\r\n"+
		"2,2021-01-01 01:01:01.000000001 +0000 UTC,b|c,\"t\r\nw\r\no\"\r\n"+
		"3,2022-01-01 01:01:01.000000001 +0000 UTC,c|d,\"\"\"three\"\"\"\r\n", w.Body.String())
}

func TestMaster_ExportFeedback(t *testing.T) {
	s := newMockServer(t)
	defer s.Close(t)
	// insert feedback
	feedbacks := []data.Feedback{
		{FeedbackKey: data.FeedbackKey{FeedbackType: "click", UserId: "0", ItemId: "2"}},
		{FeedbackKey: data.FeedbackKey{FeedbackType: "share", UserId: "1", ItemId: "4"}},
		{FeedbackKey: data.FeedbackKey{FeedbackType: "read", UserId: "2", ItemId: "6"}},
	}
	err := s.DataClient.BatchInsertFeedback(feedbacks, true, true)
	assert.Nil(t, err)
	// send request
	req := httptest.NewRequest("GET", "https://example.com/", nil)
	w := httptest.NewRecorder()
	s.importExportFeedback(w, req)
	assert.Equal(t, http.StatusOK, w.Result().StatusCode)
	assert.Equal(t, "text/csv", w.Header().Get("Content-Type"))
	assert.Equal(t, "attachment;filename=feedback.csv", w.Header().Get("Content-Disposition"))
	assert.Equal(t, "feedback_type,user_id,item_id,time_stamp\r\n"+
		"click,0,2,0001-01-01 00:00:00 +0000 UTC\r\n"+
		"read,2,6,0001-01-01 00:00:00 +0000 UTC\r\n"+
		"share,1,4,0001-01-01 00:00:00 +0000 UTC\r\n", w.Body.String())
}

func TestMaster_ImportItems(t *testing.T) {
	s := newMockServer(t)
	defer s.Close(t)
	// send request
	buf := bytes.NewBuffer(nil)
	writer := multipart.NewWriter(buf)
	err := writer.WriteField("has-header", "false")
	assert.Nil(t, err)
	err = writer.WriteField("sep", "\t")
	assert.Nil(t, err)
	err = writer.WriteField("label-sep", "::")
	assert.Nil(t, err)
	err = writer.WriteField("format", "ilct")
	assert.Nil(t, err)
	file, err := writer.CreateFormFile("file", "items.csv")
	assert.Nil(t, err)
	_, err = file.Write([]byte("1\ta::b\t\"o,n,e\"\t2020-01-01 01:01:01.000000001 +0000 UTC\n" +
		"2\tb::c\t\"t\r\nw\r\no\"\t2021-01-01 01:01:01.000000001 +0000 UTC\n" +
		"3\tc::d\t\"\"\"three\"\"\"\t2022-01-01 01:01:01.000000001 +0000 UTC\n"))
	assert.Nil(t, err)
	err = writer.Close()
	assert.Nil(t, err)
	req := httptest.NewRequest("POST", "https://example.com/", buf)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()
	s.importExportItems(w, req)
	// check
	assert.Equal(t, http.StatusOK, w.Result().StatusCode)
	assert.JSONEq(t, marshal(t, server.Success{RowAffected: 3}), w.Body.String())
	_, items, err := s.DataClient.GetItems("", 100, nil)
	assert.Nil(t, err)
	assert.Equal(t, []data.Item{
		{"1", time.Date(2020, 1, 1, 1, 1, 1, 1, time.UTC), []string{"a", "b"}, "o,n,e"},
		{"2", time.Date(2021, 1, 1, 1, 1, 1, 1, time.UTC), []string{"b", "c"}, "t\r\nw\r\no"},
		{"3", time.Date(2022, 1, 1, 1, 1, 1, 1, time.UTC), []string{"c", "d"}, "\"three\""},
	}, items)
}

func TestMaster_ImportItems_DefaultFormat(t *testing.T) {
	s := newMockServer(t)
	defer s.Close(t)
	// send request
	buf := bytes.NewBuffer(nil)
	writer := multipart.NewWriter(buf)
	file, err := writer.CreateFormFile("file", "items.csv")
	assert.Nil(t, err)
	_, err = file.Write([]byte("feedback_type,user_id,item_id,time_stamp\r\n" +
		"1,2020-01-01 01:01:01.000000001 +0000 UTC,a|b,one\r\n" +
		"2,2021-01-01 01:01:01.000000001 +0000 UTC,b|c,two\r\n" +
		"3,2022-01-01 01:01:01.000000001 +0000 UTC,c|d,three\r\n"))
	assert.Nil(t, err)
	err = writer.Close()
	assert.Nil(t, err)
	req := httptest.NewRequest("POST", "https://example.com/", buf)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()
	s.importExportItems(w, req)
	// check
	assert.Equal(t, http.StatusOK, w.Result().StatusCode)
	assert.JSONEq(t, marshal(t, server.Success{RowAffected: 3}), w.Body.String())
	_, items, err := s.DataClient.GetItems("", 100, nil)
	assert.Nil(t, err)
	assert.Equal(t, []data.Item{
		{"1", time.Date(2020, 1, 1, 1, 1, 1, 1, time.UTC), []string{"a", "b"}, "one"},
		{"2", time.Date(2021, 1, 1, 1, 1, 1, 1, time.UTC), []string{"b", "c"}, "two"},
		{"3", time.Date(2022, 1, 1, 1, 1, 1, 1, time.UTC), []string{"c", "d"}, "three"},
	}, items)
}

func TestMaster_ImportFeedback(t *testing.T) {
	s := newMockServer(t)
	defer s.Close(t)
	// send request
	buf := bytes.NewBuffer(nil)
	writer := multipart.NewWriter(buf)
	err := writer.WriteField("format", "uift")
	assert.Nil(t, err)
	err = writer.WriteField("sep", "\t")
	assert.Nil(t, err)
	err = writer.WriteField("has-header", "false")
	assert.Nil(t, err)
	file, err := writer.CreateFormFile("file", "feedback.csv")
	assert.Nil(t, err)
	_, err = file.Write([]byte("0\t2\tclick\t0001-01-01 00:00:00 +0000 UTC\n" +
		"2\t6\tread\t0001-01-01 00:00:00 +0000 UTC\n" +
		"1\t4\tshare\t0001-01-01 00:00:00 +0000 UTC\n"))
	assert.Nil(t, err)
	err = writer.Close()
	assert.Nil(t, err)
	req := httptest.NewRequest("POST", "https://example.com/", buf)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()
	s.importExportFeedback(w, req)
	// check
	assert.Equal(t, http.StatusOK, w.Result().StatusCode)
	assert.JSONEq(t, marshal(t, server.Success{RowAffected: 3}), w.Body.String())
	_, feedback, err := s.DataClient.GetFeedback("", 100, nil)
	assert.Nil(t, err)
	assert.Equal(t, []data.Feedback{
		{FeedbackKey: data.FeedbackKey{FeedbackType: "click", UserId: "0", ItemId: "2"}},
		{FeedbackKey: data.FeedbackKey{FeedbackType: "read", UserId: "2", ItemId: "6"}},
		{FeedbackKey: data.FeedbackKey{FeedbackType: "share", UserId: "1", ItemId: "4"}},
	}, feedback)
}

func TestMaster_ImportFeedback_Default(t *testing.T) {
	s := newMockServer(t)
	defer s.Close(t)
	// send request
	buf := bytes.NewBuffer(nil)
	writer := multipart.NewWriter(buf)
	file, err := writer.CreateFormFile("file", "feedback.csv")
	assert.Nil(t, err)
	_, err = file.Write([]byte("feedback_type,user_id,item_id,time_stamp\r\n" +
		"click,0,2,0001-01-01 00:00:00 +0000 UTC\r\n" +
		"read,2,6,0001-01-01 00:00:00 +0000 UTC\r\n" +
		"share,1,4,0001-01-01 00:00:00 +0000 UTC\r\n"))
	assert.Nil(t, err)
	err = writer.Close()
	assert.Nil(t, err)
	req := httptest.NewRequest("POST", "https://example.com/", buf)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()
	s.importExportFeedback(w, req)
	// check
	assert.Equal(t, http.StatusOK, w.Result().StatusCode)
	assert.JSONEq(t, marshal(t, server.Success{RowAffected: 3}), w.Body.String())
	_, feedback, err := s.DataClient.GetFeedback("", 100, nil)
	assert.Nil(t, err)
	assert.Equal(t, []data.Feedback{
		{FeedbackKey: data.FeedbackKey{FeedbackType: "click", UserId: "0", ItemId: "2"}},
		{FeedbackKey: data.FeedbackKey{FeedbackType: "read", UserId: "2", ItemId: "6"}},
		{FeedbackKey: data.FeedbackKey{FeedbackType: "share", UserId: "1", ItemId: "4"}},
	}, feedback)
}

func TestMaster_GetCluster(t *testing.T) {
	s := newMockServer(t)
	defer s.Close(t)
	// add nodes
	serverNode := &Node{"alan turnin", ServerNode, "192.168.1.100", 1080}
	workerNode := &Node{"dennis ritchie", WorkerNode, "192.168.1.101", 1081}
	s.nodesInfo = make(map[string]*Node)
	s.nodesInfo["alan turning"] = serverNode
	s.nodesInfo["dennis ritchie"] = workerNode
	// get nodes
	apitest.New().
		Handler(s.handler).
		Get("/api/dashboard/cluster").
		Expect(t).
		Status(http.StatusOK).
		Body(marshal(t, []*Node{workerNode, serverNode})).
		End()
}

func TestMaster_GetStats(t *testing.T) {
	s := newMockServer(t)
	defer s.Close(t)
	// set stats
	s.rankingScore = ranking.Score{Precision: 0.1}
	s.clickScore = click.Score{Precision: 0.2}
	err := s.CacheClient.SetString(cache.GlobalMeta, cache.NumItems, "123")
	assert.Nil(t, err)
	err = s.CacheClient.SetString(cache.GlobalMeta, cache.NumUsers, "234")
	assert.Nil(t, err)
	err = s.CacheClient.SetString(cache.GlobalMeta, cache.NumPositiveFeedback, "345")
	assert.Nil(t, err)
	// get stats
	apitest.New().
		Handler(s.handler).
		Get("/api/dashboard/stats").
		Expect(t).
		Status(http.StatusOK).
		Body(marshal(t, Status{
			NumUsers:       "234",
			NumItems:       "123",
			NumPosFeedback: "345",
			RankingScore:   0.1,
			ClickScore:     0.2,
		})).
		End()
}

func TestMaster_GetUsers(t *testing.T) {
	s := newMockServer(t)
	defer s.Close(t)
	// add users
	users := []User{
		{data.User{UserId: "0"}, "2000-01-01", "2020-01-02"},
		{data.User{UserId: "1"}, "2001-01-01", "2021-01-02"},
		{data.User{UserId: "2"}, "2002-01-01", "2022-01-02"},
	}
	for _, user := range users {
		err := s.DataClient.InsertUser(user.User)
		assert.Nil(t, err)
		err = s.CacheClient.SetString(cache.LastActiveTime, user.UserId, user.LastActiveTime)
		assert.Nil(t, err)
		err = s.CacheClient.SetString(cache.LastUpdateRecommendTime, user.UserId, user.LastUpdateTime)
		assert.Nil(t, err)
	}
	// get users
	apitest.New().
		Handler(s.handler).
		Get("/api/dashboard/users").
		Expect(t).
		Status(http.StatusOK).
		Body(marshal(t, UserIterator{
			Cursor: "",
			Users:  users,
		})).
		End()
	// get a user
	apitest.New().
		Handler(s.handler).
		Get("/api/dashboard/user/1").
		Expect(t).
		Status(http.StatusOK).
		Body(marshal(t, users[1])).
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
		{cache.LatestItems, "", "/api/dashboard/latest/"},
		{cache.PopularItems, "", "/api/dashboard/popular/"},
		{cache.SimilarItems, "0", "/api/dashboard/neighbors/0"},
	}
	for _, operator := range operators {
		t.Logf("test RESTful API: %v", operator.Get)
		// Put itemIds
		itemIds := []cache.ScoredItem{
			{"0", 100},
			{"1", 99},
			{"2", 98},
			{"3", 97},
			{"4", 96},
		}
		err := s.CacheClient.SetScores(operator.Prefix, operator.Label, itemIds)
		assert.Nil(t, err)
		items := make([]data.Item, 0)
		for _, item := range itemIds {
			items = append(items, data.Item{ItemId: item.ItemId})
			err = s.DataClient.InsertItem(data.Item{ItemId: item.ItemId})
			assert.Nil(t, err)
		}
		apitest.New().
			Handler(s.handler).
			Get(operator.Get).
			Expect(t).
			Status(http.StatusOK).
			Body(marshal(t, items)).
			End()
	}
}

func TestServer_Feedback(t *testing.T) {
	s := newMockServer(t)
	defer s.Close(t)
	// insert feedback
	feedback := []Feedback{
		{FeedbackType: "click", UserId: "0", Item: data.Item{ItemId: "0"}},
		{FeedbackType: "click", UserId: "0", Item: data.Item{ItemId: "2"}},
		{FeedbackType: "click", UserId: "0", Item: data.Item{ItemId: "4"}},
		{FeedbackType: "click", UserId: "0", Item: data.Item{ItemId: "6"}},
		{FeedbackType: "click", UserId: "0", Item: data.Item{ItemId: "8"}},
	}
	for _, v := range feedback {
		err := s.DataClient.InsertFeedback(data.Feedback{
			FeedbackKey: data.FeedbackKey{FeedbackType: v.FeedbackType, UserId: v.UserId, ItemId: v.Item.ItemId},
		}, true, true)
		assert.Nil(t, err)
	}
	// get feedback
	apitest.New().
		Handler(s.handler).
		Get("/api/dashboard/user/0/feedback/click").
		Expect(t).
		Status(http.StatusOK).
		Body(marshal(t, feedback)).
		End()
}

func TestServer_GetRecommends(t *testing.T) {
	s := newMockServer(t)
	defer s.Close(t)
	// inset recommendation
	itemIds := []cache.ScoredItem{
		{"1", 99},
		{"2", 98},
		{"3", 97},
		{"4", 96},
		{"5", 95},
		{"6", 94},
		{"7", 93},
		{"8", 92},
	}
	err := s.CacheClient.SetScores(cache.RecommendItems, "0", itemIds)
	assert.Nil(t, err)
	// insert feedback
	feedback := []data.Feedback{
		{FeedbackKey: data.FeedbackKey{FeedbackType: "a", UserId: "0", ItemId: "2"}},
		{FeedbackKey: data.FeedbackKey{FeedbackType: "a", UserId: "0", ItemId: "4"}},
	}
	err = s.RestServer.InsertFeedbackToCache(feedback)
	assert.Nil(t, err)
	// insert items
	for _, item := range itemIds {
		err = s.DataClient.InsertItem(data.Item{ItemId: item.ItemId})
		assert.Nil(t, err)
	}
	apitest.New().
		Handler(s.handler).
		Get("/api/dashboard/recommend/0").
		Expect(t).
		Status(http.StatusOK).
		Body(marshal(t, []data.Item{
			{ItemId: "1"}, {ItemId: "3"}, {ItemId: "5"}, {ItemId: "6"}, {ItemId: "7"}, {ItemId: "8"},
		})).
		End()
}

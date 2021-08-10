package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"
)

var entryPoint string = "http://127.0.0.1:8087"

func insertFeedback(feedbackType string, userId string, itemId string) {
	data := []map[string]interface{}{
		{
			"FeedbackType": feedbackType,
			"UserId":       fmt.Sprint(userId),
			"ItemId":       itemId,
			"Timestamp":    time.Now().Format("2006-01-02 15:04:05"),
		},
	}

	d, _ := json.Marshal(data)

	resp, err := http.Post(entryPoint+"/api/feedback",
		"application/json",
		strings.NewReader(string(d)))
	if err != nil {
		fmt.Println(err)
		return
	}
	defer resp.Body.Close()
	fmt.Println(resp)
}

func getRecommend(userId string, n int64) []string {
	resp, err := http.Get(entryPoint + fmt.Sprintf("/api/recommend/%s?n=%d", userId, n))
	var IDs []string
	if err != nil {
		fmt.Println(err)
		return IDs
	}
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)
	err = json.Unmarshal(body, &IDs)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(IDs)
	return IDs
}

func insertItem(item interface{}) {
	d, _ := json.Marshal(item)

	resp, err := http.Post(entryPoint+"/api/item",
		"application/json",
		strings.NewReader(string(d)))
	if err != nil {
		fmt.Println(err)
		return
	}
	defer resp.Body.Close()
	fmt.Println(resp)
}

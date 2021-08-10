package crawler

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

var entryPoint string = "https://gitee.com/api/v5/"

func GetStared() (r []map[string]interface{}) {
	token := "b3f76a65da9e86521f37cfefeba06eb7"
	resp, err := http.Get(entryPoint + fmt.Sprintf("user/starred?access_token=%s&sort=created&direction=desc&page=1&per_page=20", token))
	if err != nil {
		fmt.Println(err)
		return
	}
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)
	err = json.Unmarshal(body, &r)
	if err != nil {
		fmt.Println(err)
	}
	/*
		"id": 11528706,
		"full_name": "yomorun/yomo",
		"human_name": "YoMo/yomo",
		"url": "https://gitee.com/api/v5/repos/yomorun/yomo",
		"description": "YoMo是一个Streaming Serverless Framework，支持低时延边缘计算应用的开发。YoMo基于QUIC Transport和FRP范式，已被应用在工业互联网、物联网领域。",
	*/
	return
}

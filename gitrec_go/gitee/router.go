package main

import (
	"fmt"
	"io"
	"strings"

	restful "github.com/emicklei/go-restful"
	restfulspec "github.com/emicklei/go-restful-openapi/v2"
)

type Router struct {
}

func (r *Router) RegisterTo(container *restful.Container) {
	ws := new(restful.WebService)
	ws.Route(ws.GET("/").To(r.index).
		Doc("home page").
		Metadata(restfulspec.KeyOpenAPITags, []string{"home"}).
		Param(ws.HeaderParameter("X-API-Key", "secret key for RESTful API")))

	ws.Route(ws.GET("/login").To(r.login).
		Doc("login a user.").
		Metadata(restfulspec.KeyOpenAPITags, []string{"login"}).
		Param(ws.HeaderParameter("X-API-Key", "secret key for RESTful API")).
		Param(ws.PathParameter("user-id", "identifier of the user").DataType("string")))

	ws.Route(ws.GET("/api/repo").To(r.repo).
		Doc("api repo.").
		Metadata(restfulspec.KeyOpenAPITags, []string{"login"}).
		Param(ws.HeaderParameter("X-API-Key", "secret key for RESTful API")).
		Param(ws.PathParameter("user-id", "identifier of the user").DataType("string")))

	ws.Route(ws.GET("/api/like/{repo_name}").To(r.repoLike).
		Doc("api repo.").
		Metadata(restfulspec.KeyOpenAPITags, []string{"login"}).
		Param(ws.HeaderParameter("X-API-Key", "secret key for RESTful API")).
		Param(ws.PathParameter("user-id", "identifier of the user").DataType("string")))

	ws.Route(ws.GET("/api/read/{repo_name}").To(r.repoRead).
		Doc("api repo.").
		Metadata(restfulspec.KeyOpenAPITags, []string{"login"}).
		Param(ws.HeaderParameter("X-API-Key", "secret key for RESTful API")).
		Param(ws.PathParameter("user-id", "identifier of the user").DataType("string")))

	//https://github.com/emicklei/go-restful/blob/v3/examples/static/restful-serve-static.go
	ws.Route(ws.GET("/static/{subpath:*}").To(staticFromPathParam))
	ws.Route(ws.GET("/static").To(staticFromQueryParam))

	container.Add(ws)
}

var login bool = false

func (r *Router) index(request *restful.Request, response *restful.Response) {
	//https://www.cnblogs.com/xiaoleiel/p/8295635.html
	if !login {
		url := "/static/index.html"
		redirect(request, response, url)
		return
	}
	io.WriteString(response.ResponseWriter, "this would be a normal response")
}

func (r *Router) login(request *restful.Request, response *restful.Response) {
	url := "/static/login.html"
	redirect(request, response, url)
	return
}

var sessionUserId string = "zhenghaoz"

func (r *Router) repo(request *restful.Request, response *restful.Response) {
	repoIDs := getRecommend(sessionUserId, 10)
	ID := "test"
	if len(repoIDs) > 1 {
		ID = repoIDs[0]
	}
	ID = strings.Replace(ID, ":", "/", -1)
	content := map[string]interface{}{
		"item_id":        "test",
		"full_name":      ID,
		"html_url":       "http://127.0.0.1:5000/test",
		"stargazers_url": "http://127.0.0.1:5000/test",
		"forks_url":      "http://127.0.0.1:5000/test",
		"stargazers":     3,
		"forks":          5,
		"readme":         ID,
	}
	if err := response.WriteAsJson(content); err != nil {
		fmt.Println(err)
	}

}

func (r *Router) repoLike(request *restful.Request, response *restful.Response) {
	repoName := request.PathParameter("repo_name")
	insertFeedback("like", sessionUserId, repoName)
	io.WriteString(response.ResponseWriter, "like sucess")
}

func (r *Router) repoRead(request *restful.Request, response *restful.Response) {
	repoName := request.PathParameter("repo_name")
	insertFeedback("read", sessionUserId, repoName)
	io.WriteString(response.ResponseWriter, "read sucess")
}

package main

import (
	"log"
	"net/http"

	restful "github.com/emicklei/go-restful"
	swagger "github.com/emicklei/go-restful-swagger12"
)

func main() {
	wsContainer := restful.NewContainer()
	// 跨域过滤器
	cors := restful.CrossOriginResourceSharing{
		ExposeHeaders:  []string{"X-My-Header"},
		AllowedHeaders: []string{"Content-Type", "Accept"},
		AllowedMethods: []string{"GET", "POST"},
		CookiesAllowed: false,
		Container:      wsContainer}
	wsContainer.Filter(cors.Filter)

	// Add container filter to respond to OPTIONS
	wsContainer.Filter(wsContainer.OPTIONSFilter)

	config := swagger.Config{
		WebServices:    restful.DefaultContainer.RegisteredWebServices(), // you control what services are visible
		WebServicesUrl: "http://localhost:5000",
		ApiPath:        "/apidocs.json",
		ApiVersion:     "V1.0",
		// Optionally, specify where the UI is located
		SwaggerPath:     "/apidocs/",
		SwaggerFilePath: "./doc"}
	swagger.RegisterSwaggerService(config, wsContainer)
	swagger.InstallSwaggerService(config)

	r := Router{}
	r.RegisterTo(wsContainer)

	log.Print("start listening on localhost:5000")
	server := &http.Server{Addr: ":5000", Handler: wsContainer}
	defer server.Close()
	log.Fatal(server.ListenAndServe())
}

package main

import (
	"fmt"
	"io"
	"net/http"

	restful "github.com/emicklei/go-restful"
)

func redirect(request *restful.Request, response *restful.Response, url string) {
	//http.Redirect(response, request, url, http.StatusFound)
	h := response.Header()

	// RFC 7231 notes that a short HTML body is usually included in
	// the response because older user agents may not understand 301/307.
	// Do it only if the request didn't already have a Content-Type header.
	h.Set("Location", url)

	h.Set("Content-Type", "text/html; charset=utf-8")

	response.WriteHeader(http.StatusFound)

	// Shouldn't send the body for POST or HEAD; that leaves GET.

	body := "<a href=\"" + url + "\">" + fmt.Sprint(http.StatusFound) + "</a>.\n"
	io.WriteString(response.ResponseWriter, body)
}

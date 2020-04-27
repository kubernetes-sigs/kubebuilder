package main

import (
	. "github.com/emicklei/go-restful"
	"io"
	"log"
	"net/http"
)

// This example shows how to create a Route with google custom method
// Requires the use of a CurlyRouter and path should end with the custom method
//
// GET http://localhost:8080/resource:validate
// POST http://localhost:8080/resource/some-resource-id:init
// POST http://localhost:8080/resource/some-resource-id:recycle

func main() {
	DefaultContainer.Router(CurlyRouter{})
	ws := new(WebService)

	ws.Route(ws.GET("/resource:validate").To(validateHandler))
	ws.Route(ws.POST("/resource/{resourceId}:init").To(initHandler))
	ws.Route(ws.POST("/resource/{resourceId}:recycle").To(recycleHandler))

	Add(ws)

	println("[go-restful] serve path tails from http://localhost:8080/basepath")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func validateHandler(req *Request, resp *Response) {
	io.WriteString(resp, "validate resource completed")
}

func initHandler(req *Request, resp *Response) {
	io.WriteString(resp, "init resource completed, resourceId: "+req.PathParameter("resourceId"))
}

func recycleHandler(req *Request, resp *Response) {
	io.WriteString(resp, "recycle resource completed, resourceId: "+req.PathParameter("resourceId"))
}

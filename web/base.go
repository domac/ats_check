package web

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

//init the server with ip:port
func InitServer(addr string) (*http.Server, error) {
	r, err := loadRouter()
	if err != nil {
		return nil, err
	}
	srv := &http.Server{
		Handler:      r,
		Addr:         addr,
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}
	return srv, nil
}

//the context with request from client
//the context with response to client
type Context struct {
	R         *http.Request
	W         http.ResponseWriter
	callbacks []func()
	Data      map[interface{}]interface{}
}

func newContext(w http.ResponseWriter, r *http.Request) *Context {
	return &Context{
		R:         r,
		W:         w,
		callbacks: make([]func(), 0, 2),
		Data:      make(map[interface{}]interface{}, 2),
	}
}

//callback function invoked by context which happen after request
func (ctx *Context) AddCallBack(f func()) {
	ctx.callbacks = append(ctx.callbacks, f)
}

func (ctx *Context) Done() {
	n := len(ctx.callbacks) - 1
	for i := n; i >= 0; i-- {
		ctx.callbacks[i]()
	}
}

//web api base class
type BaseHandler struct {
	Ctx    map[string]interface{}
	Handle func(ctx *Context)
}

func NewBaseHandler(f func(ctx *Context)) BaseHandler {
	return BaseHandler{
		Handle: f,
	}
}

//http main
func (b BaseHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	defer func() {
		err_ := recover()
		if err_ == nil {
			return
		}
		return
	}()

	ctx := newContext(w, r)
	defer ctx.Done()
	b.Handle(ctx)
}

func getStringVal(n string, r *http.Request) string {
	return strings.TrimSpace(r.FormValue(n))
}

//reponse data using Json
func reponseJson(w http.ResponseWriter, data interface{}) {
	reponseJsonWithStatusCode(w, http.StatusOK, data)
}

//reponse data using Json + statusCode
func reponseJsonWithStatusCode(w http.ResponseWriter, httpCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	s := ""
	b, err := json.Marshal(data)
	if err != nil {
		s = `{"error":"json.Marshal error"}`
		w.WriteHeader(http.StatusInternalServerError)
	} else {
		s = string(b)
		w.WriteHeader(httpCode)
	}
	fmt.Fprint(w, s)
}

//reponse data using PlainText
func reponsePlainText(w http.ResponseWriter, data string) {
	w.Header().Set("Content-Type", "text/plain")
	fmt.Fprint(w, data)
}

func reponsePlainTextWithStatusCode(w http.ResponseWriter, httpCode int, data string) {
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(httpCode)
	fmt.Fprint(w, data)
}

func reponseByteText(w http.ResponseWriter, data []byte) {
	s := string(data)
	reponsePlainText(w, s)
}

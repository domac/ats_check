package web

import (
	"github.com/gorilla/mux"
	"github.com/domac/ats_check/app"
)

//加载路由
func loadRouter(applicationContext *app.App) (r *mux.Router, err error) {
	atsHandler := &ATSHandler{applicationContext: applicationContext}
	r = mux.NewRouter()
	v1Subrouter := r.PathPrefix("/ats").Subrouter()
	ih := NewATSHandler(atsHandler.Parents)
	v1Subrouter.Handle("/parents", ih).Methods("GET") //GET Method

	verHandler := &VerHandler{}
	iv := NewVerHandler(verHandler.Version)
	v1Subrouter.Handle("/", iv).Methods("GET") //GET Method
	return r, nil
}

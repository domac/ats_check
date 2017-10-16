package web

import (
	"github.com/gorilla/mux"
)

//load router table
func loadRouter() (r *mux.Router, err error) {
	atsHandler := &ATSHandler{}
	r = mux.NewRouter()
	v1Subrouter := r.PathPrefix("/ats").Subrouter()
	ih := NewATSHandler(atsHandler.Parents)
	v1Subrouter.Handle("/parents", ih).Methods("GET") //GET Method

	verHandler := &VerHandler{}
	iv := NewVerHandler(verHandler.Version)
	v1Subrouter.Handle("/", iv).Methods("GET") //GET Method
	return r, nil
}

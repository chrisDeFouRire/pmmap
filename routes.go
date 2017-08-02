package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/gorilla/mux"
)

type createJobJSON struct {
	URL         string `json:"url"`
	Maxsize     uint   `json:"maxsize"`
	Concurrency int    `json:"concurrency"`
}

type kvJSON struct {
	Key   string      `json:"key"`
	Value interface{} `json:"value"`
}

func createJob(w http.ResponseWriter, req *http.Request) {
	defer req.Body.Close()
	var query createJobJSON

	if err := json.NewDecoder(req.Body).Decode(&query); err == nil {
		if u, err := url.Parse(query.URL); err == nil {

			job := CreateJob(Secret, *u, query.Maxsize)
			job.Start(query.Concurrency)
			Manager.addJob(job)

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(job)
			return
		}
	}
	w.WriteHeader(http.StatusBadRequest)
}

func getJob(w http.ResponseWriter, req *http.Request) {
	id := mux.Vars(req)["id"]
	job := Manager.getJob(id)
	if job == nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(job)
}

func addInput(w http.ResponseWriter, req *http.Request) {
	defer req.Body.Close()
	id := mux.Vars(req)["id"]
	job := Manager.getJob(id)
	if job == nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	var body []kvJSON
	if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	}
	for _, eachkv := range body {
		bytes, _ := json.Marshal(eachkv.Value)
		if err := job.AddToJob(eachkv.Key, bytes); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(err.Error()))
			return
		}
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(job)
}

func allInputSent(w http.ResponseWriter, req *http.Request) {
	defer req.Body.Close()
	id := mux.Vars(req)["id"]
	job := Manager.getJob(id)
	if job == nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	if err := job.AllInputsWereSent(); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(job)
}

func getJobOutputs(w http.ResponseWriter, req *http.Request) {
	defer req.Body.Close()
	id := mux.Vars(req)["id"]
	job := Manager.getJob(id)
	if job == nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	if job.State != AllOutputReceived {
		w.WriteHeader(http.StatusExpectationFailed)
		return
	}
	<-job.Complete
	res, err := job.GetResults()
	fmt.Println(res)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}
	result := make([]kvJSON, len(res))
	for index, eachkv := range res {
		var value interface{}
		json.Unmarshal(eachkv.Value, &value)
		result[index] = kvJSON{
			Key:   eachkv.Key,
			Value: value,
		}
	}
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(result)
}

func deleteJob(w http.ResponseWriter, req *http.Request) {

}

func routes() *mux.Router {
	routes := mux.NewRouter()

	routes.HandleFunc("/job", createJob).Methods("POST")
	routes.HandleFunc("/job/{id}", getJob).Methods("GET")
	routes.HandleFunc("/job/{id}/output", getJobOutputs).Methods("GET")
	routes.HandleFunc("/job/{id}/input", addInput).Methods("PUT")
	routes.HandleFunc("/job/{id}/complete", allInputSent).Methods("POST")
	routes.HandleFunc("/job/{id}", deleteJob).Methods("DELETE")
	return routes
}

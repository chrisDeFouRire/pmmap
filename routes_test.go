package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/gorilla/mux"
)

const (
	localServerAddress = "localhost:7777"
	webhook            = "/dowork"
)

func TestMain(m *testing.M) {
	go func() { // start a dummy server
		srv := &http.Server{
			Addr: "localhost:7777",
		}
		r := mux.NewRouter()
		r.HandleFunc(webhook+"/{key}", func(w http.ResponseWriter, req *http.Request) {
			defer req.Body.Close()
			if req.Method != "POST" {
				log.Fatalf("Method %s should have been POST", req.Method)
			}
			if req.Header.Get("PMMAP-auth") != "testSecret" {
				log.Fatalf("Incorrect secret key")
			}
			if mux.Vars(req)["key"][0:5] != "hello" {
				log.Fatalf("Incorrect key")
			}
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("\"world (" + mux.Vars(req)["key"] + ")\""))
			fmt.Print("Backend received")
		})
		r.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
			log.Fatalf("Incorrect URL: %s", req.RequestURI)
		})
		http.Handle("/", r)
		srv.ListenAndServe()
	}()
	go main()
	time.Sleep(time.Millisecond * 50)
	os.Exit(m.Run())
}

func TestIntegration(t *testing.T) {
	req := struct {
		Secret      string `json:"secret"`
		URL         string `json:"url"`
		Maxsize     int    `json:"maxsize"`
		Concurrency int    `json:"concurrency"`
	}{"testSecret", "localhost:7777/work", 10, 10}
	b, _ := json.Marshal(req)
	create, createerr := http.Post("http://localhost:8080/job", "application/json", bytes.NewReader(b))
	if createerr != nil {
		t.Fatalf("error at creation %v", createerr)
	}
	fmt.Println("Created")
	var m map[string]interface{}
	json.NewDecoder(create.Body).Decode(&m)
	jobid := m["id"].(string)

	one := struct {
		Key   string      `json:"key"`
		Value interface{} `json:"value"`
	}{"hello23", ""}
	var onearray = make([]interface{}, 1)
	onearray[0] = one
	b, _ = json.Marshal(onearray)
	putreq, _ := http.NewRequest("PUT", "http://localhost:8080/job/"+jobid, bytes.NewReader(b))
	putreq.Header.Add("Content-Type", "application/json")
	client := &http.Client{}
	client.Do(putreq)
	fmt.Println("Input added")

	http.Post("http://localhost:8080/job/"+jobid+"/complete", "application/json", nil)
	fmt.Println("Job complete")

	res, _ := http.Get("http://localhost:8080/job/" + jobid + "/output")
	var r interface{}

	decodeerr := json.NewDecoder(res.Body).Decode(&r)
	if decodeerr != nil {
		log.Fatal(decodeerr)
	}
	fmt.Printf("output %v", r)
}

package main

import (
	"bytes"
	"encoding/json"
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
			if req.Header.Get("PMMAP-auth") != Secret {
				log.Fatalf("Incorrect secret key %s vs. %s", req.Header.Get("PMMAP-auth"), Secret)
			}
			if mux.Vars(req)["key"][0:5] != "hello" {
				log.Fatalf("Incorrect key")
			}
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("\"world (" + mux.Vars(req)["key"] + ")\""))
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
	// Step 1: create job
	req := struct {
		Secret      string `json:"secret"`
		URL         string `json:"url"`
		Maxsize     int    `json:"maxsize"`
		Concurrency int    `json:"concurrency"`
	}{Secret, "http://localhost:7777/dowork", 10, 10}
	b, _ := json.Marshal(req)
	create, createerr := http.Post("http://localhost:8080/job", "application/json", bytes.NewReader(b))
	if createerr != nil {
		t.Fatalf("error at creation %v", createerr)
	}
	if create.StatusCode != http.StatusCreated {
		t.Fatalf("create job should have returned code 201-Created")
	}

	var m map[string]interface{}
	json.NewDecoder(create.Body).Decode(&m)
	jobid := m["id"].(string)
	if m["inputs"].(float64) != 0 {
		t.Fatalf("Job should report 0 inputs")
	}
	if m["outputs"].(float64) != 0 {
		t.Fatalf("Job should report 0 outpus")
	}
	if m["url"].(string) != "http://localhost:7777/dowork" {
		t.Fatalf("Job should report correct url")
	}

	// step 2: send input
	one := struct {
		Key   string      `json:"key"`
		Value interface{} `json:"value"`
	}{"hello23", "testhello23"}
	var onearray = make([]interface{}, 1)
	onearray[0] = one
	b, _ = json.Marshal(onearray)
	putreq, _ := http.NewRequest("PUT", "http://localhost:8080/job/"+jobid+"/input", bytes.NewReader(b))
	putreq.Header.Add("Content-Type", "application/json")
	client := &http.Client{}
	putres, puterr := client.Do(putreq)
	if puterr != nil {
		log.Println("Error while PUTting input")
		log.Fatal(puterr)
	}
	if putres.StatusCode != http.StatusCreated {
		log.Fatalf("Putting input should reply with 201-Created instead of %d", putres.StatusCode)
	}

	// step 3: signal the input set is complete
	http.Post("http://localhost:8080/job/"+jobid+"/complete", "application/json", nil)

	// step 4: get results
	res, reserr := http.Get("http://localhost:8080/job/" + jobid + "/output")
	if reserr != nil {
		log.Print("Error while getting results")
		log.Fatal(reserr)
	}
	var r []map[string]interface{}
	decodeerr := json.NewDecoder(res.Body).Decode(&r)
	if decodeerr != nil {
		log.Fatal(decodeerr)
	}
	if len(r) != 1 {
		log.Fatalf("There should have been only one result, there were %d", len(r))
	}
	if r[0]["value"] != "world (hello23)" || r[0]["key"] != "hello23" {
		log.Fatalf("Wrong result received")
	}
}

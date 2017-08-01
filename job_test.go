package main

import (
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/gorilla/mux"
)

const (
	localServerAddress = "localhost:7777"
	webhook            = "/dowork"
)

func TestMain(m *testing.M) {
	ready := make(chan bool)
	go func() { // start a dummy server
		srv := &http.Server{
			Addr: localServerAddress,
		}
		r := mux.NewRouter()
		r.HandleFunc(webhook+"/{key}", func(w http.ResponseWriter, req *http.Request) {
			defer req.Body.Close()
			if req.Method != "POST" {
				log.Fatalf("Method %s should have been POST", req.Method)
			}
			if req.Header.Get("PMMAP-auth") != "testSecret!321" {
				log.Fatalf("Incorrect secret key")
			}
			if mux.Vars(req)["key"][0:5] != "hello" {
				log.Fatalf("Incorrect key")
			}
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("world"))
		})
		r.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
			log.Fatalf("Incorrect URL: %s", req.RequestURI)
		})
		http.Handle("/", r)
		ready <- true
		srv.ListenAndServe()
	}()
	<-ready
	time.Sleep(5 * time.Millisecond)
	os.Exit(m.Run())
}

// TestCreateJob tests that a job with overcapacity in both channel and goroutines
// does finish as it should
func TestCreateWithOneJob(t *testing.T) {
	u, _ := url.Parse("http://" + localServerAddress + webhook)
	job := CreateJob("testSecret!321", *u, 10, 10)
	job.Start()

	if job.GetCompletionRate() != 0 {
		t.Fatalf("Job completion rate should be 0, it is %f", job.GetCompletionRate())
	}
	if job.GetInputsCount() != 0 {
		t.Fatalf("there should be 0 input, there are %d", job.GetInputsCount())
	}
	if job.GetOutputsCount() != 0 {
		t.Fatalf("there should be 0 output, there are %d", job.GetOutputsCount())
	}

	job.AddToJob("hello", []byte("world"))
	job.AllInputsWereSent()
	<-job.Complete

	if job.GetCompletionRate() != 1.0 {
		t.Fatalf("Job completion rate should be 1, it is %f", job.GetCompletionRate())
	}
	if job.GetInputsCount() != 1 {
		t.Fatalf("there should be 1 input, there are %d", job.GetInputsCount())
	}
	if job.GetOutputsCount() != 1 {
		t.Fatalf("there should be 1 output, there are %d", job.GetOutputsCount())
	}
	if string(job.GetResult("hello")) != "world" {
		t.Fatalf("result should been returned")
	}
}

const count = 200
const concurrency = 2

// TestCreateWithNJobs tests with N jobs (N = count)
func TestCreateWithNjobs(t *testing.T) {
	u, _ := url.Parse("http://" + localServerAddress + webhook)
	job := CreateJob("testSecret!321", *u, count, concurrency)
	job.Start()
	if job.GetCompletionRate() != 0 {
		t.Fatalf("Job completion rate should be 0, it is %f", job.GetCompletionRate())
	}
	if job.GetInputsCount() != 0 {
		t.Fatalf("there should be 0 input, there are %d", job.GetInputsCount())
	}
	if job.GetOutputsCount() != 0 {
		t.Fatalf("there should be 0 output, there are %d", job.GetOutputsCount())
	}

	for c := 0; c < count; c++ {
		job.AddToJob("hello"+strconv.Itoa(c), []byte("world"))
	}
	job.AllInputsWereSent()
	<-job.Complete

	if job.GetCompletionRate() != 1.0 {
		t.Fatalf("Job completion rate should be 1, it is %f", job.GetCompletionRate())
	}
	if job.GetInputsCount() != count {
		t.Fatalf("there should be 1 input, there are %d", job.GetInputsCount())
	}
	if job.GetOutputsCount() != count {
		t.Fatalf("there should be 1 output, there are %d", job.GetOutputsCount())
	}
	if string(job.GetResult("hello0")) != "world" && string(job.GetResult("hello"+strconv.Itoa(count-1))) != "world" {
		t.Fatalf("result should be returned")
	}
}

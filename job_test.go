package main

import (
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	go func() { // start a dummy server
		srv := &http.Server{
			Addr: "localhost:7777",
		}
		http.HandleFunc("/dowork", func(w http.ResponseWriter, req *http.Request) {
			defer req.Body.Close()
			if req.Method != "POST" {
				log.Fatalf("Method %s should have been POST", req.Method)
			}
			io.Copy(w, req.Body)
		})
		srv.ListenAndServe()
	}()
	os.Exit(m.Run())
}

// TestCreateJob tests that a job with overcapacity in both channel and goroutines
// does finish as it should
func TestCreateJob(t *testing.T) {
	u, _ := url.Parse("http://localhost:7777/dowork")
	job := CreateJob(*u, 10, 10)

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
	if string(job.Results["hello"]) != "put the result here" {
		t.Fatalf("result should been returned")
	}
}

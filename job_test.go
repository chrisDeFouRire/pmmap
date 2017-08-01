package main

import (
	"net/url"
	"strconv"
	"testing"
)

// TestCreateJob tests that a job with overcapacity in both channel and goroutines
// does finish as it should
func aTestCreateWithOneJob(t *testing.T) {
	u, _ := url.Parse("http://" + localServerAddress + webhook)
	job := CreateJob("testSecret!321", *u, 10)
	job.Start(10)

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
func aTestCreateWithNjobs(t *testing.T) {
	u, _ := url.Parse("http://" + localServerAddress + webhook)
	job := CreateJob("testSecret!321", *u, count)
	job.Start(10)
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

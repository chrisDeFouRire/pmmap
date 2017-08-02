package main

import (
	"net/url"
	"strconv"
	"testing"
	"time"
)

// TestCreateJob tests that a job with overcapacity in both channel and goroutines
// does finish as it should
func TestCreateWithOneJob(t *testing.T) {
	u, _ := url.Parse("http://" + localServerAddress + webhook)
	job := CreateJob(Secret, *u, 10)
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
	time.Sleep(5 * time.Millisecond)
	if job.GetInputsCount() != 1 {
		t.Fatalf("there should be 1 input, there are %d", job.GetInputsCount())
	}
	if job.GetOutputsCount() != 1 {
		t.Fatalf("there should be 1 output, there are %d", job.GetOutputsCount())
	}
	if job.GetCompletionRate() != 1.0 {
		t.Fatalf("Job completion rate should be 1, it is %f", job.GetCompletionRate())
	}
	if string(job.GetResult("hello")) != `"world (hello)"` {
		t.Fatalf("result should been returned (%s)", string(job.GetResult("hello")))
	}
}

const _count = 200
const concurrency = 2

// TestCreateWithNJobs tests with N jobs (N = _count)
func TestCreateWithNjobs(t *testing.T) {
	u, _ := url.Parse("http://" + localServerAddress + webhook)
	job := CreateJob(Secret, *u, _count)
	job.Start(2)
	if job.GetCompletionRate() != 0 {
		t.Fatalf("Job completion rate should be 0, it is %f", job.GetCompletionRate())
	}
	if job.GetInputsCount() != 0 {
		t.Fatalf("there should be 0 input, there are %d", job.GetInputsCount())
	}
	if job.GetOutputsCount() != 0 {
		t.Fatalf("there should be 0 output, there are %d", job.GetOutputsCount())
	}

	for c := 0; c < _count; c++ {
		job.AddToJob("hello"+strconv.Itoa(c), []byte("world"))
	}
	job.AllInputsWereSent()
	<-job.Complete

	if job.GetInputsCount() != _count {
		t.Fatalf("there should be %d, input, there are %d", _count, job.GetInputsCount())
	}
	if job.GetOutputsCount() != _count {
		t.Fatalf("there should be %d output, there are %d", _count, job.GetOutputsCount())
	}
	if job.GetCompletionRate() != 1.0 {
		t.Fatalf("Job completion rate should be 1, it is %f", job.GetCompletionRate())
	}
	if string(job.GetResult("hello0")) != `"world (hello0)"` && string(job.GetResult("hello"+strconv.Itoa(_count-1))) != `"world (hello199)"` {
		t.Fatalf("result should be returned %s", string(job.GetResult("hello0")))
	}
}

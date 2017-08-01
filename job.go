package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"sync"
	"sync/atomic"

	"github.com/satori/go.uuid"
)

const dbPath = "./db/"

// Input is the data structure used as inputs to jobs
type Input struct {
	Key        string
	Value      []byte
	RetryCount int
}

// Output encapsulates the output of jobs
type Output struct {
	Key   string
	Value []byte
}

// Job encapsulate a single instance of a job
type Job struct {
	ID        string  // the job ID
	secretKey string  // the job secret key
	workURL   url.URL // the URL we send work to
	inChan    chan (Input)
	outChan   chan (Output)
	wg        *sync.WaitGroup
	Complete  chan (bool)

	inputsCount  int64
	outputsCount int64
	State        int64

	Results map[string][]byte
}

// CreateJob creates a new Job, ready to start
// returns a job
func CreateJob(secret string, u url.URL, maxsize uint, concurrency int) *Job {
	_id := uuid.NewV4().String()

	// create the job instance
	job := &Job{
		ID:        _id,
		secretKey: secret,
		workURL:   u,
		inChan:    make(chan Input, maxsize),
		outChan:   make(chan Output),
		wg:        &sync.WaitGroup{},
		Complete:  make(chan bool, 1),
		State:     Created,
		Results:   make(map[string][]byte),
	}

	// wait until all workers are done
	job.startCompletionWaiter()

	// start Output receiver
	job.startOutputLogger()

	// start all workers
	job.startWorkers(concurrency)

	return job
}

// AddInputsToJob adds more than one input to the job
func (job *Job) AddInputsToJob(inputs []Input) error {
	if !job.canReceiveInput() {
		return fmt.Errorf("Job %s can't receive more inputs", job.ID)
	}
	job.receiving(len(inputs))
	for _, eachJob := range inputs {
		job.inChan <- eachJob
	}
	return nil
}

// AddToJob adds an input to the job
func (job *Job) AddToJob(key string, value []byte) error {
	if !job.canReceiveInput() {
		return fmt.Errorf("Job %s can't receive more inputs", job.ID)
	}
	job.receiving(1)
	atomic.AddInt64(&job.inputsCount, 1)
	job.inChan <- Input{Key: key, Value: value}
	return nil
}

// AllInputsWereSent is called when all inputs have been sent
// no more input can be sent
func (job *Job) AllInputsWereSent() error {
	state := atomic.LoadInt64(&job.State)
	if state != ReceivingInputs {
		return fmt.Errorf("Wrong state transition")
	}
	atomic.StoreInt64(&job.State, AllInputReceived)
	close(job.inChan)
	return nil
}

// GetInputsCount returns the current number of inputs in the job
func (job *Job) GetInputsCount() int64 {
	return atomic.LoadInt64(&job.inputsCount)
}

// GetOutputsCount returns the current number of outputs from the job
func (job *Job) GetOutputsCount() int64 {
	return atomic.LoadInt64(&job.outputsCount)
}

// GetCompletionRate rate returns percentage of completion. Returns 0 if not started
func (job *Job) GetCompletionRate() float64 {
	inputs := job.GetInputsCount()
	if inputs == 0 { // return 0 if no inputs received
		return 0
	}
	return float64(job.GetOutputsCount()) / float64(inputs)
}

// startOutputLogger receives all outputs
func (job *Job) startOutputLogger() {
	go func() {
		for result := range job.outChan {
			log.Printf("Outputlogger received %s -> %s", result.Key, string(result.Value))
			job.Results[result.Key] = result.Value
		}
		job.Complete <- true // indicates all results were received
		close(job.Complete)
	}()
}

// startOne starts a single worker, doesn't create goroutine
func (job *Job) startOne() {
	defer job.wg.Done()
	for {
		input, ok := <-job.inChan
		if !ok { // no more work to do
			break
		}
		// make http request to backend URL
		reply := Output{}
		reply.Key = input.Key

		client := &http.Client{}
		reader := bytes.NewReader(input.Value)
		req, errRequest := http.NewRequest("POST", job.workURL.String()+"/"+input.Key, reader)
		if errRequest != nil {
			log.Print(errRequest)
		}
		req.Header.Add("PMMAP-job", job.ID)
		req.Header.Add("PMMAP-auth", job.secretKey)
		res, errResponse := client.Do(req)
		if errResponse != nil {
			log.Print(errResponse)
		}
		defer res.Body.Close()
		// TODO handle each error case
		if errResponse != nil {
			input.RetryCount++
			job.inChan <- input
			// handle max retry count
		}
		reply.Value, _ = ioutil.ReadAll(res.Body)
		log.Printf("Received from server (key %s): %s", reply.Key, string(reply.Value))
		job.outChan <- reply
	}
}

// startCompletionWaiter runs a goroutine that's waiting until completion
func (job *Job) startCompletionWaiter() {
	go func() {
		job.wg.Wait()
		atomic.StoreInt64(&job.State, AllOutputReceived)

		close(job.outChan) // don't let anyone write to it anymore
	}()
}

// startWorkers starts workers, each in his goroutine
func (job *Job) startWorkers(concurrency int) {
	job.wg.Add(concurrency)
	for count := 0; count < concurrency; count++ {
		go func() {
			job.startOne()
		}()
	}
}

// canReceiveInput tell whether it's OK to accept new inputs
func (job *Job) canReceiveInput() bool {
	state := atomic.LoadInt64(&job.State)
	return state == Created || state == ReceivingInputs
}

// tells the job it's receiving inputs
func (job *Job) receiving(count int) {
	state := atomic.LoadInt64(&job.State)
	if state != Created && state != ReceivingInputs {
		log.Print("Job receiving inputs while not in right state")
		return
	}
	atomic.StoreInt64(&job.State, ReceivingInputs)
	atomic.AddInt64(&job.outputsCount, int64(count))
}

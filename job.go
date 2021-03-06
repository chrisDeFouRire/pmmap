package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"sync"
	"sync/atomic"
	"time"

	"github.com/satori/go.uuid"
	"github.com/syndtr/goleveldb/leveldb"
)

var dbPath = "./db/"

// OutputError is set on Output when an unrecuperable error has occurred
type OutputError struct {
	StatusCode int    `json:"statusCode"`
	Message    string `json:"message"`
	Body       string `json:"body"`
}

// Input is the data structure used as inputs to jobs
type Input struct {
	Key        string
	Value      []byte // TODO use interface{} instead?
	retryCount int
}

// Output encapsulates the output of jobs
type Output struct {
	Key   string
	Value []byte // TODO use interface{} instead?
	Error *OutputError
}

// Job encapsulate a single instance of a job
type Job struct {
	sync.Mutex
	ID           string          // the job ID
	Complete     chan bool       // true is sent upon completion
	secretKey    string          // the job secret key (sent to webhooks)
	workURL      url.URL         // the URL radix we send work to
	inChan       chan Input      // channel where input is sent
	outChan      chan Output     // channel where output is sent
	wg           *sync.WaitGroup // to synchronize workers
	inputsCount  int64           // counts inputs received
	outputsCount int64           // counts outputs received
	State        int64           // the state of the job
	outputsDB    *leveldb.DB     // the storage for outputs
}

// MarshalJSON gives a JSON representation of a Job
func (job *Job) MarshalJSON() ([]byte, error) {
	job.Lock()
	defer job.Unlock()
	return json.Marshal(&struct {
		ID           string `json:"id"`
		InputsCount  int    `json:"inputs"`
		OutputsCount int    `json:"outputs"`
		URL          string `json:"url"`
	}{
		job.ID,
		int(job.GetInputsCount()),
		int(job.GetOutputsCount()),
		job.workURL.String()})
}

// CreateJob creates a new Job, ready to start
// returns a job
// TODO add backend timeout
func CreateJob(secret string, u url.URL, maxsize uint) *Job {
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
		outputsDB: nil,
	}
	return job
}

// Start working goroutines
func (job *Job) Start(concurrency int) {
	// wait until all workers are done
	go job.startCompletionWaiter()

	// start Output receiver
	go job.startOutputLogger()

	// start all workers
	job.startWorkers(concurrency)
}

// AddInputsToJob adds more than one input to the job
func (job *Job) AddInputsToJob(inputs []Input) error {
	job.Lock()
	defer job.Unlock()

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
	array := make([]Input, 1)
	array[0] = Input{Key: key, Value: value}
	return job.AddInputsToJob(array)
}

// AllInputsWereSent is called when all inputs have been sent
// no more input can be sent
// the job can't become "complete" until this function is called
func (job *Job) AllInputsWereSent() error {
	state := atomic.LoadInt64(&job.State)
	if state != ReceivingInputs {
		close(job.inChan)
		atomic.StoreInt64(&job.State, ErrorState)
		return fmt.Errorf("Wrong state transition")
	}
	atomic.StoreInt64(&job.State, AllInputReceived)
	close(job.inChan) // close will trigger each worker goroutine's exit
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

// GetResult returns the result for a key after all outputs are received, or nil if not found
func (job *Job) GetResult(key string) []byte {
	if job.State != AllOutputReceived {
		return nil
	}
	value, err := job.outputsDB.Get([]byte(key), nil)
	if err != nil {
		return nil
	}
	return value
}

// GetResults returns all Outputs
func (job *Job) GetResults() ([]*Output, error) {
	if job.State != AllOutputReceived {
		return nil, fmt.Errorf("Can't get results before all outputs are received")
	}
	var result []*Output
	iter := job.outputsDB.NewIterator(nil, nil)
	for iter.Next() {
		value := iter.Value()
		dst := make([]byte, len(value))
		copy(dst, value)
		output := &Output{
			Key:   string(iter.Key()),
			Value: dst,
		}
		result = append(result, output)
	}
	iter.Release()
	err := iter.Error()
	return result, err
}

// startOutputLogger receives all outputs
func (job *Job) startOutputLogger() {
	var err error
	job.outputsDB, err = leveldb.OpenFile(dbPath+"/job/"+job.ID, nil)
	if err != nil {
		log.Fatal(err)
	}

	for result := range job.outChan {
		atomic.AddInt64(&job.outputsCount, int64(1))
		// TODO use a key prefix to differenciate from errors
		// TODO store errors too
		err = job.outputsDB.Put([]byte(result.Key), result.Value, nil)
	}
	job.Complete <- true // indicates all results were received, won't block
	close(job.Complete)
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
		reply := Output{Key: input.Key}

		client := &http.Client{
			Timeout: time.Second * 30, // TODO job param
		}
		bodyreader := bytes.NewReader(input.Value)
		req, errRequest := http.NewRequest("POST", job.workURL.String()+"/"+input.Key, bodyreader)
		if errRequest != nil {
			error := &OutputError{
				Message: "Can't create POST request to the backend endpoint",
			}
			reply.Error = error
			job.outChan <- reply
			continue
		}
		req.Header.Add("PMMAP-job", job.ID)
		req.Header.Add("PMMAP-auth", job.secretKey)
		req.Header.Add("Content-Type", "application/json")
		res, errResponse := client.Do(req)
		if errResponse != nil {
			input.retryCount++
			job.inChan <- input // TODO exponential backup
			continue
		}
		defer res.Body.Close()

		if res.StatusCode == http.StatusOK {
			var readerr error
			reply.Value, readerr = ioutil.ReadAll(res.Body)
			if readerr != nil {
				error := &OutputError{
					Message: "Can't read body from response",
				}
				reply.Error = error
				job.outChan <- reply
				continue
			}
			reply.Error = nil
			job.outChan <- reply
			continue
		}
		if res.StatusCode >= 500 || input.retryCount < 5 { // retryable error // TODO job param max retries
			input.retryCount++
			job.inChan <- input
		} else { // fatal error
			error := &OutputError{}
			error.StatusCode = res.StatusCode
			if b, berr := ioutil.ReadAll(res.Body); berr == nil {
				error.Body = string(b)
			}
			log.Printf("Backend replied %d for %s", res.StatusCode, input.Key)

			reply.Error = error
			reply.Value = nil
			job.outChan <- reply
		}
	}
}

// startCompletionWaiter runs a goroutine that's waiting until completion
func (job *Job) startCompletionWaiter() {
	job.wg.Wait()
	atomic.StoreInt64(&job.State, AllOutputReceived)

	close(job.outChan) // don't let anyone write to it anymore
}

// startWorkers starts workers, each in his goroutine
func (job *Job) startWorkers(concurrency int) {
	job.wg.Add(concurrency)
	// TODO add rampup
	for count := 0; count < concurrency; count++ {
		go job.startOne()
	}
}

// canReceiveInput tells whether it's OK to accept new inputs
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
	atomic.AddInt64(&job.inputsCount, int64(count))
}

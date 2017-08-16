package main

import (
	"sync"
)

// manager holds jobs
type manager struct {
	sync.RWMutex
	jobs map[string]*Job
}

// Manager is the entry point to jobs
var Manager = manager{jobs: make(map[string]*Job)}

func (man *manager) addJob(job *Job) {
	man.Lock()
	defer man.Unlock()
	man.jobs[job.ID] = job
}

func (man *manager) getJob(id string) *Job {
	man.RLock()
	defer man.RUnlock()
	return man.jobs[id]
}

func (man *manager) delJob(id string) {
	man.Lock()
	defer man.Unlock()
	delete(man.jobs, id)
}

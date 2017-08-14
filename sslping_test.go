package main

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"testing"
)

const (
	sslping = "https://sslping.com/api/v1/check"
)

func TestSSLPing(t *testing.T) {
	// Step 1: create job
	req := struct {
		Secret      string `json:"secret"`
		URL         string `json:"url"`
		Maxsize     int    `json:"maxsize"`
		Concurrency int    `json:"concurrency"`
	}{Secret, sslping, 10, 10}
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
	if m["url"].(string) != sslping {
		t.Fatalf("Job should report correct url")
	}

	// step 2: send input
	one := struct {
		Key   string      `json:"key"`
		Value interface{} `json:"value"`
	}{"sslping.com", ""}
	two := struct {
		Key   string      `json:"key"`
		Value interface{} `json:"value"`
	}{"hire.chris-hartwig.com", ""}
	var onearray = make([]interface{}, 2)
	onearray[0] = one
	onearray[1] = two
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
	if reserr != nil || res.StatusCode != http.StatusOK {
		log.Printf("Error while getting results (status= %d)", res.StatusCode)
		log.Fatal(reserr)
	}
	var r []struct {
		Key   string `json:"key"`
		Value struct {
			Messages []struct {
				Name string `json:"name"`
				Prio int    `json:"prio"`
			} `json:"messages"`
		} `json:"value"`
	}
	decodeerr := json.NewDecoder(res.Body).Decode(&r)
	if decodeerr != nil {
		log.Fatal(decodeerr)
	}
	if len(r) != 2 {
		log.Fatalf("There should have been two results, there were %d", len(r))
	}
	if r[0].Key != "hire.chris-hartwig.com" {
		log.Fatal("Wrong result[0] received ", r[0])
	}
	if r[1].Key != "sslping.com" {
		log.Fatal("Wrong result[1] received ", r[1])
	}
	_b, _ := json.Marshal(r)
	log.Print(string(_b))
}

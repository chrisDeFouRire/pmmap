package main

import (
	"log"
	"net/http"

	"github.com/gorilla/mux"
)

// Secret is sent back in webhooks to ensure origin
var Secret = "This is an extremely bad secret"

func main() {
	go func() { // start a dummy server
		srv := &http.Server{
			Addr: "localhost:7777",
		}
		r := mux.NewRouter()
		r.HandleFunc("work/{key}", func(w http.ResponseWriter, req *http.Request) {
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
		})
		r.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
			log.Fatalf("Incorrect URL: %s", req.RequestURI)
		})
		http.Handle("/", r)
		srv.ListenAndServe()
	}()

	panic(http.ListenAndServe("localhost:8080", routes()))
}

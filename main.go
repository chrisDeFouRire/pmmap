package main

import (
	"log"
	"net/http"
)

var (
	// ListenAddress specifies the address to listen to
	ListenAddress = "localhost:8080"
)

func main() {
	log.Print("Starting PMmap on ", ListenAddress)
	panic(http.ListenAndServe(ListenAddress, routes()))
}

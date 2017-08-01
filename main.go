package main

import (
	"net/http"
)

// Secret is sent back in webhooks to ensure origin
var Secret = "This is an extremely bad secret"

func main() {
	panic(http.ListenAndServe("localhost:8080", routes()))
}

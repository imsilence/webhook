package main

import (
	"fmt"
	"net/http"
	"time"
)

var Version string

func main() {
	addr := ":8888"
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "%s: %d", Version, time.Now().Unix())
	})
	http.ListenAndServe(addr, nil)
}

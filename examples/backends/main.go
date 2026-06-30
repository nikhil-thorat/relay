package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
)

func main() {
	port := flag.Int(
		"port",
		9001,
		"port to listen on",
	)

	id := flag.String(
		"id",
		"api_1",
		"backend identifier",
	)

	flag.Parse()

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("[%s] %s %s", *id, r.Method, r.URL.Path)
		fmt.Fprintf(w, "Hello from %s\n", *id)
	})

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})

	addr := fmt.Sprintf(":%d", *port)

	log.Printf("%s listening on %s", *id, addr)

	err := http.ListenAndServe(addr, nil)
	if err != nil {
		log.Fatal(err)
	}
}

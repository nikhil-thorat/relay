package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"
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

	quiet := flag.Bool(
		"quiet",
		false,
		"disable request logging",
	)

	delay := flag.Duration(
		"delay",
		0,
		"delay before each response",
	)

	bodySize := flag.Int(
		"body-size",
		0,
		"pad response body to this many bytes",
	)

	flag.Parse()

	body := fmt.Sprintf("Hello from %s\n", *id)
	if *bodySize > len(body) {
		body += strings.Repeat("x", *bodySize-len(body))
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if !*quiet {
			log.Printf("[%s] %s %s", *id, r.Method, r.URL.Path)
		}
		if *delay > 0 {
			time.Sleep(*delay)
		}
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		fmt.Fprint(w, body)
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

package main

import (
	"fmt"
	"html"
	"log"
	"net/http"
	"time"
)

func main() {
	que := make(chan any)
	go func() {
		defer close(que)

		for {
			select {
			case <-que:
				fmt.Println("do a heavy work...")
				time.Sleep(3 * time.Second)
				fmt.Println("done!")
			default:
			}
		}
	}()

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		que <- "hoge"
		time.Sleep(1 * time.Second)
		fmt.Fprintf(w, "Hello, %q", html.EscapeString(r.URL.Path))
	})

	log.Fatal(http.ListenAndServe(":8080", nil))
}

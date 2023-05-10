package main

import (
	"context"
	"fmt"
	"html"
	"log"
	"net/http"
	"time"
)

func main() {
	dowork := func(ctx context.Context, que <-chan any, pulseInterval time.Duration) <-chan any {
		ctx, cancel := context.WithCancel(ctx)
		defer cancel()

		heartbeat := make(chan any)
		go func() {
			defer close(heartbeat)
			pulse := time.Tick(pulseInterval)
			sendPulse := func() {
				select {
				case heartbeat <- struct{}{}:
				default:
				}
			}

			for {
				select {
				case <-pulse:
					sendPulse()
				case str := <-que:
					fmt.Printf("do a heavy work %s...\n", str)
					time.Sleep(3 * time.Second)
					fmt.Println("done!\n")
				default:
				}
			}
		}()

		return heartbeat
	}

	que := make(chan any)
	ctx := context.TODO()
	heartbeat := dowork(ctx, que, 1*time.Second)
	go func() {
		for {
			select {
			case <-heartbeat:
				fmt.Println("pulse")
			}
		}
	}()

	http.HandleFunc("/hoge", func(w http.ResponseWriter, r *http.Request) {
		que <- "hoge"
		fmt.Fprintf(w, "Hello, %q", html.EscapeString(r.URL.Path))
	})

	log.Fatal(http.ListenAndServe(":8080", nil))
}

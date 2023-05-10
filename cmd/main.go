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
	doWork := func(ctx context.Context, que <-chan any, pulseInterval time.Duration) <-chan any {
		heartbeat := make(chan any)
		go func() {
			defer close(heartbeat)

			pulse := time.Tick(pulseInterval)
			sendPulse := func() {
				select {
				case heartbeat <- struct{}{}:
				case <-ctx.Done():
					return
				default:
				}
			}

			// fmt.Println("worker: Hello! I'm irresponsible")
			// <-ctx.Done()
			// fmt.Println("worker: I'm halting")

			for {
				select {
				case <-pulse:
					sendPulse()
				case str := <-que:
					go func() {
						fmt.Printf("do a heavy work %s...\n", str)
						time.Sleep(3 * time.Second)
						fmt.Println("done!")
					}()
				case <-ctx.Done():
					return
				default:
				}
			}
		}()

		return heartbeat
	}

	type startGoroutineFn func(
		ctx context.Context,
		que <-chan any,
		pulseInterval time.Duration,
	) (heartbeat <-chan any)
	newObserver := func(timeout time.Duration, startGoroutine startGoroutineFn) startGoroutineFn {
		return func(
			ctx context.Context,
			que <-chan any,
			pulseInterval time.Duration,
		) <-chan any {
			heartbeat := make(chan any)
			go func() {
				defer close(heartbeat)

				var workerHeartbeat <-chan any
				var cancel func()
				startWorker := func() {
					_, cancel = context.WithCancel(ctx)
					workerHeartbeat = startGoroutine(ctx, que, timeout/2)
				}
				startWorker()
				pulse := time.Tick(pulseInterval)
			monitorLoop:
				for {
					timeoutSignal := time.After(timeout)

					for {
						select {
						case <-pulse:
							select {
							case heartbeat <- struct{}{}:
							default:
							}
						case <-workerHeartbeat:
							fmt.Println("worker: pulse")
							continue monitorLoop
						case <-timeoutSignal:
							log.Println("observer: worker unhealthy; restarting...")
							cancel()
							startWorker()
							continue monitorLoop
						case <-ctx.Done():
							return
						}
					}
				}
			}()

			return heartbeat
		}
	}

	que := make(chan any)
	ctx := context.TODO()
	ctx, cancel := context.WithCancel(ctx)
	doWorkWithObserver := newObserver(5*time.Second, doWork)

	// time.AfterFunc(9*time.Second, func() {
	// 	log.Println("main: halting observer and worker")
	// 	cancel()
	// })

	heartbeat := doWorkWithObserver(ctx, que, 1*time.Second)

	go func() {
		for {
			select {
			case <-heartbeat:
				fmt.Println("observer: pulse")
			case <-ctx.Done():
				log.Fatal("application is unhealthy!")
			}
		}
	}()

	http.HandleFunc("/hoge", func(w http.ResponseWriter, r *http.Request) {
		que <- "hoge"
		fmt.Fprintf(w, "Hello, %q", html.EscapeString(r.URL.Path))
	})

	log.Fatal(http.ListenAndServe(":8080", nil))
}

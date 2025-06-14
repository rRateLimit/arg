package main

import (
	"flag"
	"fmt"
	"sync"
	"time"
)

// RateLimiter implements a token bucket algorithm for rate limiting
type RateLimiter struct {
	rate       int           // tokens per second
	burst      int           // maximum number of tokens
	tokens     int           // current number of tokens
	lastUpdate time.Time     // last time tokens were updated
	mu         sync.Mutex    // mutex for thread safety
}

// NewRateLimiter creates a new rate limiter with the specified rate and burst size
func NewRateLimiter(rate, burst int) *RateLimiter {
	return &RateLimiter{
		rate:       rate,
		burst:      burst,
		tokens:     burst, // start with full bucket
		lastUpdate: time.Now(),
	}
}

// Allow checks if a request can be processed and consumes a token if available
func (rl *RateLimiter) Allow() bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	// Calculate tokens to add based on elapsed time
	now := time.Now()
	elapsed := now.Sub(rl.lastUpdate)
	rl.lastUpdate = now

	// Add tokens based on rate and elapsed time
	tokensToAdd := int(elapsed.Seconds() * float64(rl.rate))
	rl.tokens = min(rl.tokens+tokensToAdd, rl.burst)

	// Check if we have tokens available
	if rl.tokens > 0 {
		rl.tokens--
		return true
	}
	return false
}

// Wait blocks until a token is available
func (rl *RateLimiter) Wait() {
	for !rl.Allow() {
		// Sleep for approximately the time it takes to generate one token
		time.Sleep(time.Duration(1000/rl.rate) * time.Millisecond)
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func main() {
	rate := flag.Int("rate", 10, "Rate limit (requests per second)")
	burst := flag.Int("burst", 20, "Burst size (maximum tokens)")
	requests := flag.Int("requests", 50, "Number of requests to simulate")
	workers := flag.Int("workers", 5, "Number of concurrent workers")
	
	flag.Parse()

	rl := NewRateLimiter(*rate, *burst)

	fmt.Printf("Rate Limiter Configuration:\n")
	fmt.Printf("- Rate: %d requests/second\n", *rate)
	fmt.Printf("- Burst: %d tokens\n", *burst)
	fmt.Printf("- Simulating %d requests with %d workers\n\n", *requests, *workers)

	var wg sync.WaitGroup
	requestChan := make(chan int, *requests)
	
	for i := 1; i <= *requests; i++ {
		requestChan <- i
	}
	close(requestChan)

	startTime := time.Now()

	for i := 0; i < *workers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for reqID := range requestChan {
				rl.Wait()
				fmt.Printf("Worker %d: Processing request %d at %s\n", 
					workerID, reqID, time.Now().Format("15:04:05.000"))
			}
		}(i)
	}

	wg.Wait()
	elapsed := time.Since(startTime)
	
	fmt.Printf("\nCompleted %d requests in %v\n", *requests, elapsed)
	fmt.Printf("Actual rate: %.2f requests/second\n", float64(*requests)/elapsed.Seconds())
}
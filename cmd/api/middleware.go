package main

import (
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// Middleware http.Handler to log and recover gracefully from any upstream application panics
func (app *application) recoverPanic(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Create a deferred function which will always be run in the event of panic
		// as Go unwinds the stack.
		defer func() {
			// Use the builtin recover function to check if there has been a panic or not.
			if err := recover(); err != nil {
				w.Header().Set("Connection", "close")

				app.serverErrorResponse(w, r, fmt.Errorf("%s", err))
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// Middleware http.Handler that implements a bucket token rate-limiter pattern.
func (app *application) rateLimit(next http.Handler) http.Handler {
	// Define a client struct to hold the rate limiter and last seen time for each client
	type client struct {
		limiter  *rate.Limiter
		lastSeen time.Time
	}

	// Declare a mutex and a map to hold the clients' IP addresses and client struct.
	var (
		mu      sync.Mutex
		clients = make(map[string]*client)
	)

	// Launch a background goroutine which removes old entries from the clients map once
	// every minute.
	go func() {
		for {
			time.Sleep(time.Minute)

			// Lock the mutex to prevent any rate limiter checks from happening while
			// the cleaning is taking place
			mu.Lock()

			// Loop through all the clients.  If they haven't been seen within the last three
			// minutes, delete the corresponding entry from the map.
			for ip, client := range clients {
				if time.Since(client.lastSeen) > 3*time.Minute {
					delete(clients, ip)
				}
			}
			// Importantly, unlock the mutex when the cleanup is complete
			mu.Unlock()
		}
	}()

	// The function we are returning is a closure which 'closes over' the client struct and
	// limiter variable.
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Only carry out the check if rate limiting is enabled.
		if app.config.limiter.enabled {

			// Extract the clients IP address from the request
			ip, _, err := net.SplitHostPort(r.RemoteAddr)
			if err != nil {
				app.serverErrorResponse(w, r, err)
				return
			}

			// Lock the mutex to prevent this code from being executed concurrently.
			mu.Lock()

			// Check to see if the IP address already exists in the map.  If it doesn't, then
			// initialise a new rate limiter and add the IP address and limiter to the map.
			if _, found := clients[ip]; !found {
				// Create a new client struct to the map if it doesn't exist.
				clients[ip] = &client{limiter: rate.NewLimiter(rate.Limit(app.config.limiter.rps), app.config.limiter.burst)}
			}

			// Update the last seen time for the client.
			clients[ip].lastSeen = time.Now()

			// Call limiter.Allow() on the rate limiter for the current IP address.  If the
			// request isn't allowed, unlock the mutex and send a 429 Too Many Requests
			// Response.
			if !clients[ip].limiter.Allow() {
				mu.Unlock()
				app.rateLimitExceededResponse(w, r)
				return
			}

			// Very importantly, unlock the mutex before calling the next handler in the
			// chain.  Notice that we DON'T use defer to unlock the mutex, as that would mean
			// that the mutex isnt unlocked until all the handlers downstream of this
			// middleware have also returned.
			mu.Unlock()
		}

		next.ServeHTTP(w, r)
	})
}

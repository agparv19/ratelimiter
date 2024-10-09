package main

import (
	"codemill/go/ratelimiter/fixedwindow"
	"codemill/go/ratelimiter/slidingwindow"
	"codemill/go/ratelimiter/tokenbucket"
	"flag"
	"fmt"
	"log"
	"net/http"
	"strings"
)

// Problem statement: https://codingchallenges.fyi/challenges/challenge-rate-limiter

type rateLimiter interface {
	// returns true if the given ip has exceeded
	// rate limit
	LimitExceeded(ip string) bool
}

var myRateLimiter rateLimiter

func handler(w http.ResponseWriter, r *http.Request) {

	request := r.URL.Path[1:] // remove '/' prefix
	if request == "unlimited" {
		fmt.Fprintf(w, "Unlimited! Let's Go!\n")
	} else if request == "limited" {

		// RemoteAddr contains host:port
		ip := r.RemoteAddr
		lastColonIndex := strings.LastIndex(ip, ":")
		if lastColonIndex < 0 {
			panic("malformed client ip!")
		}
		ip = ip[:lastColonIndex]

		limitExceeded := myRateLimiter.LimitExceeded(ip)

		if !limitExceeded {
			fmt.Fprintf(w, "Limited, don't over use me!\n")
		} else {
			w.WriteHeader(http.StatusTooManyRequests)
		}
	}
}

func main() {

	limiterAlgo := flag.String("limiterAlgo", "", "The rate limiting algorithm to use")
	flag.Parse()

	if *limiterAlgo == "tokenbucket" {
		myRateLimiter = tokenbucket.NewRateLimiter()
	} else if *limiterAlgo == "fixedwindow" {
		myRateLimiter = fixedwindow.NewRateLimiter()
	} else if *limiterAlgo == "slidingwindow" {
		myRateLimiter = slidingwindow.NewRateLimiter()
	} else {
		log.Fatalf("Unsupported rate limiter algorithm: %v", *limiterAlgo)
	}

	http.HandleFunc("/", handler)
	log.Fatal(http.ListenAndServe(":8080", nil))
}

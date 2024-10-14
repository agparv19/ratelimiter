package tokenbucket

import (
	"log"
	"sync"
	"time"
)

// thread-safe bucket of tokens
type tokenBucket struct {
	// mutex to synchronize
	// this can also be per ip,
	// but i am just using one for all
	mu sync.Mutex

	// ip -> # of tokens left
	buckets map[string]int
}

type RateLimiter struct {

	// global token bucket
	myTokenBucket tokenBucket

	// max capacity of each bucket (per IP address)
	tokenCap int

	// in seconds, the time period of bucket refresh
	refreshPeriod time.Duration
}

func NewRateLimiter() *RateLimiter {

	rl := &RateLimiter{tokenCap: 10,
		refreshPeriod: 1 * time.Second,
		myTokenBucket: tokenBucket{buckets: make(map[string]int)}}

	// launch a goroutine to keep refreshing
	// buckets with new token, in the background
	go RefreshBucket(rl)

	// return our rate limiter instance
	return rl
}

// This function adds one new token for each
// known ip for every refreshPeriod
func RefreshBucket(rl *RateLimiter) {
	for {
		rl.myTokenBucket.mu.Lock()
		for key, value := range rl.myTokenBucket.buckets {
			if value > rl.tokenCap {
				panic("Too many tokens, something is wrong!")
			} else if value < rl.tokenCap {
				rl.myTokenBucket.buckets[key] = value + 1
			}
			// don't do anything if bucket is full
		}
		rl.myTokenBucket.mu.Unlock()
		time.Sleep(rl.refreshPeriod)
		// note the time period of referesh is techincally
		// refreshPeriod + time taken to execute above code
	}
}

func (rl *RateLimiter) LimitExceeded(ip string) bool {
	limitExceeded := false
	rl.myTokenBucket.mu.Lock()
	nTokens, exist := rl.myTokenBucket.buckets[ip]
	if !exist {
		// create a bucket, give tokenCap tokens,
		// take 1 from it process current request
		rl.myTokenBucket.buckets[ip] = rl.tokenCap - 1
	} else if nTokens > 0 {
		// tokens available, take one
		rl.myTokenBucket.buckets[ip] = nTokens - 1
	} else {
		// no more tokens available, can not
		// process this request!
		log.Printf("Rejecting request from %v, received %v requests already!\n", ip, rl.tokenCap)
		limitExceeded = true
	}
	rl.myTokenBucket.mu.Unlock()
	return limitExceeded
}

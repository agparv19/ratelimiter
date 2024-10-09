package slidingwindow

import (
	"log"
	"sync"
	"time"
)

type slidingWindow struct {
	mu   sync.Mutex
	data map[string][]time.Time
}

type RateLimiter struct {
	// thread-safe data structure
	// to store traffic data of all
	// incoming IPs
	myWindow slidingWindow

	// size of window on time axis
	windowSize time.Duration

	// max number of requests allowed
	// within one window per ip
	maxReqPerWindow int
}

func NewRateLimiter() *RateLimiter {

	rl := &RateLimiter{myWindow: slidingWindow{data: make(map[string][]time.Time)},
		windowSize:      10 * time.Second,
		maxReqPerWindow: 20}
	return rl
}

func (rl *RateLimiter) LimitExceeded(ip string) bool {

	currentTime := time.Now()

	rl.myWindow.mu.Lock()
	defer rl.myWindow.mu.Unlock()

	ipwindow, exists := rl.myWindow.data[ip]

	var limitExceeded bool

	if !exists {
		// no window exists for this ip yet
		// create one, and add current time stamp
		ipwindow = append(ipwindow, currentTime)
		// assumes maxReqPerWindow >= 1
		limitExceeded = false
	} else {
		thresh := currentTime.Add(-1 * rl.windowSize)
		// all entries in ipwindow that are older than
		// thresh, should be discarded
		idx := len(ipwindow) - 1

		// search for a time just >= thresh
		// this can be improved with binary search
		for i, t := range ipwindow {
			if t.After(thresh) || t.Equal(thresh) {
				idx = i
				break
			}
		}
		if idx >= 0 {
			// this check is important as
			// ipwindow can be empty slice as well
			ipwindow = ipwindow[idx:]
		}

		if len(ipwindow) >= rl.maxReqPerWindow {
			// reject this request
			log.Printf("Rejecting request from %v, received %v requests already in current window!\n", ip, len(ipwindow))
			limitExceeded = true
		} else {
			ipwindow = append(ipwindow, currentTime)
			limitExceeded = false
		}
	}
	rl.myWindow.data[ip] = ipwindow
	return limitExceeded
}

package fixedwindow

import (
	"log"
	"sync"
	"time"
)

// this struct contains traffic
// info for a given IP Addr
type trafficData struct {
	count      int
	start_time time.Time
}

type window struct {
	mu sync.Mutex

	// ip -> trafficData
	data map[string]trafficData
}

type RateLimiter struct {
	// thread-safe data structure
	// to store traffic data of all
	// incoming IPs
	myWindow window

	// size of window on time axis
	windowSize time.Duration

	// max number of requests allowed
	// within one window per ip
	maxReqPerWindow int
}

func NewRateLimiter() *RateLimiter {
	rl := &RateLimiter{maxReqPerWindow: 60,
		windowSize: 1 * time.Minute,
		myWindow:   window{data: make(map[string]trafficData)}}

	return rl
}

func (rl *RateLimiter) LimitExceeded(ip string) bool {
	currentTime := time.Now()
	flooredTime := currentTime.Truncate(rl.windowSize)

	rl.myWindow.mu.Lock()
	defer rl.myWindow.mu.Unlock()
	td, exists := rl.myWindow.data[ip]
	if exists {
		if flooredTime == td.start_time {
			// same window
			if td.count >= rl.maxReqPerWindow {
				// to many requests handeled in this
				// window already, reject this
				log.Printf("Rejecting request from %v, received %v requests already in current window!\n", ip, td.count)
				return true
			} else {
				td.count++
				// update the value in map.
				// I know copying isn't cool,
				// but hey, it's pretty cheap here!
				rl.myWindow.data[ip] = td
				return false
			}
		} else {
			// new window
			td.count = 1
			td.start_time = flooredTime
			rl.myWindow.data[ip] = td
			return false
		}
	} else {
		// assign count of 1 for this request
		// assign current (floored) time as the window's start_time
		rl.myWindow.data[ip] = trafficData{count: 1, start_time: flooredTime}

		// this assumes maxReqPerWindow >= 1
		return false
	}
}

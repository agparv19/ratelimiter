package swcounter

import (
	"log"
	"math"
	"sync"
	"time"
)

// this struct contains traffic
// info for a given IP Addr
type trafficData struct {
	// # of requests processed
	// in the current window
	curr_count int

	// # of requests processed
	// in the previous window
	prev_count int

	// start time of the current
	// fixed window
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
		windowSize: 30 * time.Second,
		myWindow:   window{data: make(map[string]trafficData)}}

	return rl
}

func (rl *RateLimiter) LimitExceeded(ip string) bool {

	rl.myWindow.mu.Lock()
	defer rl.myWindow.mu.Unlock()

	currentTime := time.Now()
	flooredTime := currentTime.Truncate(rl.windowSize)

	td, exists := rl.myWindow.data[ip]

	// this counter stores the sliding count i.e.
	// a fraction from current window, and another
	// from previous window
	sliding_count := float64(0)

	if !exists {

		td = trafficData{prev_count: 0, curr_count: 1, start_time: flooredTime}
		rl.myWindow.data[ip] = td
		// assume the new request is allowed i.e. rl.maxReqPerWindow >= 1
		return false
	} else {
		if flooredTime == td.start_time {
			// same window
			// fraction of curr window completed
			f := float64(currentTime.Sub(td.start_time).Milliseconds()) / float64(rl.windowSize.Milliseconds())

			if f > 1 || f < 0 {
				panic("fraction is outside [0, 1] !!")
			}

			sliding_count = float64(td.curr_count) + float64(td.prev_count)*(1-f)

			if int(math.Floor(sliding_count)) >= rl.maxReqPerWindow {
				// limit exceeded!
				log.Printf("Rejecting request from %v, received %v requests already in current window!\n", ip, int(sliding_count))
				return true
			} else {
				// not exceeded the limit yet
				td.curr_count++
				rl.myWindow.data[ip] = td
				return false
			}
		} else {

			// a new window
			// check if the stored window is previous window
			curr_window_start := flooredTime
			prev_window_start := flooredTime.Add(-1 * rl.windowSize)

			if td.start_time == prev_window_start {

				// td has data for previous window, but not current.
				// note that td.start_time is prev_window_start which
				// mean td.curr_count in this context is actuall prev window count

				// fraction of curr window completed
				f := float64(currentTime.Sub(curr_window_start).Milliseconds()) / float64(rl.windowSize.Milliseconds())

				if f > 1 || f < 0 {
					panic("fraction is outside [0, 1] !!")
				}

				sliding_count = float64(td.curr_count) * (1 - f)

				if int(math.Floor(sliding_count)) >= rl.maxReqPerWindow {
					// limit exceeded!
					log.Printf("Rejecting request from %v, received %v requests already in current window!\n", ip, int(sliding_count))
					return true
				} else {
					td.prev_count = td.curr_count
					td.curr_count = 1
					td.start_time = curr_window_start
					rl.myWindow.data[ip] = td
					return false
				}

			} else {
				// the data in td is neither current nor previous
				// re-initialize td
				td = trafficData{prev_count: 0, curr_count: 1, start_time: flooredTime}
				rl.myWindow.data[ip] = td
				return false
			}
		}
	}
}

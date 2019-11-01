package limiter

import (
	"fmt"
	"time"
)

// FpsLimiter will trigger every frames per second
type FpsLimiter struct {
	framesPerSecond int
	secondsPerFrame time.Duration

	tick chan bool
}

// NewFPSLimiter is the preferred method of initialisation for FpsLimiter type
func NewFPSLimiter(framesPerSecond int) (*FpsLimiter, error) {
	lim := new(FpsLimiter)
	lim.SetLimit(framesPerSecond)

	lim.tick = make(chan bool)

	// run ticker concurrently
	go func() {
		adjustedSecondPerFrame := lim.secondsPerFrame
		t := time.Now()
		for {
			time.Sleep(adjustedSecondPerFrame)
			nt := time.Now()
			lim.tick <- true
			adjustedSecondPerFrame -= nt.Sub(t) - lim.secondsPerFrame
			if adjustedSecondPerFrame < 0 {
				adjustedSecondPerFrame = 0
			}
			t = nt
		}
	}()

	return lim, nil
}

// SetLimit defines how frame limiter rate
func (lim *FpsLimiter) SetLimit(framesPerSecond int) {
	lim.framesPerSecond = framesPerSecond
	lim.secondsPerFrame, _ = time.ParseDuration(fmt.Sprintf("%fs", float64(1.0)/float64(framesPerSecond)))
}

// Wait will block until trigger
func (lim *FpsLimiter) Wait() {
	<-lim.tick
}

// HasWaited will return true if time has already elapsed and false it it is
// still yet to happen
func (lim *FpsLimiter) HasWaited() bool {
	select {
	case <-lim.tick:
		return true
	default:
		return false
	}
}

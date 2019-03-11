package sdl

import (
	"fmt"
	"time"
)

type fpsLimiter struct {
	framesPerSecond int
	secondsPerFrame time.Duration

	tick chan bool
}

func newFPSLimiter(framesPerSecond int) (*fpsLimiter, error) {
	var err error

	lim := new(fpsLimiter)

	lim.framesPerSecond = framesPerSecond
	lim.secondsPerFrame, err = time.ParseDuration(fmt.Sprintf("%fs", float64(1.0)/float64(framesPerSecond)))
	if err != nil {
		return nil, err
	}

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
			t = nt
		}
	}()

	return lim, nil
}
func (lim *fpsLimiter) wait() {
	<-lim.tick
}

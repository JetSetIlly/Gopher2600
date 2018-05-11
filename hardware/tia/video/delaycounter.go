package video

import "fmt"

// delayCounter is a general purpose counter that can be labelled. it is used
// in the sprite class and for delaying enabling signals
type delayCounter struct {
	count int
	Value interface{}

	// because we use the delayCounter type in more than one context we need some
	// way of providing the String() output with a helpful label
	label string
}

// newDelayCounter is the preferred method of initialisation for DelayCounter
func newDelayCounter(label string) *delayCounter {
	dc := new(delayCounter)
	if dc == nil {
		return nil
	}
	dc.count = -1
	dc.Value = true
	dc.label = label
	return dc
}

func (dc delayCounter) String() string {
	if dc.isRunning() {
		return fmt.Sprintf(" %s in %d cycles(s)", dc.label, dc.count)
	}
	return fmt.Sprintf(" [no %s pending]", dc.label)
}

// set the amount of delay and the delayed value
func (dc *delayCounter) set(count int, value interface{}) {
	dc.count = count
	dc.Value = value
}

// isRunning returns true if delay counter is still running
func (dc delayCounter) isRunning() bool {
	return dc.count > -1
}

// tick moves the delay counter on one step
func (dc *delayCounter) tick() bool {
	if dc.count == 0 {
		dc.count--
		return true
	}

	if dc.isRunning() {
		dc.count--
	}
	return false
}

package future

import "fmt"

// Event represents a single occurence (contained in payload) that is to be
// deployed in the future
type Event struct {
	// the future ticker this event belongs to
	ticker *Ticker

	// label is a short decription describing the future payload
	label string

	// the number of cycles the event began with
	initialCycles int

	// the number of remaining ticks before the pending action is resolved
	RemainingCycles int

	// the value that is to be the result of the pending action
	payload func()

	// arguments to the payload function
	args []interface{}
}

func (ins Event) String() string {
	return fmt.Sprintf("%s -> %d", ins.label, ins.RemainingCycles)
}

func schedule(ticker *Ticker, cycles int, payload func(), label string) *Event {
	return &Event{ticker: ticker, label: label, initialCycles: cycles, RemainingCycles: cycles, payload: payload}
}

func (ins *Event) tick() bool {
	// 0 is the trigger state
	if ins.RemainingCycles == 0 {
		ins.RemainingCycles--
		ins.payload()
		return true
	}

	// -1 is the off state
	if ins.RemainingCycles != -1 {
		ins.RemainingCycles--
	}

	return false
}

// Force can be used to immediately run the event's payload
func (ins *Event) Force() {
	ins.payload()
	ins.ticker.Drop(ins)
}

// Drop can be used to remove the event from the ticker queue without running
// the payload
func (ins *Event) Drop() {
	ins.ticker.Drop(ins)
}

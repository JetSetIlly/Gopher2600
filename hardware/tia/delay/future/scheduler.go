package future

// Scheduler exposes only the Schedule() function
type Scheduler interface {
	Schedule(cycles int, payload func(), label string) *Event
}

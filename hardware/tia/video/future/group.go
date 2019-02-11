package future

import (
	"container/list"
)

// Group is used to buffer payloads for future triggering.
type Group struct {
	instances list.List
}

// MachineInfo returns future group information in verbose format
func (fut Group) MachineInfo() string {
	return "not implemented"
}

// MachineInfoTerse returns future group information in terse format
func (fut Group) MachineInfoTerse() string {
	return "not implemented"
}

// Schedule the pending future action
func (fut *Group) Schedule(cycles int, payload func(), label string) *Instance {
	ins := schedule(fut, cycles, payload, label)
	fut.instances.PushBack(ins)
	return ins
}

// IsScheduled returns true if there are any outstanding future instances
func (fut Group) IsScheduled() bool {
	return fut.instances.Len() > 0
}

// Tick moves the pending action counter on one step
func (fut *Group) Tick() bool {
	r := false

	e := fut.instances.Front()
	for e != nil {
		t := e.Value.(*Instance).tick()
		r = r || t

		if t {
			f := e.Next()
			fut.instances.Remove(e)
			e = f
		} else {
			e = e.Next()
		}
	}

	return r
}

// Force can be used to immediately run the future payload
func (fut *Group) Force(ins *Instance) {
	e := fut.instances.Front()
	for e != nil {
		i := e.Value.(*Instance)
		if i == ins {
			i.payload()
			fut.instances.Remove(e)
			break
		} else {
			e = e.Next()
		}
	}
}

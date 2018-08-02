package memory

type periphPayload struct {
	address uint16
	data    uint8
}

const periphQueueLen = 10

// PeriphWrite writes the data to the memory area's address specified by
// registerName - not directly, but to the periphQueue
func (area *ChipMemory) PeriphWrite(address uint16, data uint8) {
	loop := true
	for loop {
		loop = false
		select {
		case area.periphQueue <- &periphPayload{address: address, data: data}:
		default:
			// else remove an item from the queue (making some room) and try
			// again. we do this because we want the most recent inputs to take
			// priority
			<-area.periphQueue
			loop = true
		}
	}
}

// we resolve all pending peripheral input whenever we get the chance. in
// practice this means before reading or writing memory from a non-peripheral
// bus
func (area *ChipMemory) resolvePeriphQueue() {
	loop := true
	for loop {
		select {
		case cw := <-area.periphQueue:
			area.ChipWrite(cw.address, cw.data)
		default:
			loop = false
		}
	}
}

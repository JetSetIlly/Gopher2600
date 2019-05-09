package peripherals

// Controller implementations feed controller Events to the peripheral to which
// it is attached.
//
// Peripherals can also be controlled more directly by calling the Handle
// function of that peripheral.
type Controller interface {
	GetInput(id PeriphID) (Event, error)
}

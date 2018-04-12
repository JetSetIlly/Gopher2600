package memory

// Memory defines the operations for a memory system
type Memory interface {
	Read(address uint16) (uint8, error)
	Write(address uint16, data uint8) error
}

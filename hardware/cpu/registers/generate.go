package registers

import (
	"headlessVCS/hardware/cpu/registers/r16bit"
	"headlessVCS/hardware/cpu/registers/rbits"
)

// Register defines the minumum implementation for a register type. mutating
// methods are not included because of the limitation in the language with
// regard to implementing an interface with pointer receivers -- I'm not keen
// on the idea on receiving and returning new values
type Register interface {
	Size() int
	Label() string
	ToBits() string
	ToHex() string
	ToUint() uint
	ToUint16() uint16
	IsNegative() bool
	IsZero() bool
}

// Generate calls the appropriate implementation's Generate function
func Generate(v interface{}, bitlen int, label string) (Register, error) {
	if bitlen == 16 {
		return r16bit.Generate(v, bitlen, label)
	}

	return rbits.Generate(v, bitlen, label)
}

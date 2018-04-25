package registers

import (
	"headlessVCS/hardware/cpu/registers/r16bit"
	"headlessVCS/hardware/cpu/registers/rbits"
)

// Generate calls the appropriate implementation's Generate function
func Generate(v interface{}, bitlen int) (interface{}, error) {
	if bitlen == 16 {
		return r16bit.Generate(v, bitlen)
	}

	return rbits.Generate(v, bitlen)
}

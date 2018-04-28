package rbits

import "fmt"

// implementing bit as a simple boolean
type bit bool

// Register is an array of of type bit, used for register representation
type Register struct {
	value []bit
	label string
}

// Size returns the number of bits in register
func (r Register) Size() int {
	return len(r.value)
}

// Label returns the label assigned to the register
func (r Register) Label() string {
	return r.label
}

// IsNegative checks the sign bit of the register
func (r Register) IsNegative() bool {
	return bool(r.value[0])
}

// IsZero checks if register is zero
func (r Register) IsZero() bool {
	for b := range r.value {
		if r.value[b] == true {
			return false
		}
	}
	return true
}

// IsOverflow checks the 'overflow' bit of the register
func (r Register) IsOverflow() bool {
	return bool(r.value[1])
}

func (r Register) String() string {
	return fmt.Sprintf("%s: %d [%s] %s", r.label, r.ToUint(), r.ToHex(), r.ToBits())
}

// ToString returns the string representation of an aribtrary value
func (r Register) ToString(v interface{}) string {
	vr, err := Generate(v, r.Size(), r.Label())
	if err != nil {
		return err.Error()
	}
	return fmt.Sprintf("%v", vr)
}

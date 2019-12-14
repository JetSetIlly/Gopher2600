// Package registers implements the three types of registers found in the 6507.
// The three types are the: program counter, status register and the 8 bit
// accumulator type used for A, X, Y.
//
// The 8 bit registers implemented as the Register type, define all the basic
// operations available to the 6507: load, add, subtract, logical operations and
// shifts/rotates. In addition it implements the tests required for status
// updates: is the value zero, is the number negative or is the overflow bit
// set.
//
// The program counter by comparison is 16 bits wide and defines only the load
// and add operations.
//
// The status register is implemented as a series of flags. Setting of flags
// is done directly. For instance, in the CPU, we might have this sequence of
// function calls:
//
//	a.Load(10)
//	a.Subtract(11)
//	sr.Zero = a.IsZero()
//
// In this case, the zero flag in the status register will be false.
package registers

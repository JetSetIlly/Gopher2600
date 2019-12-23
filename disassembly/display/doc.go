// Package display facilitates the presentation of disassembled ROMs.
//
// The Instruction type stores the formatted parts of an individual
// disassembled instruction. Instruction should be instantiated with the
// Format command(). The Format() command takes an instance of execution.Result
// and annotates it for easy reading.
//
// The actual presentation of formatted results to the user is outside of the
// scope of this package but the Columns type is intended to help. The Update()
// function should be used to ensure that column widths are enough for all
// instances in a group of Instructions.
package display

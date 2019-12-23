// Package execution tracks the result of instruction execution on the CPU.
// The Result type stores detailed information about each instruction
// encountered during a program execution on the CPU. A Result can then be used
// to produce output for disassemblers and debuggers with the help of the
// disassembly package.
//
// The Result.IsValid() function can be used to check whether results are
// consistent with the instruction definition. The CPU pcakage doesn't call
// this function because it would introduce unwanted performance penalties, but
// it's probably okay to use in a debugging context.
package execution

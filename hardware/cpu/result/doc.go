// Package result handles storage and presentation of CPU instruction results.
// The main product of this package is the Instruction type.
//
// The Instruction type is used by the CPU to store detailed information about
// each instruction executed. This type can then be used to produce output for
// disassemblers and debuggers with the GetString() function. The Style type
// helps the user of the package to control how the string is constructed.
//
// The Instruction.IsValid() function can be used to check whether the results
// CPU execution, as stored in the Insruction type, is consistent with the
// instruction definition. For example, the PageFault field is set to true:
// does the instruction definition suggest that this instruction can even cause
// page faults under certain conditions.
package result

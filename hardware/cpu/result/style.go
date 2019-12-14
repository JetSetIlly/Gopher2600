package result

// Style specifies the elements to include in strings constructed by
// Instruction.GetString(). Different styles can be ORed together to produce
// custom styles.
type Style int

// List of valid Style flags.
const (
	// ByteCode flag causes the program data to be printed verbatim before
	// the disassembly
	StyleFlagByteCode Style = 0x01 << iota

	// specifying StyleFlagSymbols or StyleFlagLocation has no effect if no
	// symbols type instance is available
	StyleFlagAddress
	StyleFlagSymbols
	StyleFlagLocation

	// the number of cycles consumed by the instruction
	StyleFlagCycles

	// include any useful notes about the disassembly. for example, whether a
	// page-fault occurred
	StyleFlagNotes

	// force output into columns of suitable width
	StyleFlagColumns

	// remove leading/trailing whitespace
	StyleFlagCompact
)

// List of compound styles. For convenience.
const (
	StyleExecution = StyleFlagAddress | StyleFlagSymbols | StyleFlagLocation | StyleFlagCycles | StyleFlagNotes | StyleFlagColumns
	StyleDisasm    = StyleFlagByteCode | StyleFlagAddress | StyleFlagSymbols | StyleFlagLocation | StyleFlagColumns
	StyleBrief     = StyleFlagSymbols | StyleFlagCompact
)

// Has tests to see if style has the supplied flag in its definition
func (style Style) Has(flag Style) bool {
	return style&flag == flag
}

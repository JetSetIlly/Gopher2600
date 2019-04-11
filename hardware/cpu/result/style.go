package result

// Style is the type used to specify what to include in a disassembly string
type Style int

// style flags to hint at what to include when creating disassembly output
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

	// compound style
	StyleExecution = StyleFlagAddress | StyleFlagSymbols | StyleFlagLocation | StyleFlagCycles | StyleFlagNotes | StyleFlagColumns
	StyleDisasm    = StyleFlagByteCode | StyleFlagAddress | StyleFlagSymbols | StyleFlagLocation | StyleFlagColumns
)

// Has tests to see if style has the supplied flag in its definition
func (style Style) Has(flag Style) bool {
	return style&flag == flag
}

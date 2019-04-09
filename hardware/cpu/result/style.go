package result

// Style is the type used to specify what to include in a disassembly string
type Style int

// style flags to hint at what to include when creating disassembly output
const (
	// specifying StyleFlagSymbols or StyleFlagLocation has no effect if no
	// symbols type instance is available
	StyleFlagAddress Style = 0x01 << iota
	StyleFlagSymbols
	StyleFlagLocation

	// StyleFlagColumns forces output into columns of suitable width
	StyleFlagColumns

	// include any useful notes about the disassembly. for example, whether a
	// page-fault occured
	StyleFlagNotes

	// ByteCode flag causes the program data to be printed verbatim before
	// the disassembly
	StyleFlagByteCode
)

// compound styles
const (
	StyleBrief = StyleFlagAddress | StyleFlagSymbols
	StyleFull  = StyleFlagAddress | StyleFlagSymbols | StyleFlagLocation | StyleFlagColumns | StyleFlagNotes
)

// Has tests to see if style has the supplied flag in its definition
func (style Style) Has(flag Style) bool {
	return style&flag == flag
}

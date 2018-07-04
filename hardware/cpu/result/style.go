package result

// Style is the type used to specify what to include in a disassembly string
type Style int

// style flags to hint at what to include when creating disassembly output
const (
	// specifying StyleFlagSymbols or StyleFlagLocation has no effect if no
	// symbols type instance is available
	StyleFlagSymbols Style = 0x01 << iota
	StyleFlagLocation

	StyleFlagHex
	StyleFlagColumns
	StyleFlagNotes
)

// compound styles
const (
	StyleBrief = StyleFlagSymbols
	StyleFull  = StyleFlagSymbols | StyleFlagLocation | StyleFlagColumns | StyleFlagNotes
)

// Has tests to see if style has the supplied flag in its definition
func (style Style) Has(flag Style) bool {
	return style&flag == flag
}

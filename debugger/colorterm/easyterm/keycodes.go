package easyterm

// list of ASCII codes for non-alphanumeric characters
const (
	KeyCtrlC          = 3
	KeyTab            = 9
	KeyCarriageReturn = 13
	KeyEsc            = 27
	KeyBackspace      = 127
)

// list of ASCII code for characters that can follow KeyEsc
const (
	EscDelete = 51
	EscCursor = 91
	EscHome   = 72
	EscEnd    = 70
)

// list of ASCII code for characters that can follow EscCursor
const (
	CursorUp       = 'A'
	CursorDown     = 'B'
	CursorForward  = 'C'
	CursorBackward = 'D'
)

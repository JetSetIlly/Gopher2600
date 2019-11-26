package easyterm

// list of ASCII codes for non-alphanumeric characters
const (
	KeyInterrupt      = 3  // end-of-text character
	KeySuspend        = 26 // substitute character
	KeyTab            = 9
	KeyCarriageReturn = 13
	KeyEsc            = 27
	KeyBackspace      = 8
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

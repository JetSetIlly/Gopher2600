package ansi

import (
	"fmt"
	"strings"
)

// ansi color
const (
	colBlack   = 0
	colRed     = 1
	colGreen   = 2
	colYelow   = 3
	colBlue    = 4
	colMagenta = 5
	colCyan    = 6
	colWhite   = 7
	colDefault = 9
)

// ansi target
const (
	targetPen         = 3
	targetPaper       = 4
	targetBrightPen   = 9
	targetBrightPaper = 10
)

// ansi attribute
const (
	attrBold      = 1
	attrUnderline = 4
	attrInverse   = 7
	attrStrike    = 8
)

// PenColor is the table of colors to be used for text
var PenColor map[string]string

// DimPens is the table of pastel colors to be used for text
var DimPens map[string]string

// PenStyles is the table of styles to be used for text
var PenStyles map[string]string

// NormalPen is the CSI sequence for regular text
var NormalPen string

func init() {
	PenColor = make(map[string]string)
	DimPens = make(map[string]string)
	PenStyles = make(map[string]string)

	NormalPen, _ = ColorBuild("", "", "", false, false)

	PenColor["red"], _ = ColorBuild("red", "normal", "", true, false)
	PenColor["green"], _ = ColorBuild("green", "normal", "", true, false)
	PenColor["yellow"], _ = ColorBuild("yellow", "normal", "", true, false)
	PenColor["blue"], _ = ColorBuild("blue", "normal", "", true, false)
	PenColor["magenta"], _ = ColorBuild("magenta", "normal", "", true, false)
	PenColor["cyan"], _ = ColorBuild("cyan", "normal", "", true, false)
	PenColor["white"], _ = ColorBuild("white", "normal", "", true, false)

	DimPens["red"], _ = ColorBuild("red", "normal", "", false, false)
	DimPens["green"], _ = ColorBuild("green", "normal", "", false, false)
	DimPens["yellow"], _ = ColorBuild("yellow", "normal", "", false, false)
	DimPens["blue"], _ = ColorBuild("blue", "normal", "", false, false)
	DimPens["magenta"], _ = ColorBuild("magenta", "normal", "", false, false)
	DimPens["cyan"], _ = ColorBuild("cyan", "normal", "", false, false)
	DimPens["white"], _ = ColorBuild("white", "normal", "", false, false)

	PenStyles["bold"], _ = ColorBuild("", "", "bold", false, false)
	PenStyles["underline"], _ = ColorBuild("", "", "underline", false, false)
}

// ColorBuild creates the ANSI sequence to create the pen with the correct
// foreground/background color and attribute
func ColorBuild(pen, paper, attribute string, brightPen, brightPaper bool) (string, error) {
	s := strings.Builder{}
	s.Grow(32)
	s.WriteString("\033[")

	// pen
	if pen != "" {
		penType := targetPen
		if brightPen {
			penType = targetBrightPen
		}
		switch strings.ToUpper(pen) {
		case "BLACK":
			s.WriteString(fmt.Sprintf("%d%d", penType, colBlack))
		case "RED":
			s.WriteString(fmt.Sprintf("%d%d", penType, colRed))
		case "GREEN":
			s.WriteString(fmt.Sprintf("%d%d", penType, colGreen))
		case "YELLOW":
			s.WriteString(fmt.Sprintf("%d%d", penType, colYelow))
		case "BLUE":
			s.WriteString(fmt.Sprintf("%d%d", penType, colBlue))
		case "MAGENTA":
			s.WriteString(fmt.Sprintf("%d%d", penType, colMagenta))
		case "CYAN":
			s.WriteString(fmt.Sprintf("%d%d", penType, colCyan))
		case "WHITE":
			s.WriteString(fmt.Sprintf("%d%d", penType, colWhite))
		case "NORMAL":
			s.WriteString(fmt.Sprintf("%d%d", penType, colDefault))
		case "":
		default:
			return "", fmt.Errorf("unknown ANSI pen (%s)", pen)
		}
	}

	// paper
	if paper != "" {
		if s.Len() > 2 {
			s.WriteString(";")
		}
		// paper
		paperType := targetPaper
		if brightPaper {
			paperType = targetBrightPaper
		}
		switch strings.ToUpper(paper) {
		case "BLACK":
			s.WriteString(fmt.Sprintf("%d%d", paperType, colBlack))
		case "RED":
			s.WriteString(fmt.Sprintf("%d%d", paperType, colRed))
		case "GREEN":
			s.WriteString(fmt.Sprintf("%d%d", paperType, colGreen))
		case "YELLOW":
			s.WriteString(fmt.Sprintf("%d%d", paperType, colYelow))
		case "BLUE":
			s.WriteString(fmt.Sprintf("%d%d", paperType, colBlue))
		case "MAGENTA":
			s.WriteString(fmt.Sprintf("%d%d", paperType, colMagenta))
		case "CYAN":
			s.WriteString(fmt.Sprintf("%d%d", paperType, colCyan))
		case "WHITE":
			s.WriteString(fmt.Sprintf("%d%d", paperType, colWhite))
		case "NORMAL":
			s.WriteString(fmt.Sprintf("%d%d", paperType, colDefault))
		case "":
		default:
			return "", fmt.Errorf("unknown ANSI paper (%s)", paper)
		}
	}

	// attribute
	if attribute != "" {
		if s.Len() > 2 {
			s.WriteString(";")
		}
		switch strings.ToUpper(attribute) {
		case "BOLD": // bold
			s.WriteString(fmt.Sprintf("%d", attrBold))
		case "UNDERLINE": // underline
			s.WriteString(fmt.Sprintf("%d", attrUnderline))
		case "ITALIC": // italic
			s.WriteString(fmt.Sprintf("%d", attrInverse))
		case "STRIKE": // strikethrough
			s.WriteString(fmt.Sprintf("%d", attrStrike))
		case "NORMAL": // normal
		case "":
		default:
			return "", fmt.Errorf("unknown ANSI attribute (%s)", attribute)
		}
	}

	// terminate ANSI sequence
	s.WriteString("m")

	return s.String(), nil
}

// ClearLine is the CSI sequence to clear the entire of the current line
const ClearLine = "\033[2K"

// CursorStore if the CSI sequence to store the current cursor position
const CursorStore = "\033[s"

// CursorRestore if the CSI sequence to restore the cursor position to a
// previous store
const CursorRestore = "\033[u"

// CursorForwardOne is the CSI sequence to move the cursor forward (to the right
// for latin fonts) one character
const CursorForwardOne = "\033[1C"

// CursorBackwardOne is the CSI sequence to move the cursor backward (to the left
// for latin fonts) one character
const CursorBackwardOne = "\033[1D"

// CursorMove is the CSI sequence to move the cursor n characters forward
// (positive numbers) or n characters backwards (negative numbers)
func CursorMove(n int) string {
	if n < 0 {
		return fmt.Sprintf("\033[%dD", -n)
	} else if n > 0 {
		return fmt.Sprintf("\033[%dC", n)
	}
	return ""
}

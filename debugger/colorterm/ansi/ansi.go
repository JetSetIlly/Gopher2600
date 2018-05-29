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

	NormalPen, _ = colorBuild("", "", "", false, false)

	PenColor["red"], _ = colorBuild("red", "normal", "", true, false)
	PenColor["green"], _ = colorBuild("green", "normal", "", true, false)
	PenColor["yellow"], _ = colorBuild("yellow", "normal", "", true, false)
	PenColor["blue"], _ = colorBuild("blue", "normal", "", true, false)
	PenColor["magenta"], _ = colorBuild("magenta", "normal", "", true, false)
	PenColor["cyan"], _ = colorBuild("cyan", "normal", "", true, false)
	PenColor["white"], _ = colorBuild("white", "normal", "", true, false)

	DimPens["red"], _ = colorBuild("red", "normal", "", false, false)
	DimPens["green"], _ = colorBuild("green", "normal", "", false, false)
	DimPens["yellow"], _ = colorBuild("yellow", "normal", "", false, false)
	DimPens["blue"], _ = colorBuild("blue", "normal", "", false, false)
	DimPens["magenta"], _ = colorBuild("magenta", "normal", "", false, false)
	DimPens["cyan"], _ = colorBuild("cyan", "normal", "", false, false)
	DimPens["white"], _ = colorBuild("white", "normal", "", false, false)

	PenStyles["bold"], _ = colorBuild("", "", "bold", false, false)
	PenStyles["underline"], _ = colorBuild("", "", "underline", false, false)
}

func colorBuild(pen, paper, attribute string, brightPen, brightPaper bool) (string, error) {
	s := "\033["

	// pen
	if pen != "" {
		penType := targetPen
		if brightPen {
			penType = targetBrightPen
		}
		switch strings.ToUpper(pen) {
		case "BLACK":
			s = fmt.Sprintf("%s%d%d", s, penType, colBlack)
		case "RED":
			s = fmt.Sprintf("%s%d%d", s, penType, colRed)
		case "GREEN":
			s = fmt.Sprintf("%s%d%d", s, penType, colGreen)
		case "YELLOW":
			s = fmt.Sprintf("%s%d%d", s, penType, colYelow)
		case "BLUE":
			s = fmt.Sprintf("%s%d%d", s, penType, colBlue)
		case "MAGENTA":
			s = fmt.Sprintf("%s%d%d", s, penType, colMagenta)
		case "CYAN":
			s = fmt.Sprintf("%s%d%d", s, penType, colCyan)
		case "WHITE":
			s = fmt.Sprintf("%s%d%d", s, penType, colWhite)
		case "NORMAL":
			s = fmt.Sprintf("%s%d%d", s, penType, colDefault)
		case "":
		default:
			return "", fmt.Errorf("unknown ANSI pen (%s)", pen)
		}
	}

	// paper
	if paper != "" {
		if len(s) > 2 {
			s = fmt.Sprintf("%s;", s)
		}
		// paper
		paperType := targetPaper
		if brightPaper {
			paperType = targetBrightPaper
		}
		switch strings.ToUpper(paper) {
		case "BLACK":
			s = fmt.Sprintf("%s%d%d", s, paperType, colBlack)
		case "RED":
			s = fmt.Sprintf("%s%d%d", s, paperType, colRed)
		case "GREEN":
			s = fmt.Sprintf("%s%d%d", s, paperType, colGreen)
		case "YELLOW":
			s = fmt.Sprintf("%s%d%d", s, paperType, colYelow)
		case "BLUE":
			s = fmt.Sprintf("%s%d%d", s, paperType, colBlue)
		case "MAGENTA":
			s = fmt.Sprintf("%s%d%d", s, paperType, colMagenta)
		case "CYAN":
			s = fmt.Sprintf("%s%d%d", s, paperType, colCyan)
		case "WHITE":
			s = fmt.Sprintf("%s%d%d", s, paperType, colWhite)
		case "NORMAL":
			s = fmt.Sprintf("%s%d%d", s, paperType, colDefault)
		case "":
		default:
			return "", fmt.Errorf("unknown ANSI paper (%s)", paper)
		}
	}

	// attribute
	if attribute != "" {
		if len(s) > 2 {
			s = fmt.Sprintf("%s;", s)
		}
		switch strings.ToUpper(attribute) {
		case "BOLD": // bold
			s = fmt.Sprintf("%s%d", s, attrBold)
		case "UNDERLINE": // underline
			s = fmt.Sprintf("%s%d", s, attrUnderline)
		case "ITALIC": // italic
			s = fmt.Sprintf("%s%d", s, attrInverse)
		case "STRIKE": // strikethrough
			s = fmt.Sprintf("%s%d", s, attrStrike)
		case "NORMAL": // normal
		case "":
		default:
			return "", fmt.Errorf("unknown ANSI attribute (%s)", attribute)
		}
	}

	// terminate ANSI sequence
	s = fmt.Sprintf("%sm", s)

	return s, nil
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

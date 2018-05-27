package colorterm

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

var pens map[string]string
var dimPens map[string]string
var penStyles map[string]string
var ansiOff string

func init() {
	pens = make(map[string]string)
	dimPens = make(map[string]string)
	penStyles = make(map[string]string)

	ansiOff, _ = ansiBuild("", "", "", false, false)

	pens["red"], _ = ansiBuild("red", "normal", "", true, false)
	pens["green"], _ = ansiBuild("green", "normal", "", true, false)
	pens["yellow"], _ = ansiBuild("yellow", "normal", "", true, false)
	pens["blue"], _ = ansiBuild("blue", "normal", "", true, false)
	pens["magenta"], _ = ansiBuild("magenta", "normal", "", true, false)
	pens["cyan"], _ = ansiBuild("cyan", "normal", "", true, false)
	pens["white"], _ = ansiBuild("white", "normal", "", true, false)

	dimPens["red"], _ = ansiBuild("red", "normal", "", false, false)
	dimPens["green"], _ = ansiBuild("green", "normal", "", false, false)
	dimPens["yellow"], _ = ansiBuild("yellow", "normal", "", false, false)
	dimPens["blue"], _ = ansiBuild("blue", "normal", "", false, false)
	dimPens["magenta"], _ = ansiBuild("magenta", "normal", "", false, false)
	dimPens["cyan"], _ = ansiBuild("cyan", "normal", "", false, false)
	dimPens["white"], _ = ansiBuild("white", "normal", "", false, false)

	penStyles["bold"], _ = ansiBuild("", "", "bold", false, false)
	penStyles["underline"], _ = ansiBuild("", "", "underline", false, false)
}

func printAnsiTable() {
	for k, v := range pens {
		fmt.Printf("%s%s = <esc>%s\n", v, k, v[1:])
		fmt.Print(ansiOff)
	}
	for k, v := range dimPens {
		fmt.Printf("%s%s = <esc>%s\n", v, k, v[1:])
		fmt.Print(ansiOff)
	}
	for k, v := range penStyles {
		fmt.Printf("%s%s = <esc>%s\n", v, k, v[1:])
		fmt.Print(ansiOff)
	}
}

func ansiBuild(pen, paper, attribute string, brightPen, brightPaper bool) (string, error) {
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

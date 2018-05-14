package colorterm

import (
	"fmt"
	"strings"
)

// ansi color
const (
	black   = 0
	red     = 1
	green   = 2
	yellow  = 3
	blue    = 4
	magenta = 5
	cyan    = 6
	white   = 7
)

// ansi target
const (
	pen         = 3
	paper       = 4
	brightPen   = 9
	brightPaper = 10
)

// ansi attribute
const (
	bold      = 1
	underline = 4
	inverse   = 7
	strike    = 8
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
		penType := 3
		if brightPen {
			penType = 9
		}
		switch strings.ToUpper(pen)[0] {
		case 'R':
			s = fmt.Sprintf("%s%d%d", s, penType, red)
		case 'G':
			s = fmt.Sprintf("%s%d%d", s, penType, green)
		case 'Y':
			s = fmt.Sprintf("%s%d%d", s, penType, yellow)
		case 'B':
			s = fmt.Sprintf("%s%d%d", s, penType, blue)
		case 'M':
			s = fmt.Sprintf("%s%d%d", s, penType, magenta)
		case 'C':
			s = fmt.Sprintf("%s%d%d", s, penType, cyan)
		case 'W':
			s = fmt.Sprintf("%s%d%d", s, penType, white)
		case 'D', 'N':
			s = fmt.Sprintf("%s%d9", s, penType)
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
		paperType := 4
		if brightPaper {
			paperType = 10
		}
		switch strings.ToUpper(paper)[0] {
		case 'R':
			s = fmt.Sprintf("%s%d%d", s, paperType, red)
		case 'G':
			s = fmt.Sprintf("%s%d%d", s, paperType, green)
		case 'Y':
			s = fmt.Sprintf("%s%d%d", s, paperType, yellow)
		case 'B':
			s = fmt.Sprintf("%s%d%d", s, paperType, blue)
		case 'M':
			s = fmt.Sprintf("%s%d%d", s, paperType, magenta)
		case 'C':
			s = fmt.Sprintf("%s%d%d", s, paperType, cyan)
		case 'W':
			s = fmt.Sprintf("%s%d%d", s, paperType, white)
		case 'D', 'N':
			s = fmt.Sprintf("%s%d9", s, paperType)
		default:
			return "", fmt.Errorf("unknown ANSI paper (%s)", paper)
		}
	}

	// attribute
	if attribute != "" {
		if len(s) > 2 {
			s = fmt.Sprintf("%s;", s)
		}
		switch strings.ToUpper(attribute)[0] {
		case 'B':
			s = fmt.Sprintf("%s%d", s, bold)
		case 'U':
			s = fmt.Sprintf("%s%d", s, underline)
		case 'I':
			s = fmt.Sprintf("%s%d", s, inverse)
		case 'S':
			s = fmt.Sprintf("%s%d", s, strike)
		case 'D', 'N', 'P':
			s = fmt.Sprintf("%s", s)
		default:
			return "", fmt.Errorf("unknown ANSI attribute (%s)", attribute)
		}
	}

	// terminate ANSI sequence
	s = fmt.Sprintf("%sm", s)

	return s, nil
}

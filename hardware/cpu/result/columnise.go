package result

import "fmt"

func columnise(s string, width int) string {
	if width > len(s) {
		t := make([]byte, width-len(s))
		for i := 0; i < len(t); i++ {
			t[i] = ' '
		}
		s = fmt.Sprintf("%s%s", s, t)
	} else if width < len(s) {
		s = s[:width]
	}
	return s
}

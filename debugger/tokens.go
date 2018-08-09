package debugger

import (
	"fmt"
	"strings"
)

type tokens struct {
	tokens []string
	curr   int
}

func (tk *tokens) reset() {
	tk.curr = 0
}

func (tk tokens) remainder() string {
	return strings.Join(tk.tokens[tk.curr:], " ")
}

func (tk tokens) remaining() int {
	return len(tk.tokens) - tk.curr
}

func (tk tokens) num() int {
	return len(tk.tokens)
}

func (tk *tokens) get() (string, bool) {
	if tk.curr >= len(tk.tokens) {
		return "", false
	}
	tk.curr++
	return tk.tokens[tk.curr-1], true
}

func (tk *tokens) unget() {
	if tk.curr > 0 {
		tk.curr--
	}
}

func (tk tokens) peek() (string, bool) {
	if tk.curr >= len(tk.tokens) {
		return "", false
	}
	return tk.tokens[tk.curr], true
}

func tokeniseInput(input string) *tokens {
	tk := new(tokens)

	// remove leading/trailing space
	input = strings.TrimSpace(input)

	// divide user input into tokens
	tk.tokens = strings.Fields(input)

	// normalise variations in syntax
	for i := 0; i < len(tk.tokens); i++ {
		// normalise hex notation
		if tk.tokens[i][0] == '$' {
			tk.tokens[i] = fmt.Sprintf("0x%s", tk.tokens[i][1:])
		}
	}

	return tk
}

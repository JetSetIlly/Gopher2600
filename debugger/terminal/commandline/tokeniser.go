// This file is part of Gopher2600.
//
// Gopher2600 is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// Gopher2600 is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with Gopher2600.  If not, see <https://www.gnu.org/licenses/>.

package commandline

import (
	"strings"
)

// Tokens represents tokenised input. This can be used to walk through the
// input string (using get()) for eas(ier) parsing.
type Tokens struct {
	input  string
	tokens []string
	curr   int
}

// String representation of tokens.
func (tk *Tokens) String() string {
	return strings.Join(tk.tokens, " ")
}

// Reset begins the token traversal process from the beginning.
func (tk *Tokens) Reset() {
	tk.curr = 0
}

// End the token traversal process. It can be restarted with the Reset()
// function.
func (tk *Tokens) End() {
	tk.curr = len(tk.tokens)
}

// IsEnd returns true if we're at the end of the token list.
func (tk Tokens) IsEnd() bool {
	return tk.curr >= len(tk.tokens)
}

// Len returns the number of tokens.
func (tk Tokens) Len() int {
	return len(tk.tokens)
}

// Remainder returns the remaining tokens as a string.
func (tk Tokens) Remainder() string {
	return strings.Join(tk.tokens[tk.curr:], " ")
}

// Remaining returns the count of reminaing tokens in the token list.
func (tk Tokens) Remaining() int {
	return len(tk.tokens) - tk.curr
}

// ReplaceEnd changes the last entry of the token list.
func (tk *Tokens) ReplaceEnd(newEnd string) {
	// change end of original string
	t := strings.LastIndex(tk.input, tk.tokens[len(tk.tokens)-1])
	tk.input = tk.input[:t] + newEnd

	// change tokens
	tk.tokens[len(tk.tokens)-1] = newEnd
}

// Get returns the next token in the list, and a success boolean - if the end
// of the token list has been reached, the function returns false instead of
// true.
func (tk *Tokens) Get() (string, bool) {
	if tk.curr >= len(tk.tokens) {
		return "", false
	}
	tk.curr++
	return tk.tokens[tk.curr-1], true
}

// Unget walks backwards in the token list.
func (tk *Tokens) Unget() {
	if tk.curr > 0 {
		tk.curr--
	}
}

// Update last token with a new value. Useful for normalising token entries.
func (tk *Tokens) Update(s string) {
	i := tk.curr
	if i > 0 {
		i--
	}
	tk.tokens[i] = s
}

// Peek returns the next token in the list (without advancing the list), and a
// success boolean - if the end of the token list has been reached, the
// function returns false instead of true.
func (tk Tokens) Peek() (string, bool) {
	if tk.curr >= len(tk.tokens) {
		return "", false
	}
	return tk.tokens[tk.curr], true
}

// TokeniseInput creates and returns a new Tokens instance.
func TokeniseInput(input string) *Tokens {
	tk := &Tokens{}

	// remove leading/trailing space
	input = strings.TrimSpace(input)

	// divide user input into tokens. removes excess white space
	tk.tokens = tokeniseInput(input)

	// take a note of the raw input
	tk.input = input

	return tk
}

// tokeniseInput is the "raw" tokenising function (without normalisation or
// wrapping everything up in a Tokens instance). used by the fancier
// TokeniseInput and anywhere else where we need to divide input into tokens
// (eg. TabCompletion.Complete()).
func tokeniseInput(input string) []string {
	quoted := false
	tokens := make([]string, 0)

	markStart := 0
	markEnd := 0

	i := 0
	for i = 0; i < len(input); i++ {
		switch input[i] {
		case ' ':
			if !quoted {
				if markEnd >= markStart {
					tokens = append(tokens, input[markStart:markEnd+1])
				}
				markStart = i + 1
			} else {
				markEnd = i
			}
		case '"':
			if quoted {
				if markEnd > markStart {
					tokens = append(tokens, input[markStart:markEnd+1])
				}
				markEnd = i
			}
			markStart = i + 1
			quoted = !quoted
		default:
			markEnd = i
		}
	}
	markEnd = i

	if markEnd > markStart {
		tokens = append(tokens, input[markStart:markEnd])
	}

	return tokens
}

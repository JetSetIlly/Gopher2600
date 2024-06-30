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

package terminal

import (
	"fmt"
	"strings"

	"github.com/jetsetilly/gopher2600/coprocessor"
)

// Prompt specifies the prompt text and the prompt style
type Prompt struct {
	Type PromptType

	// the content
	Content string

	// the current CoProcYield information. used to add additional information
	// to Content string
	CoProcYield coprocessor.CoProcYield

	// whether the terminal is recording input
	Recording bool
}

// PromptType identifies the type of information in the prompt.
type PromptType int

// List of prompt types.
const (
	PromptTypeCPUStep PromptType = iota
	PromptTypeVideoStep
	PromptTypeCartYield
	PromptTypeConfirm
)

// String returns the prompt with "standard" decordation. Good for terminals
// with no graphical capabilities at all. A GUI based terminal interface may
// choose not to use this.
func (p Prompt) String() string {
	if p.Type == PromptTypeConfirm {
		return p.Content
	}

	s := strings.Builder{}
	s.WriteString("[ ")
	if p.Recording {
		s.WriteString("(rec) ")
	}

	s.WriteString(strings.TrimSpace(p.Content))

	if !p.CoProcYield.Type.Normal() {
		s.WriteString(fmt.Sprintf(" (%s)", p.CoProcYield.Type))
	}

	s.WriteString(" ]")

	switch p.Type {
	case PromptTypeCPUStep:
		s.WriteString(" >> ")
	case PromptTypeVideoStep:
		s.WriteString(" > ")
	case PromptTypeCartYield:
		s.WriteString(" . ")
	}

	return s.String()
}

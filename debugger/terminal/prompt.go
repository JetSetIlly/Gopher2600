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

import "strings"

// Prompt specifies the prompt text and the prompt style. For CPUStep
// and VideoStep prompt types thre is some additional information that
// can be used to decorate the prompt.
type Prompt struct {
	Content string
	Type    PromptType

	// valid for PromptTypeCPUStep and PromptTypeVideoStep
	CPURdy    bool
	Recording bool
}

// PromptType identifies the type of information in the prompt.
type PromptType int

// List of prompt types.
const (
	PromptTypeCPUStep PromptType = iota
	PromptTypeVideoStep
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
	s.WriteString("[")
	if p.Recording {
		s.WriteString("(rec)")
	}
	s.WriteString(" ")
	s.WriteString(p.Content)

	s.WriteString(" ]")

	if !p.CPURdy {
		s.WriteString(" !")
	}

	if p.Type == PromptTypeCPUStep {
		s.WriteString(" >> ")
	} else {
		s.WriteString(" > ")
	}

	return s.String()
}

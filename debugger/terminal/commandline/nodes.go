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
	"fmt"
	"strings"
)

type nodeType int

func (t nodeType) String() string {
	switch t {
	case nodeRoot:
		return "nodeRoot"
	case nodeRequired:
		return "nodeRequired"
	case nodeOptional:
		return "nodeOptional"
	}
	panic("unknown nodeType")
}

const (
	nodeRoot nodeType = iota + 1
	nodeRequired
	nodeOptional
)

// nodes are chained together thought the next and branch arrays.
type node struct {
	// tag should be non-empty - except in the case of some nested groups
	tag string

	// friendly name for the placeholder tags. not used if tag is not a
	// placeholder. you can use isPlaceholder() to check
	placeholderLabel string

	typ nodeType

	next   []*node
	branch []*node

	repeatStart bool
	repeat      *node
}

// String returns the verbose representation of the node (and its children).
// Use this only for testing/validation purposes. HelpString() is more useful
// to the end user.
func (n node) String() string {
	return n.string(false, false)
}

// HelpString returns the string representation of the node (and it's children)
// without extraneous placeholder directives (if placeholderLabel is available)
//
// So called because it's better to use when displaying help.
func (n node) usageString() string {
	return n.string(true, false)
}

// string() outputs the node, and any children, as best as it can. when called
// upon the first node in a command it has the effect of recreating the
// original input to each template entry parsed by ParseCommandTemplate()
//
// however, because of the way the parsing works, it's not always possible to
// recreate accurately the original template entry, but that's okay. the node
// tree is effectively, an optimised tree and so the output from String() is
// likewise, optimised
//
// optimised in this case means the absence of superfluous group indicators.
// for example:
//
//		TEST [1 [2] [3] [4] [5]]
//
// is the same as:
//
//		TEST [1 2 3 4 5]
//
// note: string should not be called directly except as a recursive call
// or as an initial call from String() and usageString()
//
func (n node) string(useLabels bool, fromBranch bool) string {
	s := strings.Builder{}

	if n.isPlaceholder() && n.placeholderLabel != "" {
		// placeholder labels come without angle brackets
		label := fmt.Sprintf("<%s>", n.placeholderLabel)
		if useLabels {
			s.WriteString(label)
		} else {
			s.WriteString(fmt.Sprintf("%%%s%c", label, n.tag[1]))
		}
	} else {
		s.WriteString(n.tag)
	}

	if n.next != nil {
		for i := range n.next {
			prefix := " "

			// this is a bit of a special condition to catch the case of an
			// optional group followed by a required node. there may be a more
			// general case
			if n.typ == nodeOptional && n.tag == "" && !n.repeatStart && !fromBranch {
				if i == 0 && n.next[i].typ == nodeRequired {
					s.WriteString(prefix)
					s.WriteString("(")
					prefix = ""
				}
			}

			if n.next[i].repeatStart {
				s.WriteString(prefix)
				s.WriteString("{")
				prefix = ""
			}

			if n.next[i].typ == nodeRequired && (n.typ != nodeRequired || n.next[i].branch != nil) {
				s.WriteString(prefix)
				s.WriteString("[")
			} else if n.next[i].typ == nodeOptional && (n.typ != nodeOptional || n.next[i].branch != nil) {
				// repeat groups are optional groups by definition so we don't
				// need to include the optional group delimiter
				if !n.next[i].repeatStart {
					s.WriteString(prefix)
					s.WriteString("(")
				}
			} else {
				s.WriteString(prefix)
			}

			s.WriteString(n.next[i].string(useLabels, false))

			if n.next[i].typ == nodeRequired && (n.typ != nodeRequired || n.next[i].branch != nil) {
				s.WriteString("]")
			} else if n.next[i].typ == nodeOptional && (n.typ != nodeOptional || n.next[i].branch != nil) {
				// see comment above
				if !n.next[i].repeatStart {
					s.WriteString(")")
				}
			}
		}
	}

	if n.branch != nil {
		for i := range n.branch {
			s.WriteString(fmt.Sprintf("|%s", n.branch[i].string(useLabels, true)))
		}
	}

	// unlike the other close group delimiters, we add the close repeat group
	// here. this is the best way of making sure we add exactly one close
	// delimiter for every open delimiter.
	if n.repeatStart {
		s.WriteString("}")
	}

	// close an optional group that was opened because of a special condition
	// (described above)
	if n.typ == nodeOptional && n.tag == "" && !n.repeatStart && !fromBranch {
		s.WriteString(")")
	}

	return strings.TrimSpace(s.String())
}

// nodeVerbose returns a readable representation of the node, listing branches
// if necessary.
func (n node) nodeVerbose() string {
	s := strings.Builder{}
	s.WriteString(n.tagVerbose())
	for bi := range n.branch {
		if n.branch[bi].tag != "" {
			s.WriteString(" or ")
			s.WriteString(n.branch[bi].tagVerbose())
		}
	}
	return s.String()
}

// tagVerbose returns a readable versions of the tag field, using labels if
// possible.
func (n node) tagVerbose() string {
	if n.isPlaceholder() {
		if n.placeholderLabel != "" {
			return n.placeholderLabel
		}

		switch n.tag {
		case "%S":
			return "string argument"
		case "%N":
			return "numeric argument"
		case "%P":
			return "floating-point argument"
		case "%F":
			return "filename argument"
		default:
			return "placeholder argument"
		}
	}
	return n.tag
}

// isPlaceholder checks tag to see if it is a placeholder. does not check to
// see if placeholder is valid.
func (n node) isPlaceholder() bool {
	return len(n.tag) == 2 && n.tag[0] == '%'
}

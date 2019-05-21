package commandline

import (
	"fmt"
	"strings"
)

// Commands is the root of the command tree. the top-level of the Commands tree
// is an array of nodes. each of these nodes is the start of a command.
type Commands []*node

// Len implements Sort package interface
func (cmds Commands) Len() int {
	return len(cmds)
}

// Less implements Sort package interface
func (cmds Commands) Less(i int, j int) bool {
	return cmds[i].tag < cmds[j].tag
}

// Swap implements Sort package interface
func (cmds Commands) Swap(i int, j int) {
	swp := cmds[i]
	cmds[i] = cmds[j]
	cmds[j] = swp
}

func (cmds Commands) String() string {
	s := strings.Builder{}
	for c := range cmds {
		s.WriteString(fmt.Sprintf("%v", cmds[c]))
		s.WriteString("\n")
	}
	return strings.TrimRight(s.String(), "\n")
}

type nodeType int

const (
	nodeUndefined nodeType = iota
	nodeRoot
	nodeRequired
	nodeOptional
)

// nodes are chained together throught the next and branch arrays.
type node struct {
	// tag should be non-empty - except in the case of some nested groups
	tag string

	typ nodeType

	next   []*node
	branch []*node

	repeatStart bool
	repeat      *node
}

// String returns the string representation of the node (and it's children)
func (n node) String() string {
	s := strings.Builder{}

	if n.repeatStart {
		s.WriteString("{")
	} else if n.typ == nodeOptional {
		s.WriteString("(")
		defer func() {
			s.WriteString(")")
		}()
	}
	if n.typ == nodeRequired {
		s.WriteString("[")
		defer func() {
			s.WriteString("]")
		}()
	}

	s.WriteString(n.stringBuilder())
	return s.String()
}

// stringBuilder() outputs the node, and any children, as best as it can. when called
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
// note: stringBuilder should not be called directly except as a recursive call
// or as an initial call from String()
//
func (n node) stringBuilder() string {
	s := strings.Builder{}

	s.WriteString(n.tag)

	if n.next != nil {
		for i := range n.next {
			prefix := " "
			if n.next[i].repeatStart {
				s.WriteString(" {")
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

			s.WriteString(fmt.Sprintf("%s", n.next[i].stringBuilder()))

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
			s.WriteString(fmt.Sprintf("|%s", n.branch[i].stringBuilder()))
		}
	}

	// unlike the other close group delimiters, we add the close repeat group
	// here. this is the best way of making sure we add exactly one close
	// delimiter for every open delimiter.
	if n.repeatStart {
		s.WriteString("}")
	}

	return strings.TrimSpace(s.String())
}

// branchesText creates a readable string, listing all the branchesText of the node
func (n node) branchesText() string {
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

// tagVerbose returns a decriptive string for placeholder values
func (n node) tagVerbose() string {
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
		return n.tag
	}
}

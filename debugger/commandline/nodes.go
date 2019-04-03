package commandline

import (
	"fmt"
	"strings"
)

// Commands is the root of the command tree
//
// currently, the top-level of the Commands tree is an array of nodes. each
// entry in this array is effectively a branch off a conceptual root-node. with
// a bit of work, we could alter the command tree such that the array is a
// sequence of branches off an otherwise unused root-node. this would simplify
// validation and tab-completion a little bit. as it is though, this is fine
// for now.
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

type groupType int

const (
	groupUndefined groupType = iota
	groupRoot
	groupRequired
	groupOptional
)

type node struct {
	// tag should always be non-empty
	tag string

	// group will have the following values:
	//  groupRoot: nodes that are not in an explicit grouping
	//  groupRequired
	//  groupOptional
	group groupType

	next   []*node
	branch []*node
}

func (n node) String() string {
	s := strings.Builder{}

	s.WriteString(n.tag)

	if n.next != nil {
		for i := range n.next {
			if n.next[i].group == groupRequired {
				s.WriteString(" [")
			} else if n.next[i].group == groupOptional {
				s.WriteString(" (")
			} else {
				s.WriteString(" ")
			}
			s.WriteString(fmt.Sprintf("%s", n.next[i]))
			if n.next[i].group == groupRequired {
				s.WriteString("]")
			} else if n.next[i].group == groupOptional {
				s.WriteString(")")

			}
		}
	}

	if n.branch != nil {
		for i := range n.branch {
			s.WriteString(fmt.Sprintf("|%s", n.branch[i]))
		}
	}

	return s.String()
}

package commandline

import (
	"fmt"
	"gopher2600/errors"
	"strings"
)

// ParseCommandTemplate turns a string representation of a command template
// into a machine friendly representation
//
// Syntax
//   [ a ]	required keyword
//   ( a )	optional keyword
//   [ a | b | ... ]	required selection
//   ( a | b | ... )	optional selection
//
// groups can be embedded in one another
//
// Placeholders
//   %N		numeric value
//   %P		irrational number value
//   %S     string
//   %F     file name
//   %*     allow anything to follow this point
//
// note that a placeholder will implicitly be treated as a separate token
//
func ParseCommandTemplate(template []string) (*Commands, error) {
	cmds := make(Commands, 0, 10)
	for t := range template {
		defn := template[t]

		// tidy up spaces in definition string - we don't want more than one
		// consecutive space
		defn = strings.Join(strings.Fields(defn), " ")

		// normalise to upper case
		defn = strings.ToUpper(defn)

		// parse the definition for this command
		p, d, err := parseDefinition(defn, "")
		if err != nil {
			return nil, errors.NewFormattedError(errors.ParserError, defn, err, d)
		}

		// add to list of commands (order doesn't matter at this stage)
		cmds = append(cmds, p)

		// check that parsing was complete
		if d < len(defn)-1 {
			return nil, errors.NewFormattedError(errors.ParserError, defn, "outstanding characters in definition")
		}
	}

	return &cmds, nil
}

func parseDefinition(defn string, trigger string) (*node, int, error) {
	// handle special conditions before parsing loop
	if defn[0] == '(' || defn[0] == '[' {
		return nil, 0, fmt.Errorf("first argument of a group should not be itself be the start of a group")
	}

	// working nodes should be initialised with this function
	newWorkingNode := func() *node {
		if trigger == "(" {
			return &node{group: groupOptional}
		} else if trigger == "[" {
			return &node{group: groupRequired}
		} else if trigger == "|" {
			// group is left unset for the branch trigger. value will be set
			// once parseDefinition() has returned
			return &node{}
		} else if trigger == "" {
			return &node{group: groupRoot}
		}

		panic("unknown trigger")
	}

	wn := newWorkingNode() // working node (attached to the end of the sequence when required)
	sn := wn               // start node (of the sequence)

	addNext := func(nx *node) error {
		// new node is already in the correct place
		if sn == nx {
			wn = newWorkingNode()
			return nil
		}

		// do not add nodes that have no content
		if nx.tag == "" {
			return nil
		}

		// sanity check to make sure we're not clobbering an active working
		// node
		if wn != nx && wn.tag != "" {
			return fmt.Errorf("orphaned working node: %s", wn.tag)
		}

		// create a new next array if necessary, and add new node to the end of
		// it
		if sn.next == nil {
			sn.next = make([]*node, 0)
		}
		sn.next = append(sn.next, nx)

		// create new working node
		wn = newWorkingNode()

		return nil
	}

	addBranch := func(bx *node) error {
		// do not add nodes that have no content
		if bx.tag == "" {
			return nil
		}

		// sanity check to make sure we're not clobbering an active working
		// node
		if wn != bx && wn.tag != "" {
			return fmt.Errorf("orphaned working node: %s", wn.tag)
		}

		// create a new next array if necessary, and add new node to the end of
		// it
		if sn.branch == nil {
			sn.branch = make([]*node, 0)
		}
		sn.branch = append(sn.branch, bx)

		// create new working node
		wn = newWorkingNode()

		return nil
	}

	for i := 0; i < len(defn); i++ {
		switch defn[i] {
		case '[':
			err := addNext(wn)
			if err != nil {
				return nil, i, err
			}

			i++
			ns, e, err := parseDefinition(defn[i:], "[")
			if err != nil {
				return nil, i + e, err
			}
			ns.group = groupRequired

			err = addNext(ns)
			if err != nil {
				return nil, i, err
			}

			i += e

		case '(':
			err := addNext(wn)
			if err != nil {
				return nil, i, err
			}

			i++
			ns, e, err := parseDefinition(defn[i:], "(")
			if err != nil {
				return nil, i + e, err
			}
			ns.group = groupOptional

			err = addNext(ns)

			if err != nil {
				return nil, i, err
			}

			i += e

		case ']':
			err := addNext(wn)
			if err != nil {
				return nil, i, err
			}

			if trigger == "[" {
				return sn, i, nil
			}
			if trigger == "|" {
				return sn, i - 1, nil
			}
			return nil, i, fmt.Errorf("unexpected ]")

		case ')':
			err := addNext(wn)
			if err != nil {
				return nil, i, err
			}

			if trigger == "(" {
				return sn, i, nil
			}
			if trigger == "|" {
				return sn, i - 1, nil
			}
			return nil, i, fmt.Errorf("unexpected )")

		case '|':
			err := addNext(wn)
			if err != nil {
				return nil, i, err
			}

			if trigger == "|" {
				return sn, i - 1, nil
			}

			i++

			nb, e, err := parseDefinition(defn[i:], "|")
			if err != nil {
				return nil, i + e, err
			}

			// change group to current group - we don't want any unresolved
			// instances of groupUndefined
			nb.group = sn.group

			err = addBranch(nb)
			if err != nil {
				return nil, i, err
			}

			i += e

		case '%':
			if wn.tag != "" {
				return nil, i, fmt.Errorf("placeholders cannot be part of a wider string")
			}

			if i == len(defn)-1 {
				return nil, i, fmt.Errorf("orphaned placeholder directives not allowed")
			}

			// add placeholder to working node if it is recognised
			p := string(defn[i+1])

			if p != "N" && p != "P" && p != "S" && p != "F" && p != "*" && p != "%" {
				return nil, i, fmt.Errorf("unknown placeholder directive (%s)", wn.tag)
			}

			wn.tag = fmt.Sprintf("%%%s", p)

			// we've consumed an additional character when retreiving a value
			// for p
			i++

		case ' ':
			// tokens are separated by spaces as well group markers
			err := addNext(wn)
			if err != nil {
				return nil, i, err
			}

		default:
			wn.tag += string(defn[i])
		}

	}

	// make sure we've added working node to the sequence
	err := addNext(wn)
	if err != nil {
		return nil, len(defn), err
	}

	// if we reach this point and trigger is non-empty then that implies that
	// the opening trigger has not been closed correctly
	if trigger == "[" || trigger == "(" {
		return nil, len(defn), fmt.Errorf(fmt.Sprintf("unclosed %s group", trigger))
	}

	return sn, len(defn), nil
}

package commandline

import (
	"fmt"
	"gopher2600/errors"
	"io"
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
//   %S     string (numbers can be strings too)
//   %F     file name
//
// !!TODO: the following pattern parses correctly but doesn't yet mean anything
// and will never validate.
//
//    {[arg]}
//
// an idea would be to expand this to:
//
//    [arg] {arg}
//
// meaning one or more repetition of the arg pattern.
//
func ParseCommandTemplate(template []string) (*Commands, error) {
	return ParseCommandTemplateWithOutput(template, nil)
}

// ParseCommandTemplateWithOutput is the same as ParseCommandTemplate but with
// the outputting of optimised definitions
//
// an output argument of nil is valid
func ParseCommandTemplateWithOutput(template []string, output io.Writer) (*Commands, error) {
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
			return nil, errors.New(errors.ParserError, defn, err, d)
		}

		// check that parsing was complete
		if d < len(defn)-1 {
			return nil, errors.New(errors.ParserError, defn, "outstanding characters in definition")
		}

		// add to list of commands (order doesn't matter at this stage)
		cmds = append(cmds, p)

		// output optimisations
		if output != nil && p.String() != defn {
			output.Write([]byte(defn))
			output.Write([]byte(" -> "))
			output.Write([]byte(p.String()))
			output.Write([]byte("\n"))
		}
	}

	return &cmds, nil
}

func parseDefinition(defn string, trigger string) (*node, int, error) {
	// working nodes should be initialised with this function
	newWorkingNode := func() (*node, error) {
		switch trigger {
		case "(":
			return &node{typ: nodeOptional}, nil
		case "[":
			return &node{typ: nodeRequired}, nil
		case "{":
			return &node{typ: nodeOptional}, nil
		case "|":
			// group is left unset for the branch trigger. value will be set
			// once parseDefinition() has returned
			return &node{}, nil
		case "":
			return &node{typ: nodeRoot}, nil
		default:
			return nil, errors.New(errors.ParserError, defn, "unknown group type")
		}
	}

	wn, err := newWorkingNode() // working node (attached to the end of the sequence when required)
	if err != nil {
		return nil, 0, err
	}
	sn := wn // start node (of the sequence)

	addNext := func(nn *node) error {
		// new node is already in the correct place
		if sn == nn {
			wn, err = newWorkingNode()
			if err != nil {
				return err
			}
			return nil
		}

		// do not add nodes that have no content
		if nn.tag == "" && nn.next == nil {
			return nil
		}

		// create a new next array if necessary, and add new node to the end of
		// it
		if sn.next == nil {
			sn.next = make([]*node, 0)
		}
		sn.next = append(sn.next, nn)

		// create new working node
		wn, err = newWorkingNode()
		if err != nil {
			return err
		}

		return nil
	}

	addBranch := func(bn *node) error {
		// do not add nodes that have no content
		if bn.tag == "" && bn.next == nil {
			return nil
		}

		// create a new next array if necessary, and add new node to the end of
		// it
		if sn.branch == nil {
			sn.branch = make([]*node, 0)
		}
		sn.branch = append(sn.branch, bn)

		// create new working node
		wn, err = newWorkingNode()
		if err != nil {
			return err
		}

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
			ns.typ = nodeRequired

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
			ns.typ = nodeOptional

			err = addNext(ns)

			if err != nil {
				return nil, i, err
			}

			i += e

		case '{':
			err := addNext(wn)
			if err != nil {
				return nil, i, err
			}

			i++
			ns, e, err := parseDefinition(defn[i:], "{")
			if err != nil {
				return nil, i + e, err
			}
			ns.typ = nodeOptional

			// add repeat information to new nodes
			ns.repeatStart = true
			if ns.next != nil {
				ns.next[len(ns.next)-1].repeat = ns
			} else {
				ns.repeat = ns
			}

			// include branches in the repeating
			for bi := range ns.branch {
				n := ns.branch[bi]
				if n.next != nil {
					n.next[len(n.next)-1].repeat = ns
				} else {
					n.repeat = ns
				}
			}

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

		case '}':
			err := addNext(wn)
			if err != nil {
				return nil, i, err
			}

			if trigger == "{" {
				return sn, i, nil
			}
			if trigger == "|" {
				return sn, i - 1, nil
			}
			return nil, i, fmt.Errorf("unexpected }")

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

			// change group to current group
			nb.typ = sn.typ

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

			if p != "N" && p != "P" && p != "S" && p != "F" && p != "%" {
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
	err = addNext(wn)
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

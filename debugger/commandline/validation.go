package commandline

import (
	"fmt"
	"strconv"
	"strings"
)

// Validate input string against command defintions
func (cmds Commands) Validate(input string) error {
	return cmds.ValidateTokens(TokeniseInput(input))
}

// ValidateTokens like Validate, but works on tokens rather than an input
// string
func (cmds Commands) ValidateTokens(tokens *Tokens) error {
	cmd, ok := tokens.Peek()
	if !ok {
		return nil
	}
	cmd = strings.ToUpper(cmd)

	for n := range cmds {
		if cmd == cmds[n].tag {

			err := cmds[n].validate(tokens)
			if err != nil {
				return fmt.Errorf("%s for %s", err, cmd)
			}

			if tokens.Remaining() > 0 {
				return fmt.Errorf("too many arguments for %s", cmd)
			}

			return nil
		}
	}

	return fmt.Errorf("unrecognised command (%s)", cmd)
}

func placeHolderText(text string) string {
	switch text {
	case "%*":
		return "required arguments"
	case "%S":
		return "string argument"
	case "%V":
		return "numeric argument"
	case "%I":
		return "floating-point argument"
	case "%F":
		return "filename argument"
	default:
		return text
	}
}

// branches creates a readable string, listing all the branches of the node
func branches(n *node) string {
	s := strings.Builder{}
	s.WriteString(placeHolderText(n.tag))
	for bi := range n.branch {
		s.WriteString(" or ")
		s.WriteString(placeHolderText(n.branch[bi].tag))
	}
	return s.String()
}

func (n *node) validate(tokens *Tokens) error {
	// if there is no more input then return true (validation has passed) if
	// the node is optional, false if it is required
	tok, ok := tokens.Get()
	if !ok {
		// we treat arguments in the root-group as though they are required,
		// with the exception of the %* placeholder
		if n.group == groupRequired || (n.group == groupRoot && n.tag != "%*") {
			// replace placeholder arguments with something a little less cryptic
			s := strings.Builder{}
			if len(n.branch) > 0 {
				return fmt.Errorf("missing a required argument (%s)", branches(n))
			}
			s.WriteString("missing ")
			s.WriteString(placeHolderText(n.tag))
			return fmt.Errorf(s.String())
		}

		return nil
	}

	// check to see if input matches this node. using placeholder matching if
	// appropriate

	match := true

	// default error in case nothing matches - replaced as necessary
	err := fmt.Errorf("unrecognised argument (%s)", tok)

	switch n.tag {
	case "%V":
		_, e := strconv.ParseInt(tok, 0, 32)
		if e != nil {
			err = fmt.Errorf("numeric argument required (%s is not numeric)", tok)
			match = false
		}

	case "%I":
		_, e := strconv.ParseFloat(tok, 32)
		if e != nil {
			err = fmt.Errorf("float argument required (%s is not numeric)", tok)
			match = false
		}

	case "%S":
		// string placeholders do not cause a match if the node has branches.
		// if they did then they would be acting in the same way as the %*
		// placeholder and any subsequent branches will not be considered at
		// all.

		match = n.branch == nil

	case "%F":
		// TODO: check for file existance

		// see commentary for %S above
		match = n.branch == nil

	case "%*":
		// this placeholder indicates that the rest of the tokens can be
		// ignored.

		// consume the rest of the tokens without a care
		for ok {
			_, ok = tokens.Get()
		}

		return nil

	default:
		// case sensitive matching
		tok = strings.ToUpper(tok)
		match = tok == n.tag
	}

	// if input doesn't match this node, check branches
	if !match {
		if n.branch != nil {
			for bi := range n.branch {
				// recursing into the validate function means we need to use the
				// same token as above. Unget() prepares the tokens object for
				// that.
				tokens.Unget()

				if n.branch[bi].validate(tokens) == nil {

					//  break loop on first successful branch
					match = true
					break
				}
			}

			// tricky condition: if we've not found anything in any of the
			// branches and this is an optional group, then claim that we have
			// matched this group and prepare tokens object for additional
			// nodes. if group is not optional then return error.
			if !match {
				if n.group == groupOptional {
					tokens.Unget()
				} else {
					return err
				}
			}

			return nil
		}

		if !match {
			return err
		}
	}

	// input does match this node. check nodes that follow on.
	for ni := range n.next {
		err := n.next[ni].validate(tokens)
		if err != nil {
			return err
		}
	}

	return nil
}

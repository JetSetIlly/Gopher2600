package commandline

import (
	"fmt"
	"gopher2600/errors"
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

			err := cmds[n].validate(tokens, false)
			if err != nil {
				return errors.NewFormattedError(errors.ValidationError, err, cmd)
			}

			if tokens.Remaining() > 0 {
				// TODO: this error message is misleading when the last
				// argument in a branch is optional and the supplied argument
				// does not match. in those instances, it should say something
				// like: "argument does not match option"
				//
				// we need a way to detect that situation if we reach this
				// point.
				return errors.NewFormattedError(errors.ValidationError, "too many arguments", cmd)
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

// branchesText creates a readable string, listing all the branchesText of the node
func branchesText(n *node) string {
	s := strings.Builder{}
	s.WriteString(placeHolderText(n.tag))
	for bi := range n.branch {
		s.WriteString(" or ")
		s.WriteString(placeHolderText(n.branch[bi].tag))
	}
	return s.String()
}

func (n *node) validate(tokens *Tokens, speculative bool) error {
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
				return fmt.Errorf("missing a required argument (%s)", branchesText(n))
			}
			s.WriteString("missing ")
			s.WriteString(placeHolderText(n.tag))
			return fmt.Errorf(s.String())
		}

		return nil
	}

	// check to see if input matches this node. using placeholder matching if
	// appropriate

	tentativeMatch := false
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
		// string placeholders do not cause an immediate match if the node has
		// branches.  if they did then they would be acting in the same way as
		// the %* placeholder and any subsequent branches will not be
		// considered at all. we do however flag a tentative match. in this
		// way, if none of the branches cause a better match, then this match
		// will do

		tentativeMatch = true
		match = n.branch == nil

	case "%F":
		// TODO: check for file existance

		// see commentary for %S above

		tentativeMatch = true
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
	if !match && n.branch != nil {
		for bi := range n.branch {
			// recursing into the validate function means we need to use the
			// same token as above. Unget() prepares the tokens object for
			// that.
			tokens.Unget()

			if n.branch[bi].validate(tokens, true) == nil {
				match = true
				return nil
			}
		}

		// there's no explicit match in any of the matches. if we've
		// encountered a tentative match however, we can use that
		match = tentativeMatch
	}

	if !match {
		// there's no match but the speculative flags means we were half
		// expecting it. return error without further consideration of whether
		// node is an optional group
		if speculative {
			return err
		}

		// if we've not found anything in any branches and this is an optional
		// group, then claim that we have matched this group and prepare tokens
		// object for additional nodes. if group is not optional then return
		// error.
		if n.group == groupOptional {
			match = true
			tokens.Unget()
		} else {
			return err
		}
	}

	// input does match this node. check nodes that follow on.
	for ni := range n.next {
		err = n.next[ni].validate(tokens, false)
		if err != nil {
			return err
		}
	}

	return nil
}

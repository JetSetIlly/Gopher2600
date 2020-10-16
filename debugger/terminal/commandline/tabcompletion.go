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

// #tab #completion

import (
	"fmt"
	"strconv"
	"strings"
)

// TabCompletion should be initialised once with the instance of Commands it is
// to work with.
type TabCompletion struct {
	cmds *Commands

	matches []string
	match   int

	lastCompletion string
}

// NewTabCompletion initialises a new TabCompletion instance. Completion works
// best if Commands has been sorted.
func NewTabCompletion(cmds *Commands) *TabCompletion {
	tc := &TabCompletion{cmds: cmds}
	tc.Reset()
	return tc
}

// Complete transforms the input such that the last word in the input is
// expanded to meet the closest match allowed by the template.  Subsequent
// calls to Complete() without an intervening call to Reset() will cycle
// through the original available options.
func (tc *TabCompletion) Complete(input string) string {
	// split input tokens -- it's easier to work with tokens
	tokens := TokeniseInput(input)

	// common function that polishes off a successful Complete(). not using a
	// deferred function because we don't want to call this in all instances
	endGuess := func() string {
		if tc.match >= 0 {
			tokens.ReplaceEnd(tc.matches[tc.match])
			tc.lastCompletion = fmt.Sprintf("%s ", tokens.String())
		} else {
			// no matches found so completion string is by definition, the same
			// as the input
			tc.lastCompletion = input
		}
		return tc.lastCompletion
	}

	input = strings.TrimRight(input, " ")

	// if the input argument is the same as what we returned last time, then
	// cycle through the options that were compiled last time
	if strings.TrimRight(tc.lastCompletion, " ") == input && tc.match >= 0 {
		tc.match++
		if tc.match >= len(tc.matches) {
			tc.match = 0
		}
		return endGuess()
	}

	// new completion session
	tc.Reset()

	// no need to to anything if input ends with a space
	if strings.HasSuffix(input, " ") {
		return input
	}

	// get first token
	tok, ok := tokens.Get()
	if !ok {
		return input
	}
	tok = strings.ToUpper(tok)

	// look for match
	for i := range tc.cmds.cmds {
		n := tc.cmds.cmds[i]

		// if there is an exact match then recurse into the node looking for
		// where the last token coincides with the node tree
		if tok == n.tag {
			// we may have encountered partial matches earlier in the loop. now
			// that we have found an exact match however, we need to make sure
			// the match list is empty so that we don't erroneously trigger the
			// match-cycling branch above.
			tc.Reset()

			// recurse
			tokens.Unget()
			tc.buildMatches(n, tokens)

			return endGuess()
		}

		// if there is a partial match, then add the current node to the list
		// of matches
		if tokens.IsEnd() && len(tok) < len(n.tag) && tok == n.tag[:len(tok)] {
			tc.matches = append(tc.matches, n.tag)
			tc.match = 0
		}
	}

	return endGuess()
}

// Reset is used to clear an outstanding completion session.
func (tc *TabCompletion) Reset() {
	tc.matches = make([]string, 0)
	tc.match = -1
}

func (tc *TabCompletion) buildMatches(n *node, tokens *Tokens) {
	// if there is no more input then return true (validation has passed) if
	// the node is optional, false if it is required
	tok, ok := tokens.Get()
	if !ok {
		return
	}

	// we cannot do anything with a node with no tag, but if there is a "next"
	// node then we can move immediately to validation of that node instead.
	//
	// empty tags like this, happen as a result of parsing nested groups
	//
	// a node with an empty tag but no next array (or a next array with to
	// many entries) is an illegal node and should not have been parsed
	if n.tag == "" {
		if n.next == nil {
			return
		}

		tokens.Unget()
		for ni := range n.next {
			tc.buildMatches(n.next[ni], tokens)
		}

		for bi := range n.branch {
			tokens.Unget()
			tc.buildMatches(n.branch[bi], tokens)
		}

		return
	}

	var match bool

	switch n.tag {
	case "%N":
		_, err := strconv.ParseInt(tok, 0, 32)
		match = err == nil

	case "%P":
		_, err := strconv.ParseFloat(tok, 32)
		match = err == nil

	case "%S":
		// against expectations, string placeholders do not cause a match. if
		// they did then any subsequent branches will not be considered at all.
		match = false

	case "%F":
		// !!TODO: filename completion

		// see commentary for %S above
		match = false

	default:
		// case sensitive matching
		tok = strings.ToUpper(tok)
		match = tok == n.tag
	}

	// if token doesn't match this node, check branches. if there are no
	// branches, return false (validation has failed)
	if !match {
		// if there is a partial match, then add the current node to the list
		// of matches
		if tokens.IsEnd() && len(tok) < len(n.tag) && tok == n.tag[:len(tok)] {
			tc.matches = append(tc.matches, n.tag)
			tc.match = 0
		}

		if n.branch == nil {
			return
		}

		// take a note of current token position. if the token wanders past
		// this point as a result of a branch then we can see that the branch
		// was deeper then just one token. if this is the case then we can see
		// that the branch was *partially* accepted and that we should not
		// proceed onto next-nodes from here.
		tokenAt := tokens.curr

		for bi := range n.branch {
			// we want to use the current token again so we unget() the last
			// token so that it is available at the beginning of the recursed
			// function
			tokens.Unget()

			tc.buildMatches(n.branch[bi], tokens)
		}

		// the key to this condition is the tokenAt variable. see note above.
		if n.typ == nodeOptional && len(tc.matches) == 0 && tokenAt == tokens.curr {
			tokens.Unget()
		} else {
			return
		}
	}

	// token does match and there are no more tokens to consume so we can add
	// this successful token to the list of matches
	//
	// note that this is specific to tab-completion, validation has no
	// equivalent. the purpose of this is to cause the Complete() function
	// above to replace the last token with a normalised version of that token
	// and to suffix it with a space.
	if tokens.IsEnd() {
		tc.matches = append(tc.matches, tok)
		tc.match = 0
		return
	}

	// token does match this node. check nodes that follow on.
	for nx := range n.next {
		tc.buildMatches(n.next[nx], tokens)
	}

	// no more nodes in the next array. move to the repeat node if there is one
	if n.repeat != nil {
		tc.buildMatches(n.repeat, tokens)
	}
}

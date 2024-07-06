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

package dwarf

import (
	"strings"
)

// SourceLineFragmentType defines how a single SourceLineFragment should be
// interpretted.
type SourceLineFragmentType int

// A list of the valid SourceLineFragmentTypes.
const (
	FragmentCode SourceLineFragmentType = iota
	FragmentComment
	FragmentStringLiteral
)

// SourceLineFragment represents an single part of the entire source line.
type SourceLineFragment struct {
	Type    SourceLineFragmentType
	Content string
}

type fragmentParser struct {
	inCommentBlock bool
	inStringBlock  bool
}

func (fp *fragmentParser) endStringBlock(l *SourceLine, s string) string {
	if s == "" {
		return s
	}

	sp := strings.SplitN(s, `"`, 2)

	if len(sp) == 1 {
		// comment is ongoing - add comment to SourceLine and return
		l.Fragments = append(l.Fragments, SourceLineFragment{
			Type:    FragmentStringLiteral,
			Content: sp[0],
		})
		return ""
	}

	// add comment making sure to include the closing delimiter
	l.Fragments = append(l.Fragments, SourceLineFragment{
		Type:    FragmentStringLiteral,
		Content: sp[0],
	})
	l.Fragments = append(l.Fragments, SourceLineFragment{
		Type:    FragmentStringLiteral,
		Content: `"`,
	})
	fp.inStringBlock = false

	return sp[1]
}

func (fp *fragmentParser) endCommentBlock(l *SourceLine, s string) string {
	if s == "" {
		return s
	}

	sp := strings.SplitN(s, `*/`, 2)

	if len(sp) == 1 {
		// comment is ongoing - add comment to SourceLine and return
		l.Fragments = append(l.Fragments, SourceLineFragment{Type: FragmentComment,
			Content: sp[0],
		})
		return ""
	}

	// add comment making sure to include the closing delimiter
	l.Fragments = append(l.Fragments, SourceLineFragment{
		Type:    FragmentComment,
		Content: sp[0],
	})
	l.Fragments = append(l.Fragments, SourceLineFragment{
		Type:    FragmentComment,
		Content: `*/`,
	})
	fp.inCommentBlock = false

	return sp[1]
}

func (fp *fragmentParser) codeBlock(l *SourceLine, s string) {
	if s == "" {
		return
	}

	if fp.inCommentBlock {
		s = fp.endCommentBlock(l, s)
		if s == "" || fp.inCommentBlock {
			return
		}
	}

	if fp.inStringBlock {
		s = fp.endStringBlock(l, s)
		if s == "" || fp.inStringBlock {
			return
		}
	}

	var sp []string

	// check for comment block start
	sp = strings.SplitN(s, `/*`, 2)
	if len(sp) > 1 {
		// add code to fragments
		l.Fragments = append(l.Fragments, SourceLineFragment{
			Type:    FragmentCode,
			Content: sp[0],
		})

		fp.inCommentBlock = true
		l.Fragments = append(l.Fragments, SourceLineFragment{
			Type:    FragmentComment,
			Content: `/*`,
		})

		fp.codeBlock(l, sp[1])
		return
	}

	// check for single-line comment
	sp = strings.SplitN(s, `//`, 2)
	if len(sp) > 1 {
		// add code to fragments
		l.Fragments = append(l.Fragments, SourceLineFragment{
			Type:    FragmentCode,
			Content: sp[0],
		})

		l.Fragments = append(l.Fragments, SourceLineFragment{
			Type:    FragmentComment,
			Content: `//`,
		})

		l.Fragments = append(l.Fragments, SourceLineFragment{
			Type:    FragmentComment,
			Content: sp[1],
		})

		return
	}

	// check for string block start. care taken not to match a single "
	// contained in a character literal
	sp = strings.SplitN(s, `"`, 2)
	if len(sp) > 1 && !strings.HasSuffix(sp[0], `'`) {
		// add code to fragments
		l.Fragments = append(l.Fragments, SourceLineFragment{
			Type:    FragmentCode,
			Content: sp[0],
		})

		fp.inStringBlock = true
		l.Fragments = append(l.Fragments, SourceLineFragment{
			Type:    FragmentStringLiteral,
			Content: `"`,
		})

		fp.codeBlock(l, sp[1])
		return
	}

	// add code to fragments
	l.Fragments = append(l.Fragments, SourceLineFragment{
		Type:    FragmentCode,
		Content: sp[0],
	})
}

func (fp *fragmentParser) parseLine(l *SourceLine) {
	fp.codeBlock(l, l.PlainContent)
}

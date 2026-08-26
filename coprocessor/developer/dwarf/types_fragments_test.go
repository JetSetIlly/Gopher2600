package dwarf

import (
	"fmt"
	"strings"
	"testing"

	"github.com/jetsetilly/gopher2600/test"
)

func TestFragmentParser(t *testing.T) {
	type testItem struct {
		lines     string
		fragments []SourceLineFragment
	}

	var table []testItem = []testItem{
		// block comment on single line
		{
			lines: "/* comment */",
			fragments: []SourceLineFragment{
				{Type: FragmentComment, Content: "/*"},
				{Type: FragmentComment, Content: " comment "},
				{Type: FragmentComment, Content: "*/"},
			},
		},

		// block comment over multiple lines
		{
			lines: "/* line 1 of comment \n line 2 of comment \n line 3 of comment */",
			fragments: []SourceLineFragment{
				{Type: FragmentComment, Content: "/*"},
				{Type: FragmentComment, Content: " line 1 of comment "},
				{Type: FragmentComment, Content: " line 2 of comment "},
				{Type: FragmentComment, Content: " line 3 of comment "},
				{Type: FragmentComment, Content: "*/"},
			},
		},

		// block comment with surrounding text on a single line
		{
			lines: "foo /* comment */ bar",
			fragments: []SourceLineFragment{
				{Type: FragmentCode, Content: "foo "},
				{Type: FragmentComment, Content: "/*"},
				{Type: FragmentComment, Content: " comment "},
				{Type: FragmentComment, Content: "*/"},
				{Type: FragmentCode, Content: " bar"},
			},
		},

		// single line comment with embedded block comment
		{
			lines: "// foo /* comment */ bar",
			fragments: []SourceLineFragment{
				{Type: FragmentComment, Content: "//"},
				{Type: FragmentComment, Content: " foo /* comment */ bar"},
			},
		},

		// single line comment with leading code and embedded block comment
		{
			lines: "baz // foo /* comment */ bar",
			fragments: []SourceLineFragment{
				{Type: FragmentCode, Content: "baz "},
				{Type: FragmentComment, Content: "//"},
				{Type: FragmentComment, Content: " foo /* comment */ bar"},
			},
		},

		// single line comment with the start of a block comment. block comment ends on
		// following line. (this is an illegal structure in a C like language but we should
		// still test it)
		{
			lines: "// foo /* comment\n*/ bar",
			fragments: []SourceLineFragment{
				{Type: FragmentComment, Content: "//"},
				{Type: FragmentComment, Content: " foo /* comment"},
				{Type: FragmentCode, Content: "*/ bar"},
			},
		},

		// single line comment with embedded block comment
		{
			lines: "/* // comment */ bar",
			fragments: []SourceLineFragment{
				{Type: FragmentComment, Content: "/*"},
				{Type: FragmentComment, Content: " // comment "},
				{Type: FragmentComment, Content: "*/"},
				{Type: FragmentCode, Content: " bar"},
			},
		},

		// text block on a single line
		{
			lines: `"text"`,
			fragments: []SourceLineFragment{
				{Type: FragmentStringLiteral, Content: `"`},
				{Type: FragmentStringLiteral, Content: "text"},
				{Type: FragmentStringLiteral, Content: `"`},
			},
		},

		// text block inside comments. the text block should be ignored
		{
			lines: `// "comment"`,
			fragments: []SourceLineFragment{
				{Type: FragmentComment, Content: "//"},
				{Type: FragmentComment, Content: ` "comment"`},
			},
		},
		{
			lines: `/* "comment"\n*/`,
			fragments: []SourceLineFragment{
				{Type: FragmentComment, Content: "/*"},
				{Type: FragmentComment, Content: ` "comment"\n`},
				{Type: FragmentComment, Content: "*/"},
			},
		},
		{
			// with a leading text block
			lines: `foo "text" /* "comment" */`,
			fragments: []SourceLineFragment{
				{Type: FragmentCode, Content: "foo "},
				{Type: FragmentStringLiteral, Content: `"`},
				{Type: FragmentStringLiteral, Content: "text"},
				{Type: FragmentStringLiteral, Content: `"`},
				{Type: FragmentCode, Content: " "},
				{Type: FragmentComment, Content: "/*"},
				{Type: FragmentComment, Content: ` "comment" `},
				{Type: FragmentComment, Content: "*/"},
			},
		},

		// empty or near empty source lines
		{
			lines:     "",
			fragments: []SourceLineFragment{},
		},
		{
			lines:     "\n",
			fragments: []SourceLineFragment{},
		},
		{
			lines: " \n",
			fragments: []SourceLineFragment{
				{Type: FragmentCode, Content: " "},
			},
		},
	}

	var fp fragmentParser
	for _, item := range table {
		lines := strings.Split(item.lines, "\n")
		var fragments []SourceLineFragment
		for i, s := range lines {
			ln := SourceLine{
				LineNumber:   i,
				PlainContent: s,
			}
			fp.parseLine(&ln)
			fragments = append(fragments, ln.Fragments...)
		}
		if !test.ExpectEquality(t, len(item.fragments), len(fragments)) {
			fmt.Printf("%#v\n", fragments)
		}
		for i := range fragments {
			test.ExpectEquality(t, item.fragments[i], fragments[i])
		}
	}
}

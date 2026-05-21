package dwarf

import (
	_ "embed"
	"strings"
	"testing"

	"github.com/jetsetilly/gopher2600/test"
)

// the test file we use here should probably be more representative of the kind of file we're likely
// to encounter. however, to avoid licensing issues we just use one of the go files we already have.
// the fragmentParser is more-or-less language agnostic

//go:embed "types_fragments.go"
var test_data []byte

func TestFragmentParser(t *testing.T) {
	test_data := strings.Split(string(test_data), "\n")

	var fp fragmentParser
	var lines []SourceLine
	for i, s := range test_data {
		ln := SourceLine{
			LineNumber:   i,
			PlainContent: s,
		}
		fp.parseLine(&ln)
		lines = append(lines, ln)
	}

	// test SourceLine plain content as a baseline. there's no reason why this should fail
	test.ExpectEquality(t, len(test_data), len(lines))
	for i, ln := range lines {
		test.ExpectEquality(t, test_data[i], ln.PlainContent)
	}

	// test SourceLine fragments
	for i, ln := range lines {
		var s strings.Builder
		for _, fr := range ln.Fragments {
			s.WriteString(fr.Content)
		}
		test.ExpectEquality(t, test_data[i], s.String())
	}
}

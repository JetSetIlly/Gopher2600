package paths_test

import (
	"gopher2600/paths"
	"gopher2600/test"
	"testing"
)

func TestPaths(t *testing.T) {
	pth, err := paths.ResourcePath("foo/bar", "baz")
	test.Equate(t, err, nil)
	test.Equate(t, pth, ".gopher2600/foo/bar/baz")

	pth, err = paths.ResourcePath("foo/bar", "")
	test.Equate(t, err, nil)
	test.Equate(t, pth, ".gopher2600/foo/bar")

	pth, err = paths.ResourcePath("", "baz")
	test.Equate(t, err, nil)
	test.Equate(t, pth, ".gopher2600/baz")

	pth, err = paths.ResourcePath("", "")
	test.Equate(t, err, nil)
	test.Equate(t, pth, ".gopher2600")
}

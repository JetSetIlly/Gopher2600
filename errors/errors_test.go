package errors_test

import (
	"fmt"
	"gopher2600/errors"
	"testing"
)

func TestError(t *testing.T) {
	e := errors.New(errors.SetupError, "foo")
	if e.Error() != "setup error: foo" {
		t.Errorf("unexpected error message")
	}

	// packing errors of the same type next to each other causes
	// one of them to be dropped
	f := errors.New(errors.SetupError, e)
	fmt.Println(f.Error())
	if f.Error() != "setup error: foo" {
		t.Errorf("unexpected duplicate error message")
	}
}

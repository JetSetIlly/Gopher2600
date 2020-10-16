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

package curated_test

import (
	"fmt"
	"testing"

	"github.com/jetsetilly/gopher2600/curated"
	"github.com/jetsetilly/gopher2600/test"
)

const testError = "test error: %s"
const testErrorB = "test error B: %s"

func TestDuplicateErrors(t *testing.T) {
	e := curated.Errorf(testError, "foo")
	test.Equate(t, e.Error(), "test error: foo")

	// packing errors of the same type next to each other causes
	// one of them to be dropped
	f := curated.Errorf(testError, e)
	test.Equate(t, f.Error(), "test error: foo")
}

func TestIs(t *testing.T) {
	e := curated.Errorf(testError, "foo")
	test.ExpectedSuccess(t, curated.Is(e, testError))

	// Has() should fail because we haven't included testErrorB anywhere in the error
	test.ExpectedFailure(t, curated.Has(e, testErrorB))

	// packing errors of the same type next to each other causes
	// one of them to be dropped
	f := curated.Errorf(testErrorB, e)
	test.ExpectedFailure(t, curated.Is(f, testError))
	test.ExpectedSuccess(t, curated.Is(f, testErrorB))
	test.ExpectedSuccess(t, curated.Has(f, testError))
	test.ExpectedSuccess(t, curated.Has(f, testErrorB))

	// IsAny should return true for these errors also
	test.ExpectedSuccess(t, curated.IsAny(e))
	test.ExpectedSuccess(t, curated.IsAny(f))
}

func TestPlainErrors(t *testing.T) {
	// test plain errors that haven't been formatted with our errors package

	e := fmt.Errorf("plain test error")
	test.ExpectedFailure(t, curated.IsAny(e))

	const testError = "test error: %s"

	test.ExpectedFailure(t, curated.Has(e, testError))
}

func TestWrapping(t *testing.T) {
	a := 10
	e := curated.Errorf("error: value = %d", a)
	f := curated.Errorf("fatal: %v", e)

	test.ExpectedSuccess(t, curated.Has(f, "error: value = %d"))
	test.ExpectedFailure(t, curated.Is(f, "error: value = %d"))
	test.ExpectedSuccess(t, curated.Has(f, "fatal: %v"))
	test.ExpectedSuccess(t, curated.Is(f, "fatal: %v"))

	test.Equate(t, f.Error(), "fatal: error: value = 10")
}

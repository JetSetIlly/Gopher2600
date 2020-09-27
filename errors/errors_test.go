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

package errors_test

import (
	"fmt"
	"testing"

	"github.com/jetsetilly/gopher2600/errors"
	"github.com/jetsetilly/gopher2600/test"
)

const testError = "test error: %s"
const testErrorB = "test error B: %s"

func TestDuplicateErrors(t *testing.T) {

	e := errors.Errorf(testError, "foo")
	test.Equate(t, e.Error(), "test error: foo")

	// packing errors of the same type next to each other causes
	// one of them to be dropped
	f := errors.Errorf(testError, e)
	test.Equate(t, f.Error(), "test error: foo")
}

func TestIs(t *testing.T) {
	e := errors.Errorf(testError, "foo")
	test.ExpectedSuccess(t, errors.Is(e, testError))

	// Has() should fail because we haven't included testErrorB anywhere in the error
	test.ExpectedFailure(t, errors.Has(e, testErrorB))

	// packing errors of the same type next to each other causes
	// one of them to be dropped
	f := errors.Errorf(testErrorB, e)
	test.ExpectedFailure(t, errors.Is(f, testError))
	test.ExpectedSuccess(t, errors.Is(f, testErrorB))
	test.ExpectedSuccess(t, errors.Has(f, testError))
	test.ExpectedSuccess(t, errors.Has(f, testErrorB))

	// IsAny should return true for these errors also
	test.ExpectedSuccess(t, errors.IsAny(e))
	test.ExpectedSuccess(t, errors.IsAny(f))
}

func TestPlainErrors(t *testing.T) {
	// test plain errors that haven't been formatted with our errors package

	e := fmt.Errorf("plain test error")
	test.ExpectedFailure(t, errors.IsAny(e))

	const testError = "test error: %s"

	test.ExpectedFailure(t, errors.Has(e, testError))
}

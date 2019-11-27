package test

import "testing"

// ExpectedFailure tests argument v for a failure condition suitable for it's
// type. Currentlly support types:
//
//		bool -> bool == false
//		error -> error != nil
//
// If type is nil then the test will fail.
func ExpectedFailure(t *testing.T, v interface{}) bool {
	t.Helper()

	switch v := v.(type) {
	case bool:
		if v {
			t.Errorf("expected failure (bool)")
			return false
		}

	case error:
		if v == nil {
			t.Errorf("expected failure (error)")
			return false
		}

	case nil:
		t.Errorf("expected failure (nil)")
		return false

	default:
		t.Fatalf("unsupported type (%T) for expectation testing", v)
		return false
	}

	return true
}

// ExpectedSuccess tests argument v for a success condition suitable for it's
// type. Currentlly support types:
//
//		bool -> bool == true
//		error -> error == nil
//
// If type is nil then the test will succeed.
func ExpectedSuccess(t *testing.T, v interface{}) bool {
	t.Helper()

	switch v := v.(type) {
	case bool:
		if !v {
			t.Errorf("expected failure (bool)")
			return false
		}

	case error:
		if v != nil {
			t.Errorf("expected failure (error)")
			return false
		}

	case nil:
		return true

	default:
		t.Fatalf("unsupported type (%T) for expectation testing", v)
		return false
	}

	return true
}

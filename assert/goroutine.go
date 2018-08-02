package assert

import (
	"bytes"
	"runtime"
	"strconv"
)

// GetGoRoutineID returns an identify for a goroutine. it returns a result that
// is (a) different between goroutines and (b) consistent for a given
// goroutine. It is undoubtedly useful for but it should only ever be used for
// debugging or testing purposes.
func GetGoRoutineID() uint64 {
	b := make([]byte, 64)
	b = b[:runtime.Stack(b, false)]
	b = bytes.TrimPrefix(b, []byte("goroutine "))
	b = b[:bytes.IndexByte(b, ' ')]
	n, _ := strconv.ParseUint(string(b), 10, 64)
	return n
}

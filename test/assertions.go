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

// +build assertions

package test

import (
	"bytes"
	"fmt"
	"runtime"
	"strconv"
)

func init() {
	fmt.Println("running with active assertions")
}

// AssertMainThread causes a panic if calling function is not the main thread
func AssertMainThread() {
	threadInfoBuffer := make([]byte, 64)
	threadInfoBuffer = threadInfoBuffer[:runtime.Stack(threadInfoBuffer, false)]
	threadInfoBuffer = bytes.TrimPrefix(threadInfoBuffer, []byte("goroutine "))
	threadInfoBuffer = threadInfoBuffer[:bytes.IndexByte(threadInfoBuffer, ' ')]
	n, _ := strconv.ParseUint(string(threadInfoBuffer), 10, 64)

	if n != 1 {
		panic("not called from the main thread")
	}
}

// AssertNonMainThread causes a panic if calling function is the main thread
func AssertNonMainThread() {
	threadInfoBuffer := make([]byte, 64)
	threadInfoBuffer = threadInfoBuffer[:runtime.Stack(threadInfoBuffer, false)]
	threadInfoBuffer = bytes.TrimPrefix(threadInfoBuffer, []byte("goroutine "))
	threadInfoBuffer = threadInfoBuffer[:bytes.IndexByte(threadInfoBuffer, ' ')]
	n, _ := strconv.ParseUint(string(threadInfoBuffer), 10, 64)

	if n == 1 {
		panic("not called from the main thread")
	}
}

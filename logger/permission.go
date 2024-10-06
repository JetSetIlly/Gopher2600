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

package logger

// Permission implementations indicate whether the environment making a log
// request is allowed to create new log entries. Good for controlling when or if
// log entries are to be made
type Permission interface {
	AllowLogging() bool
}

type allow struct{}

func (_ allow) AllowLogging() bool {
	return true
}

// Allow indicates that the logging request should be allowed. A good default to
// use if a log entry should always be made.
var Allow Permission = allow{}

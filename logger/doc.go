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

// Package logger is the central log repository for gopher2600. There is a
// single log for the entire application and can be accessed through the
// package level functions.
//
// New log entries are made with the package level Log() and Logf() functions.
// Both these functions require an implementation of the Permission interface.
// This interface tests whether the environment making the logging request is
// allowed to make new log entries.
//
// The environment.Environment type satisfies the Permission interface. If it's
// not convenient to provide an instance of that type then logging.Allow can be
// used to provide blanket permission to the caller.
//
// The Colorizer type can be used with SetEcho() to output a simply coloured
// log entries (using ANSI control codes).
//
// The logger package should not be used inside any init() function.
package logger

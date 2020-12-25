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

// Package performance contains helper functions relating to performance.
//
// Check() is a quick way of running the emulation for a fixed duration of
// time. It will optionally generate profiling information.
//
// RunProfiler() can be used to generate the various profile types. On it's own
// it will not limit the amount of time the program runs for so it is useful
// for more real-world situations
//
// CalcFPS() calculates frames-per-second in aggregate along with an accuracy
// value (as compared to the television specification). Probably not suitable
// for "live" FPS monitoring.
package performance

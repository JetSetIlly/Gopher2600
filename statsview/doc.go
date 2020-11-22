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

// Package statsview is an optional package that will built only when the
// +statsview build constraint is present
//
//	It provides a HTTP server running locally offering runtime statistics.
//	Underlying funcionality provided by "github.com/go-echarts/statsview"
//
//	After launch, graphical statistics will be viewable at:
//
//		localhost:12600/debug/statsview
//
//	And standard Go pprof statistics available at:
//
//		localhost:12600/debug/pprof/
package statsview

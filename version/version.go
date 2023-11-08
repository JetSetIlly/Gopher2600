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

package version

import (
	"fmt"
	"runtime/debug"
)

// if number is empty then the project was probably not built using the makefile
var number string

// Version contains a the current version number of the project
//
// If the version string is "development" then it means that the project has
// been manually built (ie. not with the makefile). However, there is git
// information and so can be discerned to be part of the development process
//
// If the version string is "unknown" then it means that there is no no version
// number and no git information. This can happen when compiling/running with
// "go run ."
var Version string

// Revision contains the git revision hash. If the source has been modified but
// has not been committed then the Revision string will be suffixed with
// "[modified]"
var Revision string

func init() {
	var vcs bool
	var revision string
	var modified bool

	info, ok := debug.ReadBuildInfo()
	if ok {
		for _, v := range info.Settings {
			switch v.Key {
			case "vcs":
				vcs = true
			case "vcs.revision":
				revision = v.Value
			case "vcs.modified":
				switch v.Value {
				case "true":
					modified = true
				default:
					modified = false
				}
			}
		}
	}

	if revision == "" {
		Revision = "no revision information"
	} else {
		Revision = revision
		if modified {
			Revision = fmt.Sprintf("%s [modified]", Revision)
		}
	}

	if number == "" {
		if vcs {
			Version = "development"
		} else {
			Version = "unknown"
		}
	} else {
		Version = number
	}
}

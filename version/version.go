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

// The name to use when referring to the application
const ApplicationName = "Gopher2600"

// if number is empty then the project was probably not built using the makefile
var number string

// Revision contains the vcs revision. If the source has been modified but
// has not been committed then the Revision string will be suffixed with
// "+dirty"
var revision string

// Version contains a the current version number of the project
//
// If the version string is "unreleased" then it means that the project has
// been manually built (ie. not with the makefile)
//
// If the version string is "local" then it means that there is no no version
// number and no vcs information. This can happen when compiling/running with
// "go run ."
var version string

// Version returns the version string, the revision string and whether this is a
// numbered "release" version. if release is true then the revision information
// should be used sparingly
func Version() (string, string, bool) {
	return version, revision, version == number
}

func init() {
	var vcs bool
	var vcsRevision string
	var vcsModified bool

	info, ok := debug.ReadBuildInfo()
	if ok {
		for _, v := range info.Settings {
			switch v.Key {
			case "vcs":
				vcs = true
			case "vcs.revision":
				vcsRevision = v.Value
			case "vcs.modified":
				switch v.Value {
				case "true":
					vcsModified = true
				default:
					vcsModified = false
				}
			}
		}
	}

	if vcsRevision == "" {
		revision = "no revision information"
	} else {
		revision = vcsRevision
		if vcsModified {
			revision = fmt.Sprintf("%s+dirty", revision)
		}
	}

	if number == "" {
		if vcs {
			version = "unreleased"
		} else {
			version = "local"
		}
	} else {
		version = number
	}
}

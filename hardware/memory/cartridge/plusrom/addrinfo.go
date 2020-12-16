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

package plusrom

import (
	"fmt"
	"net/url"
	"strings"
	"unicode"
)

type AddrInfo struct {
	Host string
	Path string
}

func (ai *AddrInfo) String() string {
	return fmt.Sprintf("http://%s/%s", ai.Host, ai.Path)
}

// CopyAddrInfo returns a new instance of AddrInfo.
func (cart *PlusROM) CopyAddrInfo() AddrInfo {
	return AddrInfo{
		Host: cart.net.ai.Host,
		Path: cart.net.ai.Path,
	}
}

// SetAddrInfo updates the host/path information int the PlusROM.
func (cart *PlusROM) SetAddrInfo(host string, path string) (hostValid bool, pathValid bool) {
	if isHostValid(host) {
		cart.net.ai.Host = host
		hostValid = true
	}

	if isPathValid(path) {
		cart.net.ai.Path = path
		pathValid = true
	}

	return hostValid, pathValid
}

const (
	// max host length(s) defined by DNS specifications
	maxHostLength        = 253
	maxHostElementLength = 63

	// there is no upper limit for path size but 1024 bytes is more than enough
	maxPathLength = 1024
)

func isHostValid(host string) bool {
	if len(host) > maxHostLength {
		return false
	}

	labels := strings.Split(host, ".")
	for _, l := range labels {
		if len(l) < 1 || len(l) > maxHostElementLength {
			return false
		}

		// check for valid characters: letters (upper/lower), digits or hyphen
		for _, c := range l {
			if !isValidHostRune(c) {
				return false
			}
		}

		// a hostname may not start with a hyphen
		if l[0] == '-' {
			return false
		}
	}

	return true
}

func isPathValid(path string) bool {
	if len(path) > maxPathLength {
		return false
	}

	enc := url.PathEscape(path)
	dec, err := url.PathUnescape(enc)
	return err == nil && dec == path
}

func isValidHostRune(c rune) bool {
	return unicode.IsLetter(c) || unicode.IsDigit(c) || c == '-'
}

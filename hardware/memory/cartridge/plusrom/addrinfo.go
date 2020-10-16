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

func (ai AddrInfo) String() string {
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

func isHostValid(host string) bool {
	labels := strings.Split(host, ".")
	for _, l := range labels {
		if len(l) < 1 || len(l) > 63 {
			return false
		}
		for _, c := range l {
			if !unicode.IsLetter(c) && !unicode.IsDigit(c) {
				return false
			}
		}
	}

	return true
}

func isPathValid(path string) bool {
	enc := url.PathEscape(path)
	dec, err := url.PathUnescape(enc)
	return err == nil && dec == path
}

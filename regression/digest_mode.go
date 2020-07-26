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

package regression

import (
	"fmt"
	"strings"
)

// DigestMode specifies what type of digest to generate for the regression
// entry
type DigestMode int

// Valid digest modes. Use String() and ParseDigestMode() to convert to and
// from string representations.
const (
	DigestUndefined DigestMode = iota
	DigestVideoOnly
	DigestAudioOnly
	DigestBoth
)

func (mod DigestMode) String() string {
	switch mod {
	case DigestVideoOnly:
		return "video"
	case DigestAudioOnly:
		return "audio"
	case DigestBoth:
		return "both"
	default:
		return "undefined"
	}
}

// ParseDigestMode converts string to DigestMode represenation
func ParseDigestMode(mode string) (DigestMode, error) {
	switch strings.ToLower(mode) {
	case "video":
		return DigestVideoOnly, nil
	case "audio":
		return DigestAudioOnly, nil
	case "both":
		return DigestBoth, nil
	}

	return DigestUndefined, fmt.Errorf("invalid digest mode field (%s)", mode)
}

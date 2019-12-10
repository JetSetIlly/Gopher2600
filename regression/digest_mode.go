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

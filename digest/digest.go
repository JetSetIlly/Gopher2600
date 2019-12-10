// Package digest contain implementations of television protocol interfaces,
// namely PixelRenderer and AudioMixer, such that a crypographic hash is
// produced. The hash can then be used to compare output input from subsequent
// emulation executions - if a new hash differs from a previously recorded
// value then something has changed. We use this as the basis for regression
// tests and playback verification.
package digest

// Digest implementations should return a cryptographic hash in response to a
// String() request. Generation of the hash achieved via another interface.
type Digest interface {
	Hash() string
	ResetDigest()
}

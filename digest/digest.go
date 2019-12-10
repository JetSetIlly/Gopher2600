// Package digest is used to create mathematical hashes. The two
// implementations of the Digest interface also implement the
// television.PixelRenderer and television.AudioMixer interfaces.
//
// The digest.Video type is used to capture video output while digest.Audio is
// used to capture audio output.
//
// The hashes produced by these types are used from regression tests and for
// verification of playback scripts.
package digest

// Digest implementations compute a mathematical hash, retreivable with the
// Hash() function
type Digest interface {
	Hash() string
	ResetDigest()
}

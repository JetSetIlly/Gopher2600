// Package audio implements the audio generation of the TIA. The implementation
// is taken almost directly from Ron Fries' original implementation, found in
// TIASound.c (easily searchable). The bit patterns are taken from there and
// the channels are mixed in the same way.
//
// Unlike the Fries' implementation, the Mix() function is called every video
// cycle, returning a new sample every 114th video clock. The TIA_Process()
// function in Frie's implementation meanwhile is called to fill a buffer. The
// samepl buffer in this emulation must sit outside of the TIA emulation and
// somwhere inside the television implementation.
package audio

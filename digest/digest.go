package digest

// Digest implementations should return a cryptographic hash in response to a
// String() request. Generation of the hash achieved via another interface.
type Digest interface {
	String() string
	ResetDigest()
}

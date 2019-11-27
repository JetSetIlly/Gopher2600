// Package test contains several helper functions that should make construction
// of test packages a little bit easier and remove boilerplate in many common
// test situations.
//
// The ExpectedFailure and ExpectedSuccess functions test for failure and
// success under generic conditions. The documenation for those functions
// describe the currently supported types.
//
// It is worth describing how the "Expected" functions handle the nil type
// because it is not obvious. The nil type is considered a success and
// consequently will cause ExpectedFailure to fail and ExpectedSuccess to
// suceed. This may not be how we want to interpret nil in all situations but
// because of how errors usually works (nil to indicate no error) we *need* to
// interpret nil in this way.  If the nil was a value of a nil type we wouldn't
// have this problem incidentally, but that isn't how Go is designed (with good
// reason).
//
// The Writer type meanwhile, implements the io.Writer interface and should be
// used to capture output. The Writer.Compare() function can then be used to
// test for equality.
package test

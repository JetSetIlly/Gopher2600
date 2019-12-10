// Package regression facilitates the regression testing of emulation code. By
// adding test results to a database, the tests can be rerun automatically and
// checked for consistancy.
//
// Currently, two main types of test are supported. First the digest test. This
// test runs a ROM for a set number of frames, saving the video or audio hash
// to the test database.
//
// The second test is the Playback test. This is a slightly more complex test
// that replays user input from a previously recorded session. Recorded
// sessions take video hashes on every input trigger and so will succeed or
// fail if something has changed. The regression test automates the process.
//
// The two tests are useful for different ROMs. The digest type is useful if
// the ROM does something immediately, say an image that is stressful on the
// TIA. The playback type is more useful for real world ROMs (ie. games).
package regression

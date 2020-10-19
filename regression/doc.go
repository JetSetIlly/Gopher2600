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

// Package regression facilitates the regression testing of emulation code. By
// adding test results to a database, the tests can be rerun automatically and
// checked for consistancy.
//
// Currently, two main types of test are supported. First the video test. This
// test runs a ROM for a set number of frames. A hash of the final video output
// is created a stored for future comparison.
//
// The second test is the Playback test. This is a slightly more complex test
// that replays user input from a previously recorded session. Recorded
// sessions take video hashes on every input trigger and so will succeed or
// fail if something has changed. The regression test automates the process.
//
// The third test is the Log test. This takes a hash of the log after a set
// number of frames. Test failure for the Log test means that something
// (anything) in the log output has changed.
//
// In addition to its basic function, the video test also supports recording of
// machine state. Four machine states are supported at the moment - TV state,
// RIOT/Ports state, RIOT/Timer and CPU. Aprt from the TV state this doesn't
// fit well with the idea of the video digest and may be separated into a
// completely separate test in the future.
//
// Playback scripts and state scripts are stored in the "regressionScripts"
// directory of the emulator's configuration directory. See the gopher2600
// paths package for details about the configuration directory.
//
// To keep things simple regression runs will be performed in relation to the
// VCS hardware in its default state, in particular no randomisation. The state
// of the VCS in relation to playback regression entries is governed by the
// recorder package.
package regression

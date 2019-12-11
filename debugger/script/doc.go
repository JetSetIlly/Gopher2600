// Package script allows the debugger to record and replay debugging scripts.
// In this package we refer to this as scribing and rescribing, to avoid
// confusion with recording and playback of user input.
//
// (I'm not at all happy about the words scribe and rescribe but there it is. I
// think it's important not to overload terminology too much. Scribe and
// rescribe will do for now. If you can come up with anything better then I'm
// open for suggestions.)
//
// Scripts can of course be handwritten and be rescribed as though they had
// been scribed by the debugger. In this instance however, there is a risk that
// there will be errors - invalid commands will not be written to the script
// file by the Scribe type. On Rescribing, invalid commands will attempt to be
// replayed and the appropriate error message printed to the terminal.
//
// Scribe will also write terminal output to the script file. This is purely
// for the reader of the script file. It serves no real purpose and has no
// effect when Rescribing. It probably should be optional
//
// Scripts can be called when scribing a new script. The output of the
// existing script will not be written to the new script.
//
// The Rescribe type satisfies the terminal.Input and is used as a source for
// the debugger packages input loop.
package script

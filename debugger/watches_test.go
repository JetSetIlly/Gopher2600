package debugger_test

func (trm *mockTerm) testWatches() {
	// debugger starts off with no watches
	trm.sndInput("LIST WATCHES")
	trm.cmpOutput("no watches")

	// add read watch. there should be no output.
	trm.sndInput("WATCH READ 0x80")
	trm.cmpOutput("")

	// try to re-add the same watch
	trm.sndInput("WATCH READ 0x80")
	trm.cmpOutput("already being watched (0x0080 read)")

	// list watches
	trm.sndInput("LIST WATCHES")
	trm.cmpOutput(" 0: 0x0080 read")

	// try to re-add the same watch but with a different event selector
	trm.sndInput("WATCH WRITE 0x80")
	trm.cmpOutput("")

	// list watches
	trm.sndInput("LIST WATCHES")
	trm.cmpOutput(" 0: 0x0080 read/write")
}

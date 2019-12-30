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
	trm.cmpOutput("already being watched (0x0080 (RAM) read)")

	// list watches
	trm.sndInput("LIST WATCHES")
	trm.cmpOutput(" 0: 0x0080 (RAM) read")

	// try to re-add the same watch but with a different event selector
	trm.sndInput("WATCH WRITE 0x80")
	trm.cmpOutput("")

	// list watches
	trm.sndInput("LIST WATCHES")
	trm.cmpOutput(" 1: 0x0080 (RAM) write")

	// clear watches
	trm.sndInput("CLEAR WATCHES")
	trm.cmpOutput("watches cleared")

	// no watches after succesful clear
	trm.sndInput("LIST WATCHES")
	trm.cmpOutput("no watches")

	// try adding an invalid read address by symbol
	trm.sndInput("WATCH READ VSYNC")
	trm.cmpOutput("invalid watch address: VSYNC")

	// add address by symbol. no read/write modifier means it tries
	trm.sndInput("WATCH VSYNC")
	trm.cmpOutput("")

	// last item in list watches should be the new entry
	trm.sndInput("LIST WATCHES")
	trm.cmpOutput(" 0: 0x0000 (VSYNC) (TIA) write")

	// add address by symbol. no read/write modifier means it tries
	// plus a specific value
	trm.sndInput("WATCH VSYNC 0x1")
	trm.cmpOutput("")

	// last item in list watches should be the new entry
	trm.sndInput("LIST WATCHES")
	trm.cmpOutput(" 1: 0x0000 (VSYNC) (TIA) write (value=0x01)")
}

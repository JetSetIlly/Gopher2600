package debugger_test

func (trm *mockTerm) testTraps() {
	// debugger starts off with no traps
	trm.sndInput("LIST TRAPS")
	trm.cmpOutput("no traps")

	// add a trap. there should be no output.
	trm.sndInput("TRAP a")
	trm.cmpOutput("")

	// add same trap again. using uppercase this time.
	trm.sndInput("TRAP A")
	trm.cmpOutput("trap already exists (A)")

	// list traps. compare last line.
	trm.sndInput("LIST TRAPS")
	trm.cmpOutput(" 0: A")
}

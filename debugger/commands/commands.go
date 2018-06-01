package commands

// keywords
const (
	KeywordHelp          = "HELP"
	KeywordBreak         = "BREAK"
	KeywordTrap          = "TRAP"
	KeywordOnHalt        = "ONHALT"
	KeywordMemMap        = "MEMMAP"
	KeywordQuit          = "QUIT"
	KeywordReset         = "RESET"
	KeywordRun           = "RUN"
	KeywordStep          = "STEP"
	KeywordStepMode      = "STEPMODE"
	KeywordTerse         = "TERSE"
	KeywordVerbose       = "VERBOSE"
	KeywordVerbosity     = "VERBOSITY"
	KeywordDebuggerState = "DEBUGGERSTATE"
	KeywordCPU           = "CPU"
	KeywordPeek          = "PEEK"
	KeywordRIOT          = "RIOT"
	KeywordTIA           = "TIA"
	KeywordTV            = "TV"
	KeywordBall          = "BALL"
	KeywordDisplay       = "DISPLAY"
)

// TopLevel is the list of top-level commands
var TopLevel = []string{"HELP", "BREAK", "TRAP", "ONHALT", "MEMMAP", "QUIT", "RESET", "RUN", "STEP", "STEPMODE", "TERSE", "VERBOSE", "VERBOSITY", "DEBUGGERSTATE", "CPU", "PEEK", "RIOT", "TIA", "TV", "BALL", "DISPLAY"}

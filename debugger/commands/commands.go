package commands

// keywords
const (
	KeywordHelp          = "HELP"
	KeywordInsert        = "INSERT"
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
	KeywordPlayfield     = "PLAYFIELD"
	KeywordDisplay       = "DISPLAY"
	KeywordScript        = "SCRIPT"
)

// DebuggerCommand is the list of top-level commands
var DebuggerCommand = []string{
	KeywordHelp,
	KeywordInsert,
	KeywordBreak,
	KeywordTrap,
	KeywordOnHalt,
	KeywordMemMap,
	KeywordQuit,
	KeywordReset,
	KeywordRun,
	KeywordStep,
	KeywordStepMode,
	KeywordTerse,
	KeywordVerbose,
	KeywordVerbosity,
	KeywordDebuggerState,
	KeywordCPU,
	KeywordPeek,
	KeywordRIOT,
	KeywordTIA,
	KeywordTV,
	KeywordBall,
	KeywordPlayfield,
	KeywordDisplay,
	KeywordScript}

// completionArg and associated constants definitions, are used to inform what
// tab-completion style is expected for each command's argument
type completionArg int

const (
	compArgDebuggerCommand completionArg = iota
	compArgFile
)

// CompletionsOpts defines how tab completion should work for each argument of a
// TopLevel command
var completionsOpts = map[string]completionArg{
	KeywordHelp:   compArgDebuggerCommand,
	KeywordInsert: compArgFile,
	KeywordScript: compArgFile}

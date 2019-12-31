package debugger

type quantumMode int

const (
	quantumCPU quantumMode = iota
	quantumVideo
)

func (mode quantumMode) String() string {
	switch mode {
	case quantumCPU:
		return "CPU"
	case quantumVideo:
		return "VIDEO"
	default:
		return "unrecognised quantum mode"
	}
}

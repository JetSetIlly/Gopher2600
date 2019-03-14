package hardware

import (
	"fmt"
	"gopher2600/hardware/cpu"
	"gopher2600/hardware/cpu/result"
	"gopher2600/hardware/memory"
	"gopher2600/hardware/peripherals"
	"gopher2600/hardware/riot"
	"gopher2600/hardware/tia"
	"gopher2600/television"
	"sync/atomic"
)

// VCS struct is the main container for the emulated components of the VCS
type VCS struct {
	MC   *cpu.CPU
	Mem  *memory.VCSMemory
	TIA  *tia.TIA
	RIOT *riot.RIOT

	// tv is not part of the VCS but is attached to it
	TV television.Television

	panel      *peripherals.Panel
	Controller *peripherals.Stick

	// treat the side effects of the CPU after every CPU cycle (correct) or
	// only at the end of each instruction (wrong)
	//
	// NOTE: for correct emulation this flag should definitely be false. the
	// flag is provided only so to demonstrate the difference between the two
	// strategies
	monolithCPU bool
}

// NewVCS creates a new VCS and everything associated with the hardware. It is
// used for all aspects of emulation: debugging sessions, and regular play
func NewVCS(tv television.Television) (*VCS, error) {
	var err error

	vcs := new(VCS)
	vcs.TV = tv

	vcs.Mem, err = memory.NewVCSMemory()
	if err != nil {
		return nil, err
	}

	vcs.MC, err = cpu.NewCPU(vcs.Mem)
	if err != nil {
		return nil, err
	}

	vcs.TIA = tia.NewTIA(vcs.TV, vcs.Mem.TIA)
	if vcs.TIA == nil {
		return nil, fmt.Errorf("can't create TIA")
	}

	vcs.RIOT = riot.NewRIOT(vcs.Mem.RIOT)
	if vcs.RIOT == nil {
		return nil, fmt.Errorf("can't create RIOT")
	}

	vcs.panel = peripherals.NewPanel(vcs.Mem.RIOT)
	if vcs.panel == nil {
		return nil, fmt.Errorf("can't create console control panel")
	}

	// TODO: better contoller support
	vcs.Controller = peripherals.NewStick(vcs.Mem.TIA, vcs.Mem.RIOT, vcs.panel)
	if vcs.Controller == nil {
		return nil, fmt.Errorf("can't create stick controller")
	}

	return vcs, nil
}

// AttachCartridge loads a cartridge (given by filename) into the emulators memory
func (vcs *VCS) AttachCartridge(filename string) error {
	if filename == "" {
		vcs.Mem.Cart.Eject()
	} else {
		err := vcs.Mem.Cart.Attach(filename)
		if err != nil {
			return err
		}
	}
	err := vcs.Reset()
	if err != nil {
		return err
	}

	return nil
}

// Reset emulates the reset switch on the console panel
//  - reset the CPU
//  - destroy and create the TIA and RIOT
//  - load reset address into the PC
func (vcs *VCS) Reset() error {
	if err := vcs.MC.Reset(); err != nil {
		return err
	}

	// TODO: consider implementing tia.Reset and riot.Reset instead of
	// recreating the two components

	vcs.TIA = tia.NewTIA(vcs.TV, vcs.Mem.TIA)
	if vcs.TIA == nil {
		return fmt.Errorf("can't create TIA")
	}

	vcs.RIOT = riot.NewRIOT(vcs.Mem.RIOT)
	if vcs.RIOT == nil {
		return fmt.Errorf("can't create RIOT")
	}

	err := vcs.MC.LoadPCIndirect(memory.AddressReset)
	if err != nil {
		return err
	}

	return nil
}

// Step the emulator state one CPU instruction
// -- we can put this function in a loop for an effective debugging loop
// ths videoCycleCallback function for an additional callback point in the
// debugger.
func (vcs *VCS) Step(videoCycleCallback func(*result.Instruction) error) (int, *result.Instruction, error) {
	var r *result.Instruction
	var err error

	// the number of CPU cycles that have elapsed.  note this is *not* the same
	// as Instructionresult.ActualCycles because in the event of a WSYNC
	// cpuCycles will continue to accumulate until the WSYNC has been resolved.
	cpuCycles := 0

	if vcs.monolithCPU {
		r, err = vcs.MC.ExecuteInstruction(func(*result.Instruction) {})
		if err != nil {
			return cpuCycles, nil, err
		}

		cpuCycles = r.ActualCycles

		vcs.RIOT.ReadRIOTMemory()
		vcs.TIA.ReadTIAMemory()

		for i := 0; i < cpuCycles; i++ {
			vcs.RIOT.Step()
			vcs.MC.RdyFlg = vcs.TIA.StepVideoCycle()
			vcs.MC.RdyFlg = vcs.TIA.StepVideoCycle()
			vcs.MC.RdyFlg = vcs.TIA.StepVideoCycle()
		}

		// CPU has been left in the unready state - continue cycling the VCS hardware
		// until the CPU is ready
		for !vcs.MC.RdyFlg {
			cpuCycles++
			vcs.RIOT.Step()
			vcs.MC.RdyFlg = vcs.TIA.StepVideoCycle()
			vcs.MC.RdyFlg = vcs.TIA.StepVideoCycle()
			vcs.MC.RdyFlg = vcs.TIA.StepVideoCycle()
		}

	} else {
		// the cpu calls the cycleVCS function after every CPU cycle. the cycleVCS
		// function defines the order of operation for the rest of the VCS for
		// every CPU cycle.
		cycleVCS := func(r *result.Instruction) {
			cpuCycles++

			// run riot only once per CPU cycle
			// TODO: not sure when in the video cycle sequence it should be run
			// TODO: is this something that can drift, thereby causing subtly different
			// results / graphical effects? is this what RSYNC is for?

			vcs.RIOT.ReadRIOTMemory()
			vcs.RIOT.Step()

			// read tia memory just once and before we cycle the tia
			vcs.TIA.ReadTIAMemory()

			// three color clocks per CPU cycle so we run video cycle three times
			vcs.TIA.StepVideoCycle()
			videoCycleCallback(r)

			vcs.TIA.StepVideoCycle()
			videoCycleCallback(r)

			vcs.MC.RdyFlg = vcs.TIA.StepVideoCycle()
			videoCycleCallback(r)
		}

		r, err = vcs.MC.ExecuteInstruction(cycleVCS)
		if err != nil {
			return cpuCycles, nil, err
		}

		// CPU has been left in the unready state - continue cycling the VCS hardware
		// until the CPU is ready
		for !vcs.MC.RdyFlg {
			cycleVCS(r)
		}
	}

	return cpuCycles, r, nil
}

// RunConcurrent sets the emulation running with as many different sub-systems
// running concurrently as possible
// NOTE: this doesn't really work as yet. it's here because it's an intersting
// idea.
func (vcs *VCS) RunConcurrent(running *atomic.Value) error {
	var err error

	var triggerRIOT chan bool
	var triggerTIA chan bool

	triggerRIOT = make(chan bool)
	triggerTIA = make(chan bool)

	go func() {
		for {
			<-triggerRIOT
			vcs.RIOT.ReadRIOTMemory()
			vcs.RIOT.Step()
			triggerRIOT <- true
		}
	}()

	go func() {
		for {
			<-triggerTIA
			// read tia memory just once and before we cycle the tia
			vcs.TIA.ReadTIAMemory()

			// three color clocks per CPU cycle so we run video cycle three times
			vcs.TIA.StepVideoCycle()
			vcs.TIA.StepVideoCycle()
			vcs.MC.RdyFlg = vcs.TIA.StepVideoCycle()
			triggerTIA <- true
		}
	}()

	cycleVCS := func(r *result.Instruction) {
		triggerTIA <- true
		triggerRIOT <- true
		<-triggerTIA
		<-triggerRIOT
	}

	for running.Load().(int) >= 0 {
		_, err = vcs.MC.ExecuteInstruction(cycleVCS)
		if err != nil {
			return err
		}
	}

	return nil
}

// Run sets the emulation running as quickly as possible
// -- using an atomic value rather than a mutex
func (vcs *VCS) Run(running *atomic.Value) error {
	var err error

	cycleVCS := func(r *result.Instruction) {
		vcs.RIOT.ReadRIOTMemory()
		vcs.RIOT.Step()
		vcs.TIA.ReadTIAMemory()
		vcs.TIA.StepVideoCycle()
		vcs.TIA.StepVideoCycle()
		vcs.MC.RdyFlg = vcs.TIA.StepVideoCycle()
	}

	for running.Load().(int) >= 0 {
		_, err = vcs.MC.ExecuteInstruction(cycleVCS)
		if err != nil {
			return err
		}
	}

	return nil
}

// RunForFrameCount sets emulator running for the specified number of frames
// - not used by the debugger because traps and steptraps are more flexible
// - useful for fps and regression tests
func (vcs *VCS) RunForFrameCount(numFrames int) error {
	tvs, err := vcs.TV.GetState(television.ReqFramenum)
	if err != nil {
		return err
	}

	targetFrame := tvs.(int) + numFrames

	for tvs.(int) != targetFrame {
		_, _, err = vcs.Step(func(*result.Instruction) error { return nil })
		tvs, err = vcs.TV.GetState(television.ReqFramenum)
		if err != nil {
			return err
		}
	}

	return nil
}

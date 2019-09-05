package performance

import (
	"gopher2600/errors"
	"os"
	"runtime"
	"runtime/pprof"
)

// ProfileCPU runs supplied function "through" the pprof CPU profiler
func ProfileCPU(outFile string, run func() error) error {
	// write cpu profile
	f, err := os.Create(outFile)
	if err != nil {
		return errors.NewFormattedError(errors.PerformanceError, err)
	}
	err = pprof.StartCPUProfile(f)
	if err != nil {
		return errors.NewFormattedError(errors.PerformanceError, err)
	}
	defer pprof.StopCPUProfile()

	return run()
}

// ProfileMem takes a snapshot of memory and writes to outFile
func ProfileMem(outFile string) error {
	f, err := os.Create(outFile)
	if err != nil {
		return errors.NewFormattedError(errors.PerformanceError, err)
	}
	runtime.GC()
	err = pprof.WriteHeapProfile(f)
	if err != nil {
		return errors.NewFormattedError(errors.PerformanceError, err)
	}
	f.Close()

	return nil
}

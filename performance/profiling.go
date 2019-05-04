package performance

import (
	"gopher2600/errors"
	"os"
	"runtime"
	"runtime/pprof"
)

func cpuProfile(profile bool, outFile string, run func() error) error {
	if profile {
		// write cpu profile
		f, err := os.Create(outFile)
		if err != nil {
			return errors.NewFormattedError(errors.FPSError, err)
		}
		err = pprof.StartCPUProfile(f)
		if err != nil {
			return errors.NewFormattedError(errors.FPSError, err)
		}
		defer pprof.StopCPUProfile()
	}

	return run()
}

func memProfile(profile bool, outFile string) error {
	if profile {
		f, err := os.Create(outFile)
		if err != nil {
			return errors.NewFormattedError(errors.FPSError, err)
		}
		runtime.GC()
		err = pprof.WriteHeapProfile(f)
		if err != nil {
			return errors.NewFormattedError(errors.FPSError, err)
		}
		f.Close()
	}

	return nil
}

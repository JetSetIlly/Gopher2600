// This file is part of Gopher2600.
//
// Gopher2600 is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// Gopher2600 is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with Gopher2600.  If not, see <https://www.gnu.org/licenses/>.

package performance

import (
	"os"
	"runtime"
	"runtime/pprof"

	"github.com/jetsetilly/gopher2600/curated"
)

// ProfileCPU runs supplied function "through" the pprof CPU profiler.
func ProfileCPU(outFile string, run func() error) (rerr error) {
	// write cpu profile
	f, err := os.Create(outFile)
	if err != nil {
		return curated.Errorf("performance; %v", err)
	}
	defer func() {
		err := f.Close()
		if err != nil {
			rerr = curated.Errorf("performance; %v", err)
		}
	}()

	err = pprof.StartCPUProfile(f)
	if err != nil {
		return curated.Errorf("performance; %v", err)
	}
	defer pprof.StopCPUProfile()

	return run()
}

// ProfileMem takes a snapshot of memory and writes to outFile.
func ProfileMem(outFile string) (rerr error) {
	f, err := os.Create(outFile)
	if err != nil {
		return curated.Errorf("performance; %v", err)
	}
	defer func() {
		err := f.Close()
		if err != nil {
			rerr = curated.Errorf("performance; %v", err)
		}
	}()

	runtime.GC()
	err = pprof.WriteHeapProfile(f)
	if err != nil {
		return curated.Errorf("performance; %v", err)
	}

	return nil
}

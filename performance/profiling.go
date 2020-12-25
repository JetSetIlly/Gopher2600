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
	"fmt"
	"os"
	"runtime"
	"runtime/pprof"
	"runtime/trace"
	"strings"

	"github.com/jetsetilly/gopher2600/curated"
)

// Profile is used to specify the type of profiling to perform by RunProfiler().
type Profile int

// List of valid Profile values. Values can be combined.
const (
	ProfileNone  Profile = 0b0000
	ProfileCPU   Profile = 0b0001
	ProfileMem   Profile = 0b0010
	ProfileTrace Profile = 0b0100
	ProfileAll   Profile = 0b0111
)

// RunProfiler runs supplied function "through" the requested Profile types.
func RunProfiler(profile Profile, filenameHeader string, run func() error) (rerr error) {
	if profile&ProfileCPU == ProfileCPU {
		f, err := os.Create(fmt.Sprintf("%s_cpu.profile", filenameHeader))
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
	}

	if profile&ProfileTrace == ProfileTrace {
		f, err := os.Create(fmt.Sprintf("%s_trace.profile", filenameHeader))
		if err != nil {
			return curated.Errorf("performance; %v", err)
		}
		defer func() {
			err := f.Close()
			if err != nil {
				rerr = curated.Errorf("performance; %v", err)
			}
		}()

		err = trace.Start(f)
		if err != nil {
			return curated.Errorf("performance; %v", err)
		}
		defer trace.Stop()
	}

	if profile&ProfileMem == ProfileMem {
		f, err := os.Create(fmt.Sprintf("%s_mem.profile", filenameHeader))
		if err != nil {
			return curated.Errorf("performance; %v", err)
		}
		defer func() {
			err := f.Close()
			if err != nil {
				rerr = curated.Errorf("performance; %v", err)
			}
		}()

		defer func() {
			runtime.GC()
			err = pprof.WriteHeapProfile(f)
			if err != nil {
				rerr = curated.Errorf("performance; %v", err)
			}
		}()
	}

	return run()
}

// ParseProfileString checks a returns a profile value in response to a profile
// string. profile string can contain any combination of "cpu", "mem", "trace"
// separated by commas. For example:
//
//	"cpu,mem"
//
// Will return the numeric value produced by bitwise ORing of ProfileCPU and
// PorfileMem.
//
// For convenience, a profile string of "all" will select all profilers at
// once; a string of "none" will be ignored.
func ParseProfileString(profile string) (Profile, error) {
	p := ProfileNone

	s := strings.Split(profile, ",")
	for _, t := range s {
		switch strings.TrimSpace(strings.ToLower(t)) {
		case "none":
		case "all":
			p |= ProfileAll
		case "cpu":
			p |= ProfileCPU
		case "mem":
			p |= ProfileMem
		case "trace":
			p |= ProfileTrace
		default:
			return p, curated.Errorf("profile: unknown profile type (%s)", t)
		}
	}

	return p, nil
}

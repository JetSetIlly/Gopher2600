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

// Package performance contains helper functions relating to
// performance. This includes CPU and memory profile generation.
package performance

import (
	"fmt"
	"io"
	"time"

	"github.com/jetsetilly/gopher2600/cartridgeloader"
	"github.com/jetsetilly/gopher2600/curated"
	"github.com/jetsetilly/gopher2600/hardware"
	"github.com/jetsetilly/gopher2600/hardware/television"
	"github.com/jetsetilly/gopher2600/hardware/television/signal"
	"github.com/jetsetilly/gopher2600/setup"
)

// Check is a very rough and ready calculation of the emulator's performance.
func Check(output io.Writer, profile bool, tv *television.Television, runTime string, cartload cartridgeloader.Loader) error {
	var err error

	// create vcs using the tv created above
	vcs, err := hardware.NewVCS(tv)
	if err != nil {
		return curated.Errorf("performance; %v", err)
	}

	// attach cartridge to te vcs
	err = setup.AttachCartridge(vcs, cartload)
	if err != nil {
		return curated.Errorf("performance; %v", err)
	}

	// parse supplied duration
	duration, err := time.ParseDuration(runTime)
	if err != nil {
		return curated.Errorf("performance; %v", err)
	}

	// get starting frame number (should be 0)
	startFrame := tv.GetState(signal.ReqFramenum)

	// run for specified period of time
	runner := func() error {
		// setup trigger that expires when duration has elapsed. signals true
		// when duration has expired. signals false to indicate that
		// performance measurement should start
		timerChan := make(chan bool)

		// force a two second leadtime to allow framerate to settle down and
		// then restart timer for the specified duration
		go func() {
			time.AfterFunc(2*time.Second, func() {
				// signal parent function that 2 second leadtime has elapsed
				timerChan <- false

				// race condition when GetState() is called
				time.AfterFunc(duration, func() {
					timerChan <- true
				})
			})
		}()

		// run until specified time elapses
		err = vcs.Run(func() (bool, error) {
			for {
				select {
				case v := <-timerChan:
					// timerChan has returned true, which means measurement
					// period has finished, return false to cause vcs.Run() to
					// return
					if v {
						return false, nil
					}

					// timerChan has returned false so start measurement of
					// performance by noting the current television frame
					startFrame = tv.GetState(signal.ReqFramenum)
				default:
					return true, nil
				}
			}
		})
		if err != nil {
			return curated.Errorf("performance; %v", err)
		}
		return nil
	}

	// launch runner directly or through the CPU profiler, depending on
	// supplied arguments
	if profile {
		err = ProfileCPU("cpu.profile", runner)
	} else {
		err = runner()
	}
	if err != nil {
		return curated.Errorf("performance; %v", err)
	}

	// get ending frame number
	endFrame := vcs.TV.GetState(signal.ReqFramenum)

	// calculate performance
	numFrames := endFrame - startFrame
	fps, accuracy := CalcFPS(tv, numFrames, duration.Seconds())
	output.Write([]byte(fmt.Sprintf("%.2f fps (%d frames in %.2f seconds) %.1f%%\n", fps, numFrames, duration.Seconds(), accuracy)))

	// create memory profile depending on supplied arguments
	if profile {
		err = ProfileMem("mem.profile")
	}

	return err
}

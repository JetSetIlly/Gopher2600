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
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/jetsetilly/gopher2600/cartridgeloader"
	"github.com/jetsetilly/gopher2600/debugger/govern"
	"github.com/jetsetilly/gopher2600/environment"
	"github.com/jetsetilly/gopher2600/hardware"
	"github.com/jetsetilly/gopher2600/hardware/television"
	"github.com/jetsetilly/gopher2600/setup"
)

// sentinal error returned by Run() loop.
var timedOut = errors.New("performance timed out")

// Check the performance of the emulator using the supplied cartridge.
//
// Emulation will run of specificed duration and will create a cpu, memory
// profile, a trace (or a combination of those) as defined by the Profile
// argument.
func Check(output io.Writer, profile Profile, cartload cartridgeloader.Loader, spec string, uncapped bool, duration string) error {
	var err error

	tv, err := television.NewTelevision(spec)
	if err != nil {
		return err
	}
	defer tv.End()

	// set fps cap on television
	tv.SetFPSCap(!uncapped)

	// create vcs
	vcs, err := hardware.NewVCS(environment.MainEmulation, tv, nil, nil)
	if err != nil {
		return fmt.Errorf("performance: %w", err)
	}

	// attach cartridge to the vcs
	err = setup.AttachCartridge(vcs, cartload)
	if err != nil {
		return fmt.Errorf("performance: %w", err)
	}

	// parse supplied duration
	dur, err := time.ParseDuration(duration)
	if err != nil {
		return fmt.Errorf("performance: %w", err)
	}

	// get starting frame number (should be 0)
	startFrame := tv.GetCoords().Frame

	// run for specified period of time
	runner := func() error {
		// setup trigger that expires when duration has elapsed. signals true
		// when duration has expired. signals false to indicate that
		// performance measurement should start
		timerChan := make(chan bool)

		// force a two second leadtime to allow framerate to settle down and
		// then restart timer for the specified duration
		//
		// the two second leadtime will put false on the timerChan. the
		// conclusion of the reset of the time will put true on the timerChan.
		go func() {
			time.AfterFunc(2*time.Second, func() {
				// signal parent function that 2 second leadtime has elapsed
				timerChan <- false

				// race condition when GetCoords() is called
				time.AfterFunc(dur, func() {
					timerChan <- true
				})
			})
		}()

		// only check for end of measurement period every PerformanceBrake CPU
		// instructions. checking the timerChan is relatively expensive (worth
		// a frame a two every 5 seconds)
		performanceBrake := 0

		// run until specified time elapses
		err = vcs.Run(func() (govern.State, error) {
			for {
				performanceBrake++
				if performanceBrake >= hardware.PerformanceBrake {
					performanceBrake = 0

					select {
					case v := <-timerChan:
						// timerChan has returned true, which means measurement
						// period has finished, return false to cause vcs.Run() to
						// return
						if v {
							return govern.Ending, timedOut
						}

						// timerChan has returned false which indicates that the
						// leadtime has concluded. this means the performance
						// measurement has begun and we should record the start
						// frame.
						startFrame = tv.GetCoords().Frame
					default:
						return govern.Running, nil
					}
				}

				return govern.Running, nil
			}
		})
		return err
	}

	// launch runner directly or through the CPU profiler, depending on
	// supplied arguments
	err = RunProfiler(profile, "performance", runner)
	if err != nil && !errors.Is(err, timedOut) {
		return fmt.Errorf("performance: %w", err)
	}

	// get ending frame number
	endFrame := vcs.TV.GetCoords().Frame

	// calculate performance
	numFrames := endFrame - startFrame
	fps, accuracy := CalcFPS(tv, numFrames, dur.Seconds())
	output.Write([]byte(fmt.Sprintf("%.2f fps (%d frames in %.2f seconds) %.1f%%\n", fps, numFrames, dur.Seconds(), accuracy)))

	return nil
}

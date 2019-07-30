package performance

import (
	"fmt"
	"gopher2600/errors"
	"gopher2600/gui"
	"gopher2600/gui/sdl"
	"gopher2600/hardware"
	"gopher2600/television"
	"io"
	"time"
)

// Check is a very rough and ready calculation of the emulator's performance
func Check(output io.Writer, profile bool, cartridgeFile string, display bool, tvType string, scaling float32, runTime string) error {
	var ftv television.Television
	var err error

	// create the "correct" type of TV depending on whether the display flag is
	// set or not
	if display {
		ftv, err = sdl.NewGUI(tvType, scaling, nil)
		if err != nil {
			return errors.NewFormattedError(errors.FPSError, err)
		}

		err = ftv.(gui.GUI).SetFeature(gui.ReqSetVisibility, true)
		if err != nil {
			return errors.NewFormattedError(errors.FPSError, err)
		}
	} else {
		ftv, err = television.NewStellaTelevision(tvType)
		if err != nil {
			return errors.NewFormattedError(errors.FPSError, err)
		}
	}

	// create vcs using the tv created above
	vcs, err := hardware.NewVCS(ftv)
	if err != nil {
		return errors.NewFormattedError(errors.FPSError, err)
	}

	// attach cartridge to te vcs
	err = vcs.AttachCartridge(cartridgeFile)
	if err != nil {
		return errors.NewFormattedError(errors.FPSError, err)
	}

	// parse supplied duration
	duration, err := time.ParseDuration(runTime)
	if err != nil {
		return errors.NewFormattedError(errors.FPSError, err)
	}

	// get starting frame number (should be 0)
	startFrame, err := ftv.GetState(television.ReqFramenum)
	if err != nil {
		return errors.NewFormattedError(errors.FPSError, err)
	}

	// run for specified period of time
	err = cpuProfile(profile, "cpu.profile", func() error {
		// setup trigger that expires when duration has elapsed
		timesUp := make(chan bool)

		// force a two second leadtime to allow framerate to settle down and
		// then restart timer for the specified duration
		go func() {
			time.AfterFunc(2*time.Second, func() {
				startFrame, _ = ftv.GetState(television.ReqFramenum)
				time.AfterFunc(duration, func() {
					timesUp <- true
				})
			})
		}()

		// run until specified time elapses
		err = vcs.Run(func() (bool, error) {
			select {
			case v := <-timesUp:
				return !v, nil
			default:
				return true, nil
			}
		})
		if err != nil {
			return errors.NewFormattedError(errors.FPSError, err)
		}
		return nil
	})
	if err != nil {
		return errors.NewFormattedError(errors.FPSError, err)
	}

	// get ending frame number
	endFrame, err := vcs.TV.GetState(television.ReqFramenum)
	if err != nil {
		return errors.NewFormattedError(errors.FPSError, err)
	}

	numFrames := endFrame - startFrame
	fps, accuracy := CalcFPS(ftv, numFrames, duration.Seconds())
	output.Write([]byte(fmt.Sprintf("%.2f fps (%d frames in %.2f seconds) %.1f%%\n", fps, numFrames, duration.Seconds(), accuracy)))

	return memProfile(profile, "mem.profile")
}

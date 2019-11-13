package performance

import (
	"fmt"
	"gopher2600/cartridgeloader"
	"gopher2600/errors"
	"gopher2600/hardware"
	"gopher2600/setup"
	"gopher2600/television"
	"io"
	"time"
)

// Check is a very rough and ready calculation of the emulator's performance
func Check(output io.Writer, profile bool, tv television.Television, runTime string, cartload cartridgeloader.Loader) error {
	var err error

	// create vcs using the tv created above
	vcs, err := hardware.NewVCS(tv)
	if err != nil {
		return errors.New(errors.PerformanceError, err)
	}

	// attach cartridge to te vcs
	err = setup.AttachCartridge(vcs, cartload)
	if err != nil {
		return errors.New(errors.PerformanceError, err)
	}

	// parse supplied duration
	duration, err := time.ParseDuration(runTime)
	if err != nil {
		return errors.New(errors.PerformanceError, err)
	}

	// get starting frame number (should be 0)
	startFrame, err := tv.GetState(television.ReqFramenum)
	if err != nil {
		return errors.New(errors.PerformanceError, err)
	}

	// run for specified period of time
	runner := func() error {
		// setup trigger that expires when duration has elapsed
		timesUp := make(chan bool)

		// force a two second leadtime to allow framerate to settle down and
		// then restart timer for the specified duration
		go func() {
			time.AfterFunc(2*time.Second, func() {
				startFrame, _ = tv.GetState(television.ReqFramenum)
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
			return errors.New(errors.PerformanceError, err)
		}
		return nil
	}

	if profile {
		err = ProfileCPU("cpu.profile", runner)
	} else {
		err = runner()
	}

	if err != nil {
		return errors.New(errors.PerformanceError, err)
	}

	// get ending frame number
	endFrame, err := vcs.TV.GetState(television.ReqFramenum)
	if err != nil {
		return errors.New(errors.PerformanceError, err)
	}

	numFrames := endFrame - startFrame
	fps, accuracy := CalcFPS(tv, numFrames, duration.Seconds())
	output.Write([]byte(fmt.Sprintf("%.2f fps (%d frames in %.2f seconds) %.1f%%\n", fps, numFrames, duration.Seconds(), accuracy)))

	if profile {
		err = ProfileMem("mem.profile")
	}

	return err
}

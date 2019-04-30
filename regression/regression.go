package regression

import (
	"fmt"
	"gopher2600/errors"
	"gopher2600/hardware"
	"gopher2600/television/renderers"
	"io"
)

// RegressDeleteCartridge removes a cartridge from the regression db
func RegressDeleteCartridge(cartridgeFile string) error {
	db, err := startSession()
	if err != nil {
		return err
	}
	defer db.endSession(true)

	if _, ok := db.entries[cartridgeFile]; ok == false {
		return errors.NewFormattedError(errors.RegressionEntryDoesNotExist, cartridgeFile)
	}

	delete(db.entries, cartridgeFile)

	return nil
}

// RegressAddCartridge adds a cartridge to the regression db
func RegressAddCartridge(cartridgeFile string, tvMode string, numOfFrames int) error {
	return addCartridge(cartridgeFile, tvMode, numOfFrames, false)
}

// RegressUpdateCartridge updates a entry (or adds it if it doesn't exist)
func RegressUpdateCartridge(cartridgeFile string, tvMode string, numOfFrames int) error {
	return addCartridge(cartridgeFile, tvMode, numOfFrames, true)
}

// RegressRunTests runs all the tests in the regression database
func RegressRunTests(output io.Writer, failOnError bool) (int, int, error) {
	db, err := startSession()
	if err != nil {
		return -1, -1, err
	}
	defer db.endSession(false)

	numSucceed := 0
	numFail := 0
	for _, entry := range db.entries {
		digest, err := run(entry.cartridgePath, entry.tvMode, entry.numOFrames)

		if err != nil || entry.screenDigest != digest {
			if err == nil {
				err = errors.NewFormattedError(errors.RegressionEntryFail, entry)
			}

			numFail++
			if failOnError {
				return numSucceed, numFail, err
			}
			if output != nil {
				output.Write([]byte(fmt.Sprintf("fail: %s\n", err)))
			}

		} else {
			numSucceed++
			if output != nil {
				output.Write([]byte(fmt.Sprintf("succeed: %s\n", entry)))
			}
		}
	}

	return numSucceed, numFail, nil
}

func run(cartridgeFile string, tvMode string, numOfFrames int) (string, error) {
	tv, err := renderers.NewDigestTV(tvMode, nil)
	if err != nil {
		return "", fmt.Errorf("error preparing television: %s", err)
	}

	vcs, err := hardware.NewVCS(tv)
	if err != nil {
		return "", fmt.Errorf("error preparing VCS: %s", err)
	}

	err = vcs.AttachCartridge(cartridgeFile)
	if err != nil {
		return "", err
	}

	err = vcs.RunForFrameCount(numOfFrames)
	if err != nil {
		return "", err
	}

	// output current digest
	return fmt.Sprintf("%s", tv), nil
}

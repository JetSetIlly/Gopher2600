package regression

import (
	"fmt"
	"gopher2600/errors"
	"io"
)

// RegressDeleteCartridge removes a cartridge from the regression db
func RegressDeleteCartridge(cartridgeFile string) error {
	db, err := startSession()
	if err != nil {
		return err
	}
	defer db.endSession()

	key, err := getCartridgeHash(cartridgeFile)
	if err != nil {
		return err
	}

	if _, ok := db.entries[key]; ok == false {
		return errors.NewFormattedError(errors.RegressionEntryDoesNotExist, cartridgeFile)
	}

	delete(db.entries, key)

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

// RegressRunTests runs the
func RegressRunTests(output io.Writer, failOnError bool) (int, int, error) {
	db, err := startSession()
	if err != nil {
		return -1, -1, err
	}
	defer db.endSession()

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

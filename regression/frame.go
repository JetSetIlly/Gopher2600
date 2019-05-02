package regression

import (
	"fmt"
	"gopher2600/errors"
	"gopher2600/hardware"
	"gopher2600/television/renderers"
	"strconv"
	"strings"
)

const (
	frameFieldCartName int = iota
	frameFieldTVtype
	frameFieldNumFrames
	frameFieldDigest
	numFrameFields
)

// FrameRegression is the simplest regression type
type FrameRegression struct {
	key          int
	CartFile     string
	TVtype       string
	NumFrames    int
	screenDigest string
}

func (reg FrameRegression) getID() string {
	return "frame"
}

func newFrameRegression(key int, csv string) (*FrameRegression, error) {
	reg := &FrameRegression{key: key}

	// loop through file until EOF is reached
	fields := strings.Split(csv, ",")
	reg.screenDigest = fields[frameFieldDigest]
	reg.CartFile = fields[frameFieldCartName]
	reg.TVtype = fields[frameFieldTVtype]

	var err error

	reg.NumFrames, err = strconv.Atoi(fields[frameFieldNumFrames])
	if err != nil {
		msg := fmt.Sprintf("invalid numFrames field [%s]", fields[frameFieldNumFrames])
		return nil, errors.NewFormattedError(errors.RegressionDBError, msg)
	}

	return reg, nil
}

func (reg *FrameRegression) setKey(key int) {
	reg.key = key
}

func (reg FrameRegression) getKey() int {
	return reg.key
}

func (reg *FrameRegression) getCSV() string {
	return fmt.Sprintf("%s%s%s%s%s%s%d%s%s",
		csvLeader(reg), fieldSep,
		reg.CartFile, fieldSep,
		reg.TVtype, fieldSep,
		reg.NumFrames, fieldSep,
		reg.screenDigest,
	)
}

func (reg FrameRegression) String() string {
	return fmt.Sprintf("[%s] %s [%s] frames=%d", reg.getID(), reg.CartFile, reg.TVtype, reg.NumFrames)
}

func (reg *FrameRegression) regress(newRegression bool) (bool, error) {
	tv, err := renderers.NewDigestTV(reg.TVtype, nil)
	if err != nil {
		return false, errors.NewFormattedError(errors.RegressionFail, err)
	}

	vcs, err := hardware.NewVCS(tv)
	if err != nil {
		return false, errors.NewFormattedError(errors.RegressionFail, err)
	}

	err = vcs.AttachCartridge(reg.CartFile)
	if err != nil {
		return false, errors.NewFormattedError(errors.RegressionFail, err)
	}

	err = vcs.RunForFrameCount(reg.NumFrames)
	if err != nil {
		return false, errors.NewFormattedError(errors.RegressionFail, err)
	}

	if newRegression {
		reg.screenDigest = tv.String()
		return true, nil
	}

	return tv.String() == reg.screenDigest, nil
}

func (reg FrameRegression) cleanUp() {
}

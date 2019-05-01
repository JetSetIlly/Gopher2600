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

// FrameRecord is the simplest regression database record type
type FrameRecord struct {
	key           int
	CartridgeFile string
	TVtype        string
	NumFrames     int
	screenDigest  string
}

func (rec FrameRecord) getID() string {
	return "frame"
}

func newFrameRecord(key int, csv string) (*FrameRecord, error) {
	rec := &FrameRecord{key: key}

	// loop through file until EOF is reached
	fields := strings.Split(csv, ",")
	rec.screenDigest = fields[frameFieldDigest]
	rec.CartridgeFile = fields[frameFieldCartName]
	rec.TVtype = fields[frameFieldTVtype]

	var err error

	rec.NumFrames, err = strconv.Atoi(fields[frameFieldNumFrames])
	if err != nil {
		msg := fmt.Sprintf("invalid numFrames field [%s]", fields[frameFieldNumFrames])
		return nil, errors.NewFormattedError(errors.RegressionDBError, msg)
	}

	return rec, nil
}

func (rec *FrameRecord) setKey(key int) {
	rec.key = key
}

func (rec FrameRecord) getKey() int {
	return rec.key
}

func (rec *FrameRecord) getCSV() string {
	return fmt.Sprintf("%s%s%s%s%s%s%d%s%s",
		csvLeader(rec), fieldSep,
		rec.CartridgeFile, fieldSep,
		rec.TVtype, fieldSep,
		rec.NumFrames, fieldSep,
		rec.screenDigest,
	)
}

func (rec FrameRecord) String() string {
	return fmt.Sprintf("%s [%s] frames=%d", rec.CartridgeFile, rec.TVtype, rec.NumFrames)
}

func (rec *FrameRecord) regress(newRecord bool) (bool, error) {
	tv, err := renderers.NewDigestTV(rec.TVtype, nil)
	if err != nil {
		return false, fmt.Errorf("error preparing television: %s", err)
	}

	vcs, err := hardware.NewVCS(tv)
	if err != nil {
		return false, fmt.Errorf("error preparing VCS: %s", err)
	}

	err = vcs.AttachCartridge(rec.CartridgeFile)
	if err != nil {
		return false, err
	}

	err = vcs.RunForFrameCount(rec.NumFrames)
	if err != nil {
		return false, err
	}

	if newRecord {
		rec.screenDigest = tv.String()
		return true, nil
	}

	return tv.String() == rec.screenDigest, nil
}

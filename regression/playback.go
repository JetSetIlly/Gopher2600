package regression

import (
	"fmt"
	"strings"
)

const (
	playbackFieldScript int = iota
	numPlaybackFields
)

// PlaybackRecord i
type PlaybackRecord struct {
	key    int
	Script string
}

func (rec PlaybackRecord) getID() string {
	return "playback"
}

func newPlaybackRecord(key int, csv string) (*PlaybackRecord, error) {
	// loop through file until EOF is reached
	fields := strings.Split(csv, ",")

	rec := &PlaybackRecord{
		key:    key,
		Script: fields[playbackFieldScript],
	}

	return rec, nil
}

func (rec *PlaybackRecord) setKey(key int) {
	rec.key = key
}

func (rec PlaybackRecord) getKey() int {
	return rec.key
}

func (rec *PlaybackRecord) getCSV() string {
	return fmt.Sprintf("%s%s%s",
		csvLeader(rec), fieldSep,
		rec.Script,
	)
}

func (rec PlaybackRecord) String() string {
	return rec.Script
}

func (rec *PlaybackRecord) regress(newRecord bool) (bool, error) {
	return true, nil
}

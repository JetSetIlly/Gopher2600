package wavwriter

import (
	"gopher2600/errors"
	tia "gopher2600/hardware/tia/audio"
	"os"

	"github.com/go-audio/audio"
	"github.com/go-audio/wav"
)

// WavWriter implemented the television.AudioMixer interface
type WavWriter struct {
	filename string
	buffer   []int8
}

// New is the preferred method of initialisation for the Audio2Wav type
func New(filename string) (*WavWriter, error) {
	aw := &WavWriter{
		filename: filename,
		buffer:   make([]int8, 0, 0),
	}

	return aw, nil
}

// SetAudio implements the television.AudioMixer interface
func (aw *WavWriter) SetAudio(audioData uint8) error {
	aw.buffer = append(aw.buffer, int8(int16(audioData)-127))
	return nil
}

// FlushAudio implements the television.AudioMixer interface
func (aw *WavWriter) FlushAudio() error {
	return nil
}

// PauseAudio implements the television.AudioMixer interface
func (aw *WavWriter) PauseAudio(pause bool) error {
	return nil
}

// EndMixing implements the television.AudioMixer interface
func (aw *WavWriter) EndMixing() error {
	err := aw.FlushAudio()
	if err != nil {
		return errors.New(errors.WavWriter, err)
	}

	f, err := os.Create(aw.filename)
	if err != nil {
		return errors.New(errors.WavWriter, err)
	}
	defer f.Close()

	// see audio commentary in sdlplay package for thinking around sample rates

	enc := wav.NewEncoder(f, tia.SampleFreq, 8, 1, 1)
	if enc == nil {
		return errors.New(errors.WavWriter, "bad parameters for wav encoding")
	}
	defer enc.Close()

	buf := audio.PCMBuffer{
		Format: &audio.Format{
			NumChannels: 1,
			SampleRate:  31403,
		},
		I8:             aw.buffer,
		DataType:       audio.DataTypeI8,
		SourceBitDepth: 8,
	}

	err = enc.Write(buf.AsIntBuffer())
	if err != nil {
		return errors.New(errors.WavWriter, err)
	}

	return nil
}

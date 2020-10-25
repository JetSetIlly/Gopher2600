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

package supercharger

import (
	"fmt"
	"io"

	"github.com/go-audio/audio"
	"github.com/go-audio/wav"
	"github.com/hajimehoshi/go-mp3"
	"github.com/jetsetilly/gopher2600/cartridgeloader"
	"github.com/jetsetilly/gopher2600/logger"
)

// according to the go-mp3 docs:
//
// "The stream is always formatted as 16bit (little endian) 2 channels even if
// the source is single channel MP3. Thus, a sample always consists of 4
// bytes.".
const mp3NumChannels = 2
const mp3SourceBitDepth = 16

// getPCM() tries to interprets the data in the cartridgeloader as sound data,
// first as WAV data and then as MP3 data. returns an error if data doesn't
// apepar to either.
//
// !!TODO: handle multiple mp3/wav files to create one single multiload tape.
func getPCM(cl cartridgeloader.Loader) (*audio.Float32Buffer, error) {
	// try interpreting data as a WAV file
	wavDec := wav.NewDecoder(&pcmDecoder{data: cl.Data})
	if wavDec.IsValidFile() {
		b, err := wavDec.FullPCMBuffer()
		if err != nil {
			return nil, fmt.Errorf("soundload: wav file: %v", err)
		}

		logger.Log(soundloadLogTag, "loading from wav file")
		return b.AsFloat32Buffer(), nil
	}

	// try interpreting data as an MP3 file
	mp3Dec, err := mp3.NewDecoder(&pcmDecoder{data: cl.Data})
	if err != nil {
		return nil, fmt.Errorf("soundload: mp3 file: %v", err)
	}

	b := &audio.IntBuffer{
		Format: &audio.Format{
			NumChannels: mp3NumChannels,
			SampleRate:  mp3Dec.SampleRate(),
		},
		Data:           make([]int, 0, mp3Dec.Length()),
		SourceBitDepth: mp3SourceBitDepth,
	}

	d := make([]byte, 4068)
	for {
		n, err := mp3Dec.Read(d)
		if err == io.EOF {
			break // for loop
		}

		if err != nil {
			return nil, fmt.Errorf("soundload: mp3 file: %v", err)
		}

		// two bytes per sample per channel
		for i := 0; i < n; i += 2 {
			// little endian 16 bit sample
			f := int(d[i]) | (int((d[i+1])) << 8)

			// adjust value if it is not zero (same as interpreting
			// as two's complement)
			if f != 0 {
				f -= 32768
			}

			b.Data = append(b.Data, f)
		}
	}

	logger.Log(soundloadLogTag, "loading from mp3 file")
	return b.AsFloat32Buffer(), nil
}

// pcmDecoder is an implementation of io.ReadSeeker.
//
// this is used by wav.NewDecoder() and mp3.NewDecoder() to load data from the
// cartridgeloader data.
type pcmDecoder struct {
	data   []uint8
	offset int
}

// Read is an implementation of io.ReadSeeker.
func (d *pcmDecoder) Read(p []byte) (int, error) {
	// return EOF error if no more bytes to copy
	if d.offset >= len(d.data) {
		return 0, io.EOF
	}

	// end byte of the data we're copying from
	n := d.offset + len(p)

	if n > len(d.data) {
		n = len(d.data)
	}

	// copy data to p
	copy(p, d.data[d.offset:n])

	// how many bytes were read
	n -= d.offset

	// advance offset
	d.offset += n
	if d.offset > len(d.data) {
		d.offset = len(d.data)
	}

	return n, nil
}

// Seek is an implementation of io.ReadSeeker.
func (d *pcmDecoder) Seek(offset int64, whence int) (int64, error) {
	switch whence {
	case io.SeekStart:
		d.offset = int(offset)
	case io.SeekCurrent:
		d.offset += int(offset)
	case io.SeekEnd:
		d.offset = len(d.data) - int(offset)
	}

	if d.offset < 0 {
		d.offset = 0
		return int64(d.offset), io.EOF
	}

	return int64(d.offset), nil
}

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
	"path/filepath"
	"strings"

	"github.com/go-audio/wav"

	"github.com/hajimehoshi/go-mp3"
	"github.com/jetsetilly/gopher2600/cartridgeloader"
	"github.com/jetsetilly/gopher2600/environment"
	"github.com/jetsetilly/gopher2600/logger"
)

type pcmData struct {
	totalTime  float64 // in seconds
	sampleRate float64

	// data is mono data (taken from the left channel in the case of stero
	// source files)
	data []float32
}

func getPCM(env *environment.Environment, cl cartridgeloader.Loader) (pcmData, error) {
	p := pcmData{
		data: make([]float32, 0),
	}

	switch strings.ToLower(filepath.Ext(cl.Filename)) {
	case ".wav":
		dec := wav.NewDecoder(cl)
		if dec == nil {
			return p, fmt.Errorf("wav: error decoding")
		}

		if !dec.IsValidFile() {
			return p, fmt.Errorf("wav: not a valid wav file")
		}

		logger.Log(env, soundloadLogTag, "loading from wav file")

		// load all data at once
		buf, err := dec.FullPCMBuffer()
		if err != nil {
			return p, fmt.Errorf("soundload: wav: %w", err)
		}
		floatBuf := buf.AsFloat32Buffer()

		// copy first channel only of data stream
		p.data = make([]float32, 0, len(floatBuf.Data)/int(dec.NumChans))
		for i := 0; i < len(floatBuf.Data); i += int(dec.NumChans) {
			p.data = append(p.data, floatBuf.Data[i])
		}

		// sample rate
		p.sampleRate = float64(dec.SampleRate)

		// total time of recording in seconds
		dur, err := dec.Duration()
		if err != nil {
			return p, fmt.Errorf("wav: %w", err)
		}
		p.totalTime = dur.Seconds()

	case ".mp3":
		dec, err := mp3.NewDecoder(cl)
		if err != nil {
			return p, fmt.Errorf("mp3: %w", err)
		}

		logger.Log(env, soundloadLogTag, "loading from mp3 file")

		err = nil
		chunk := make([]byte, 4096)
		for err != io.EOF {
			var chunkLen int
			chunkLen, err = dec.Read(chunk)
			if err != nil && err != io.EOF {
				return p, fmt.Errorf("mp3: %w", err)
			}

			// index increment of 4 because:
			//  - two bytes per sample per channel
			//  - we only want the left channel
			//  - if we only wanted the right channel we could start with an
			//		index of 2
			for i := 2; i < chunkLen; i += 4 {
				// little endian 16 bit sample
				f := int(chunk[i]) | (int((chunk[i+1])) << 8)

				// adjust value if it is not zero (same as interpreting
				// as two's complement)
				if f != 0 {
					f -= 32768
				}

				p.data = append(p.data, float32(f))
			}
		}

		// according to the go-mp3 docs:
		//
		// "The stream is always formatted as 16bit (little endian) 2 channels even if
		// the source is single channel MP3. Thus, a sample always consists of 4
		// bytes.".
		p.sampleRate = float64(dec.SampleRate())

		// total time of recording in seconds
		p.totalTime = float64(len(p.data)) / p.sampleRate
	}

	logger.Logf(env, soundloadLogTag, "sample rate: %0.2fHz", p.sampleRate)
	logger.Logf(env, soundloadLogTag, "total time: %.02fs", p.totalTime)

	return p, nil
}

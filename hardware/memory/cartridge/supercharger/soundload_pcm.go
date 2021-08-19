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
	"math"
	"path/filepath"
	"strings"

	// "github.com/go-audio/audio"
	// "github.com/go-audio/wav"

	"github.com/hajimehoshi/go-mp3"
	"github.com/jetsetilly/gopher2600/cartridgeloader"
	"github.com/jetsetilly/gopher2600/logger"
	"github.com/youpy/go-wav"
)

type pcmData struct {
	totalTime  float64
	sampleRate float64

	// data is mono data (taken from the left channel in the case of stero
	// source files)
	data []float32
}

func getPCM(cl cartridgeloader.Loader) (pcmData, error) {
	p := pcmData{
		data: make([]float32, 0),
	}

	switch strings.ToLower(filepath.Ext(cl.Filename)) {
	case ".wav":
		dec := wav.NewReader(cl.StreamedData)

		format, err := dec.Format()
		if err != nil {
			return p, fmt.Errorf("soundload: wav file: %v", err)
		}

		logger.Log(soundloadLogTag, "loading from wav file")

		p.sampleRate = float64(format.SampleRate)

		// adjust zero value for unsigned data. with wav data we can assume
		// that a bit-depth of 8 or less is unsigned. bottom of page 60:
		//
		// http://www-mmsp.ece.mcgill.ca/Documents/AudioFormats/WAVE/Docs/riffmci.pdf
		//
		// data for higher bit-depths is signed and we do not need to do make
		// any adjustments
		adjust := 0
		if format.BitsPerSample == 8 {
			adjust = int(math.Pow(2.0, float64(format.BitsPerSample)/2))
		}

		logger.Log(soundloadLogTag, "using left channel only")

		err = nil
		for err != io.EOF {
			var samples []wav.Sample
			samples, err = dec.ReadSamples()
			if err != nil && err != io.EOF {
				return p, fmt.Errorf("soundload: wav file: %v", err)
			}
			for _, s := range samples {
				p.data = append(p.data, float32(s.Values[0]-adjust))
			}
		}

	case ".mp3":
		dec, err := mp3.NewDecoder(cl.StreamedData)
		if err != nil {
			return p, fmt.Errorf("soundload: mp3 file: %v", err)
		}

		logger.Log(soundloadLogTag, "loading from mp3 file")

		// according to the go-mp3 docs:
		//
		// "The stream is always formatted as 16bit (little endian) 2 channels even if
		// the source is single channel MP3. Thus, a sample always consists of 4
		// bytes.".
		p.sampleRate = float64(dec.SampleRate())

		logger.Log(soundloadLogTag, "using left channel only")

		err = nil
		chunk := make([]byte, 4096)
		for err != io.EOF {
			var chunkLen int
			chunkLen, err = dec.Read(chunk)
			if err != nil && err != io.EOF {
				return p, fmt.Errorf("soundload: mp3 file: %v", err)
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
	}

	p.totalTime = float64(len(p.data)) / p.sampleRate

	logger.Logf(soundloadLogTag, "sample rate: %0.2fHz", p.sampleRate)
	logger.Logf(soundloadLogTag, "total time: %.02fs", p.totalTime)

	return p, nil
}

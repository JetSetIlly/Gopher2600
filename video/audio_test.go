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

package video

import (
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-audio/wav"
	"github.com/jetsetilly/gopher2600/hardware/television/frameinfo"
	"github.com/jetsetilly/gopher2600/hardware/television/signal"
	"github.com/jetsetilly/gopher2600/hardware/television/specification"
)

// audio from an aux source pushed during a frame should be recorded in that frame, at its own pitch
// and at the expected level
func TestAuxAudio(t *testing.T) {
	const (
		frames     = 60
		startFrame = 30
		srcRate    = 8192
		tone       = 440.0
		amplitude  = 0.5
	)

	spec := specification.SpecNTSC
	fi := frameinfo.Current{Spec: spec, TotalScanlines: spec.ScanlinesTotal}

	path := filepath.Join(t.TempDir(), "aux.wav")
	au, err := newAudio(path, fi)
	if err != nil {
		t.Fatal(err)
	}

	// silent TIA, and a tone from startFrame, pushed in the amount of source
	// samples that each frame's emulation time produces
	var produced float64
	var n int
	for f := range frames {
		if err := au.SetAudio(make([]signal.AudioSignalAttributes, au.clip)); err != nil {
			t.Fatal(err)
		}
		produced += srcRate / float64(spec.RefreshRate)
		var buf []float32
		for ; float64(n) < produced; n++ {
			var v float32
			if f >= startFrame {
				v = float32(amplitude * math.Sin(2*math.Pi*tone*float64(n)/srcRate))
			}
			buf = append(buf, v)
		}
		if err := au.PushAuxAudio(buf, srcRate); err != nil {
			t.Fatal(err)
		}
		if err := au.NewFrame(fi); err != nil {
			t.Fatal(err)
		}
	}
	if err := au.EndMixing(); err != nil {
		t.Fatal(err)
	}

	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	dec := wav.NewDecoder(f)
	pcm, err := dec.FullPCMBuffer()
	if err != nil {
		t.Fatal(err)
	}
	x := pcm.Data
	rate := float64(dec.SampleRate)

	if len(x) != frames*au.clip {
		t.Errorf("recorded %d samples, expected %d", len(x), frames*au.clip)
	}

	// the tone starts in startFrame
	first := -1
	for i, v := range x {
		if v > 1000 || v < -1000 {
			first = i
			break
		}
	}
	if first/au.clip != startFrame {
		t.Errorf("sound starts in frame %d, expected frame %d", first/au.clip, startFrame)
	}

	// at its own pitch, measured from zero crossings
	crossings := 0
	for i := first + 1; i < len(x); i++ {
		if (x[i-1] < 0) != (x[i] < 0) {
			crossings++
		}
	}
	freq := float64(crossings) / 2 / (float64(len(x)-first) / rate)
	if math.Abs(freq-tone) > 5 {
		t.Errorf("recorded tone is %.1fHz, expected %.0fHz", freq, tone)
	}

	// and at the expected level: full scale aux audio is as loud as the loudest TIA output
	peak := 0
	for _, v := range x {
		peak = max(peak, v, -v)
	}
	want := int(math.Round(amplitude * (0x7fff >> 1)))
	if math.Abs(float64(peak-want)) > float64(want)/50 {
		t.Errorf("peak level is %d, expected about %d", peak, want)
	}
}

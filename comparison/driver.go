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

package comparison

import (
	"image"
	"image/color"

	"github.com/jetsetilly/gopher2600/hardware/television"
	"github.com/jetsetilly/gopher2600/hardware/television/frameinfo"
	"github.com/jetsetilly/gopher2600/hardware/television/signal"
	"github.com/jetsetilly/gopher2600/hardware/television/specification"
)

type driver struct {
	tv *television.Television

	frameInfo frameinfo.Current

	img     [2]*image.RGBA
	cropImg [2]*image.RGBA
	audio   [2][]uint8
	swapIdx bool

	sync chan bool
	ack  chan bool
	quit chan error
}

func newDriver(tv *television.Television) driver {
	drv := driver{
		tv:   tv,
		sync: make(chan bool),
		ack:  make(chan bool),
		quit: make(chan error),
	}

	drv.img[0] = image.NewRGBA(image.Rect(0, 0, specification.ClksScanline, specification.AbsoluteMaxScanlines))
	drv.img[1] = image.NewRGBA(image.Rect(0, 0, specification.ClksScanline, specification.AbsoluteMaxScanlines))

	// start with a NTSC television as default
	drv.Resize(frameinfo.NewCurrent(specification.SpecNTSC))
	drv.Reset()

	return drv
}

// Resize implements the television.PixelRenderer interface.
func (drv *driver) Resize(frameInfo frameinfo.Current) error {
	drv.frameInfo = frameInfo
	crop := image.Rect(
		specification.ClksHBlank, frameInfo.VisibleTop,
		specification.ClksHBlank+specification.ClksVisible, frameInfo.VisibleBottom,
	)

	drv.cropImg[0] = drv.img[0].SubImage(crop).(*image.RGBA)
	drv.cropImg[1] = drv.img[1].SubImage(crop).(*image.RGBA)

	return nil
}

// NewFrame implements the television.PixelRenderer interface.
func (drv *driver) NewFrame(frameInfo frameinfo.Current) error {
	drv.frameInfo = frameInfo
	drv.swapIdx = !drv.swapIdx

	select {
	case drv.sync <- true:
	case err := <-drv.quit:
		return err
	}

	select {
	case <-drv.ack:
	case err := <-drv.quit:
		return err
	}

	return nil
}

// NewScanline implements the television.PixelRenderer interface.
func (drv *driver) NewScanline(scanline int) error {
	return nil
}

// SetPixels implements the television.PixelRenderer interface.
func (drv *driver) SetPixels(sig []signal.SignalAttributes, last int) error {
	var col color.RGBA
	var offset int

	var pix []uint8
	if drv.swapIdx {
		pix = drv.img[0].Pix
	} else {
		pix = drv.img[1].Pix
	}

	for i := range sig {
		// handle VBLANK by setting pixels to black. we also manually handle
		// NoSignal in the same way
		if sig[i].VBlank || sig[i].Index == signal.NoSignal {
			col = drv.frameInfo.Spec.GetColor(signal.ZeroBlack)
		} else {
			col = drv.frameInfo.Spec.GetColor(sig[i].Color)
		}

		// small cap improves performance, see https://golang.org/issue/27857
		s := pix[offset : offset+3 : offset+3]
		s[0] = col.R
		s[1] = col.G
		s[2] = col.B

		offset += 4
	}
	return nil
}

// Reset implements the television.PixelRenderer AND the
// television.AudioMixer
func (m *driver) Reset() {
	// clear pixels. setting the alpha channel so we don't have to later (the
	// alpha channel never changes)
	for i := range m.img {
		for y := 0; y < m.img[i].Bounds().Size().Y; y++ {
			for x := 0; x < m.img[i].Bounds().Size().X; x++ {
				m.img[i].SetRGBA(x, y, color.RGBA{0, 0, 0, 255})
			}
		}
	}
}

// EndRendering implements the television.PixelRenderer interface.
func (drv *driver) EndRendering() error {
	return nil
}

// SetAudio implements the television.AudioMixer interface.
func (drv *driver) SetAudio(sig []signal.AudioSignalAttributes) error {
	var idx int
	if drv.swapIdx {
		idx = 0
	} else {
		idx = 1
	}
	for _, s := range sig {
		drv.audio[idx] = append(drv.audio[idx], s.AudioChannel0, s.AudioChannel1)
	}
	return nil
}

// EndMixing implements the television.AudioMixer interface.
func (drv *driver) EndMixing() error {
	return nil
}

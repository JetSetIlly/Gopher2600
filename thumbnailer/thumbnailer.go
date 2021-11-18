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

// Package thumbnailer can be used to create a single thumbnail or a series of
// thumbnail images with the Create() function.
package thumbnailer

import (
	"image"
	"image/color"

	"github.com/jetsetilly/gopher2600/cartridgeloader"
	"github.com/jetsetilly/gopher2600/curated"
	"github.com/jetsetilly/gopher2600/emulation"
	"github.com/jetsetilly/gopher2600/hardware"
	"github.com/jetsetilly/gopher2600/hardware/television"
	"github.com/jetsetilly/gopher2600/hardware/television/signal"
	"github.com/jetsetilly/gopher2600/hardware/television/specification"
	"github.com/jetsetilly/gopher2600/logger"
	"github.com/jetsetilly/gopher2600/rewind"
)

// Label that can be used in the EmulationLabel field of the cartridgeloader.Loader type. to indicate
// that the cartridge has been loaded into the thumbnailer emulation
const EmulationLabel = "thumbnailer"

// Thumbnailer type handles the emulation necessary for thumbnail image
// generation.
type Thumbnailer struct {
	vcs *hardware.VCS

	frameInfo television.FrameInfo

	img     *image.RGBA
	cropImg *image.RGBA

	emulationQuit      chan bool
	emulationCompleted chan bool

	renderChannel chan *image.RGBA
}

// NewThumbnailer is the preferred method of initialisation for the Thumbnailer type.
func NewThumbnailer() (*Thumbnailer, error) {
	thmb := &Thumbnailer{
		emulationQuit:      make(chan bool, 1),
		emulationCompleted: make(chan bool, 1),
		renderChannel:      make(chan *image.RGBA, 1),
	}

	// emulation has completed, by definition, on startup
	thmb.emulationCompleted <- true

	// create a new television. this will be used during the initialisation of
	// the VCS and not referred to directly again
	tv, err := television.NewTelevision("AUTO")
	if err != nil {
		return nil, curated.Errorf("thumbnailer: %v", err)
	}
	tv.AddPixelRenderer(thmb)
	tv.SetFPSCap(true)

	// create a new VCS instance
	thmb.vcs, err = hardware.NewVCS(tv)
	if err != nil {
		return nil, curated.Errorf("thumbnailer: %v", err)
	}

	thmb.img = image.NewRGBA(image.Rect(0, 0, specification.ClksScanline, specification.AbsoluteMaxScanlines))

	// clear pixels. setting the alpha channel so we don't have to later (the
	// alpha channel never changes)
	for y := 0; y < thmb.img.Bounds().Size().Y; y++ {
		for x := 0; x < thmb.img.Bounds().Size().X; x++ {
			thmb.img.SetRGBA(x, y, color.RGBA{0, 0, 0, 255})
		}
	}

	// start with a NTSC television as default
	thmb.Resize(television.NewFrameInfo(specification.SpecNTSC))

	return thmb, nil
}

// EndCreation ends a running emulation that is creating a stream of
// thumbnails. Safe to use even when no emulation is running.
func (thmb *Thumbnailer) EndCreation() {
	select {
	case thmb.emulationQuit <- true:
	default:
	}
}

func (thmb *Thumbnailer) wait() {
	// drain existing emulationQuit channel
	select {
	case <-thmb.emulationQuit:
	default:
	}

	// drain emulationCompleted channel. if there is nothing to drain then send
	// a quit signal and wait for emulation to complete
	select {
	case <-thmb.emulationCompleted:
	default:
		thmb.emulationQuit <- true
		<-thmb.emulationCompleted
	}
}

// UndefinedNumFrames indicates the that the thumbnailing emulation should run
// until it is explicitely stopped with the EndCreation() function (or
// implicitely with a second call to Create())
const UndefinedNumFrames = -1

// CreateFromLoader will cause images to be returned from a running emulation initialised
// with the specified cartridge loader. The emulation will run for a number of
// frames before ending.
//
// It returns the channel over which new frames will be sent.
func (thmb *Thumbnailer) CreateFromLoader(loader cartridgeloader.Loader, numFrames int) chan *image.RGBA {
	thmb.wait()

	go func() {
		defer func() {
			thmb.emulationCompleted <- true
		}()

		err := thmb.vcs.AttachCartridge(loader)
		if err != nil {
			logger.Logf("thumbnailer", err.Error())
			return
		}

		tgtFrame := thmb.vcs.TV.GetCoords().Frame + numFrames

		err = thmb.vcs.Run(func() (emulation.State, error) {
			select {
			case <-thmb.emulationQuit:
				return emulation.Ending, nil
			default:
			}

			if numFrames != UndefinedNumFrames && thmb.vcs.TV.GetCoords().Frame >= tgtFrame {
				return emulation.Ending, nil
			}
			return emulation.Running, nil
		})
		if err != nil {
			logger.Logf("thumbnailer", err.Error())
			return
		}
	}()

	return thmb.renderChannel
}

// CreateFromState will cause images to be returned from a running emulation initialised
// with the specified cartridge loader. The emulation will run for a number of
// frames before ending.
//
// It returns the channel over which new frames will be sent.
func (thmb *Thumbnailer) CreateFromState(state *rewind.State, numFrames int) chan *image.RGBA {
	thmb.wait()

	go func() {
		defer func() {
			thmb.emulationCompleted <- true
		}()

		rewind.Plumb(thmb.vcs, state, true)
		thmb.vcs.TIA.Audio.SetTracker(nil)
		thmb.vcs.RIOT.Ports.AttachEventRecorder(nil)
		thmb.vcs.RIOT.Ports.AttachPlayback(nil)
		thmb.vcs.RIOT.Ports.AttachPlugMonitor(nil)

		tgtFrame := thmb.vcs.TV.GetCoords().Frame + numFrames

		err := thmb.vcs.Run(func() (emulation.State, error) {
			select {
			case <-thmb.emulationQuit:
				return emulation.Ending, nil
			default:
			}

			if numFrames != UndefinedNumFrames && thmb.vcs.TV.GetCoords().Frame >= tgtFrame {
				return emulation.Ending, nil
			}
			return emulation.Running, nil
		})
		if err != nil {
			logger.Logf("thumbnailer", err.Error())
			return
		}
	}()

	return thmb.renderChannel
}

// Resize implements the television.PixelRenderer interface.
func (thmb *Thumbnailer) Resize(frameInfo television.FrameInfo) error {
	thmb.frameInfo = frameInfo
	crop := image.Rect(
		specification.ClksHBlank, frameInfo.VisibleTop,
		specification.ClksHBlank+specification.ClksVisible, frameInfo.VisibleBottom,
	)

	thmb.cropImg = thmb.img.SubImage(crop).(*image.RGBA)

	return nil
}

// NewFrame implements the television.PixelRenderer interface.
func (thmb *Thumbnailer) NewFrame(frameInfo television.FrameInfo) error {
	thmb.frameInfo = frameInfo

	img := *thmb.cropImg
	img.Pix = make([]uint8, len(thmb.cropImg.Pix))
	copy(img.Pix, thmb.cropImg.Pix)

	select {
	case thmb.renderChannel <- &img:
	default:
	}

	return nil
}

// NewScanline implements the television.PixelRenderer interface.
func (thmb *Thumbnailer) NewScanline(scanline int) error {
	return nil
}

// SetPixels implements the television.PixelRenderer interface.
func (thmb *Thumbnailer) SetPixels(sig []signal.SignalAttributes, last int) error {
	var col color.RGBA
	var offset int

	for i := range sig {
		// handle VBLANK by setting pixels to black
		if sig[i]&signal.VBlank == signal.VBlank {
			col = color.RGBA{R: 0, G: 0, B: 0}
		} else {
			px := signal.ColorSignal((sig[i] & signal.Color) >> signal.ColorShift)
			col = thmb.frameInfo.Spec.GetColor(px)
		}

		// small cap improves performance, see https://golang.org/issue/27857
		s := thmb.img.Pix[offset : offset+3 : offset+3]
		s[0] = col.R
		s[1] = col.G
		s[2] = col.B

		offset += 4
	}
	return nil
}

// Reset implements the television.PixelRenderer interface.
func (thmb *Thumbnailer) Reset() {
}

// EndRendering implements the television.PixelRenderer interface.
func (thmb *Thumbnailer) EndRendering() error {
	return nil
}

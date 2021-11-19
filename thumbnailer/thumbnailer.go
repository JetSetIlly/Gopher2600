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

package thumbnailer

import (
	"fmt"
	"image"
	"image/color"
	"strings"
	"sync/atomic"

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

	isEmulating        atomic.Value
	emulationQuit      chan bool
	emulationCompleted chan bool

	Render chan *image.RGBA
}

// NewThumbnailer is the preferred method of initialisation for the Thumbnailer type.
func NewThumbnailer() (*Thumbnailer, error) {
	thmb := &Thumbnailer{
		emulationQuit:      make(chan bool, 1),
		emulationCompleted: make(chan bool, 1),
		Render:             make(chan *image.RGBA, 1),
	}

	// emulation has completed, by definition, on startup
	thmb.emulationCompleted <- true

	// set isEmulating atomic as a boolean
	thmb.isEmulating.Store(false)

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

	// start with a NTSC television as default
	thmb.Resize(television.NewFrameInfo(specification.SpecNTSC))
	thmb.Reset()

	return thmb, nil
}

func (thmb *Thumbnailer) String() string {
	cart := thmb.vcs.Mem.Cart
	s := strings.Builder{}
	s.WriteString(fmt.Sprintf("%s (%s cartridge)", cart.ShortName, cart.ID()))
	if cc := cart.GetContainer(); cc != nil {
		s.WriteString(fmt.Sprintf(" [in %s]", cc.ContainerID()))
	}
	return s.String()
}

// IsEmulating returns true if the thumbnail emulator is working. Useful for
// testing whether the cartridgeloader was an emulatable file.
func (thmb *Thumbnailer) IsEmulating() bool {
	return thmb.isEmulating.Load().(bool)
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
func (thmb *Thumbnailer) CreateFromLoader(loader cartridgeloader.Loader, numFrames int) {
	thmb.wait()

	go func() {
		defer func() {
			thmb.emulationCompleted <- true
			thmb.isEmulating.Store(false)
		}()

		err := thmb.vcs.AttachCartridge(loader)
		if err != nil {
			logger.Logf("thumbnailer", err.Error())
			return
		}

		// if we get to this point then we can be reasonably sure that the
		// cartridgeloader is emulatable
		thmb.isEmulating.Store(true)

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
}

// CreateFromState will cause images to be returned from a running emulation
// initialised with the specified cartridge loader. The emulation will run for
// a number of frames before ending.
//
// See comment for rewind.GetState() about how to use handle rewind.State.
func (thmb *Thumbnailer) CreateFromState(state *rewind.State, numFrames int) {
	thmb.wait()

	go func() {
		defer func() {
			thmb.emulationCompleted <- true
		}()

		rewind.Plumb(thmb.vcs, state, true)

		// the state we've just plumbed into the thumbnailing emulation is from
		// a different emulation which potentially has some links to that
		// emulator still remaining
		thmb.vcs.DetatchEmulationExtras()

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
	if !frameInfo.Stable {
		return nil
	}

	thmb.frameInfo = frameInfo

	img := *thmb.cropImg
	img.Pix = make([]uint8, len(thmb.cropImg.Pix))
	copy(img.Pix, thmb.cropImg.Pix)

	select {
	case thmb.Render <- &img:
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
	// clear pixels. setting the alpha channel so we don't have to later (the
	// alpha channel never changes)
	for y := 0; y < thmb.img.Bounds().Size().Y; y++ {
		for x := 0; x < thmb.img.Bounds().Size().X; x++ {
			thmb.img.SetRGBA(x, y, color.RGBA{0, 0, 0, 255})
		}
	}

	img := *thmb.cropImg
	img.Pix = make([]uint8, len(thmb.cropImg.Pix))
	copy(img.Pix, thmb.cropImg.Pix)

	select {
	case thmb.Render <- &img:
	default:
	}
}

// EndRendering implements the television.PixelRenderer interface.
func (thmb *Thumbnailer) EndRendering() error {
	return nil
}

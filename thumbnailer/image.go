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

	"github.com/jetsetilly/gopher2600/coprocessor"
	"github.com/jetsetilly/gopher2600/debugger/govern"
	"github.com/jetsetilly/gopher2600/environment"
	"github.com/jetsetilly/gopher2600/hardware"
	"github.com/jetsetilly/gopher2600/hardware/preferences"
	"github.com/jetsetilly/gopher2600/hardware/television"
	"github.com/jetsetilly/gopher2600/hardware/television/signal"
	"github.com/jetsetilly/gopher2600/hardware/television/specification"
	"github.com/jetsetilly/gopher2600/logger"
	"github.com/jetsetilly/gopher2600/rewind"
)

// Image type handles the emulation necessary for thumbnail image
// generation.
type Image struct {
	vcs *hardware.VCS

	frameInfo television.FrameInfo

	img     *image.RGBA
	cropImg *image.RGBA

	isEmulating        atomic.Value
	emulationQuit      chan bool
	emulationCompleted chan bool

	Render chan *image.RGBA
}

// NewImage is the preferred method of initialisation for the Image type
func NewImage(prefs *preferences.Preferences) (*Image, error) {
	thmb := &Image{
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
		return nil, fmt.Errorf("thumbnailer: %w", err)
	}
	tv.AddPixelRenderer(thmb)
	tv.SetFPSCap(false)

	// create a new VCS emulation
	thmb.vcs, err = hardware.NewVCS(tv, prefs)
	if err != nil {
		return nil, fmt.Errorf("thumbnailer: %w", err)
	}
	thmb.vcs.Env.Label = environment.Label("thumbnail")
	thmb.img = image.NewRGBA(image.Rect(0, 0, specification.ClksScanline, specification.AbsoluteMaxScanlines))
	thmb.Reset()

	return thmb, nil
}

func (thmb *Image) String() string {
	cart := thmb.vcs.Mem.Cart
	s := strings.Builder{}
	s.WriteString(fmt.Sprintf("%s (%s cartridge)", cart.ShortName, cart.ID()))
	if cc := cart.GetContainer(); cc != nil {
		s.WriteString(fmt.Sprintf(" [in %s]", cc.ContainerID()))
	}
	return s.String()
}

func (thmb *Image) wait() {
	// drain emulationQuit channel
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

// Create will run the thumbnailer emulation for a single frame using the state
// from another emultion as a starting point
//
// The function must be called in the same goroutine as the emulation that
// generated the rewind.State
func (thmb *Image) Create(state *rewind.State) {
	thmb.wait()

	defer func() {
		thmb.emulationCompleted <- true
	}()

	// connect state to our thumbnailer vcs
	rewind.Plumb(thmb.vcs, state, true)

	// the state we've just plumbed into the thumbnailing emulation is from
	// a different emulation which potentially has some links to that
	// emulator still remaining
	thmb.vcs.DetatchEmulationExtras()

	// add yield hook
	thmb.vcs.Mem.Cart.SetYieldHook(thmb)

	// run until target frame has been generated
	tgtFrame := thmb.vcs.TV.GetCoords().Frame + 1

	err := thmb.vcs.Run(func() (govern.State, error) {
		select {
		case <-thmb.emulationQuit:
			return govern.Ending, nil
		default:
		}

		if thmb.vcs.TV.GetCoords().Frame >= tgtFrame {
			return govern.Ending, nil
		}
		return govern.Running, nil
	})

	if err != nil {
		logger.Logf("thumbnailer", err.Error())
		return
	}
}

// CartYield implements the coprocessor.CartYieldHook interface.
func (thmb *Image) CartYield(yield coprocessor.CoProcYieldType) coprocessor.YieldHookResponse {
	if yield.Normal() {
		return coprocessor.YieldHookContinue
	}

	// an unexpected yield type so end the thumbnail emulation
	select {
	case thmb.emulationQuit <- true:
	default:
	}

	// indicate that the mapper should return immediately
	return coprocessor.YieldHookEnd
}

func (thmb *Image) resize(frameInfo television.FrameInfo, force bool) error {
	if thmb.frameInfo.IsDifferent(frameInfo) && (force || frameInfo.Stable) {
		thmb.cropImg = thmb.img.SubImage(frameInfo.Crop()).(*image.RGBA)
	}
	thmb.frameInfo = frameInfo
	return nil
}

// NewFrame implements the television.PixelRenderer interface
func (thmb *Image) NewFrame(frameInfo television.FrameInfo) error {
	thmb.resize(frameInfo, false)

	img := *thmb.cropImg
	img.Pix = make([]uint8, len(thmb.cropImg.Pix))
	copy(img.Pix, thmb.cropImg.Pix)

	select {
	case thmb.Render <- &img:
	default:
	}

	return nil
}

// NewScanline implements the television.PixelRenderer interface
func (thmb *Image) NewScanline(scanline int) error {
	return nil
}

// SetPixels implements the television.PixelRenderer interface
func (thmb *Image) SetPixels(sig []signal.SignalAttributes, last int) error {
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

// Reset implements the television.PixelRenderer interface
func (thmb *Image) Reset() {
	// start with a NTSC television as default
	thmb.resize(television.NewFrameInfo(specification.SpecNTSC), true)

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

// EndRendering implements the television.PixelRenderer interface
func (thmb *Image) EndRendering() error {
	return nil
}

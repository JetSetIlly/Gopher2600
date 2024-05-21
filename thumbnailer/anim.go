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
	"errors"
	"fmt"
	"image"
	"image/color"
	"strings"
	"sync/atomic"

	"github.com/jetsetilly/gopher2600/cartridgeloader"
	"github.com/jetsetilly/gopher2600/debugger/govern"
	"github.com/jetsetilly/gopher2600/environment"
	"github.com/jetsetilly/gopher2600/hardware"
	"github.com/jetsetilly/gopher2600/hardware/preferences"
	"github.com/jetsetilly/gopher2600/hardware/riot/ports"
	"github.com/jetsetilly/gopher2600/hardware/riot/ports/plugging"
	"github.com/jetsetilly/gopher2600/hardware/television"
	"github.com/jetsetilly/gopher2600/hardware/television/signal"
	"github.com/jetsetilly/gopher2600/hardware/television/specification"
	"github.com/jetsetilly/gopher2600/logger"
	"github.com/jetsetilly/gopher2600/notifications"
	"github.com/jetsetilly/gopher2600/preview"
	"github.com/jetsetilly/gopher2600/setup"
)

// Anim type handles the emulation necessary for thumbnail image
// generation.
type Anim struct {
	vcs            *hardware.VCS
	preview        *preview.Emulation
	previewResults *preview.Results
	previewUpdate  chan *preview.Results

	frameInfo television.FrameInfo

	img     *image.RGBA
	cropImg *image.RGBA

	isEmulating        atomic.Value
	emulationQuit      chan bool
	emulationCompleted chan bool

	Render chan *image.RGBA

	// monitorCount is part of the adhoc monitor system. see SetPixels()
	// function for details
	monitorActive     bool
	monitorCount      int
	monitorInput      func()
	monitorInputDelay int
}

var animLabel = environment.Label("thumbnail_anim")

// NewAnim is the preferred method of initialisation for the Anim type
func NewAnim(prefs *preferences.Preferences, spec string) (*Anim, error) {
	thmb := &Anim{
		emulationQuit:      make(chan bool, 1),
		emulationCompleted: make(chan bool, 1),
		Render:             make(chan *image.RGBA, 60),
		previewUpdate:      make(chan *preview.Results, 1),
	}

	// emulation has completed, by definition, on startup
	thmb.emulationCompleted <- true

	// set isEmulating atomic as a boolean
	thmb.isEmulating.Store(false)

	// create a new television. this will be used during the initialisation of
	// the VCS and not referred to directly again
	tv, err := television.NewTelevision(spec)
	if err != nil {
		return nil, fmt.Errorf("thumbnailer: %w", err)
	}
	tv.AddPixelRenderer(thmb)
	tv.SetFPSCap(true)

	// create a new VCS emulation
	thmb.vcs, err = hardware.NewVCS(animLabel, tv, thmb, prefs)
	if err != nil {
		return nil, fmt.Errorf("thumbnailer: %w", err)
	}
	thmb.img = image.NewRGBA(image.Rect(0, 0, specification.ClksScanline, specification.AbsoluteMaxScanlines))
	thmb.Reset()

	// create preview emulation
	thmb.preview, err = preview.NewEmulation(thmb.vcs.Env.Prefs, "AUTO")
	if err != nil {
		return nil, fmt.Errorf("thumbnailer: %w", err)
	}

	return thmb, nil
}

func (thmb *Anim) String() string {
	cart := thmb.vcs.Mem.Cart
	s := strings.Builder{}
	s.WriteString(fmt.Sprintf("%s (%s cartridge)", cart.ShortName, cart.ID()))
	if cc := cart.GetContainer(); cc != nil {
		s.WriteString(fmt.Sprintf(" [in %s]", cc.ContainerID()))
	}
	return s.String()
}

// IsEmulating returns true if the thumbnail emulator is working. Useful for
// testing whether the cartridgeloader was an emulatable file
func (thmb *Anim) IsEmulating() bool {
	return thmb.isEmulating.Load().(bool)
}

// EndCreation ends a running emulation that is creating a stream of
// thumbnails. Safe to use even when no emulation is running
func (thmb *Anim) EndCreation() {
	select {
	case thmb.emulationQuit <- true:
	default:
	}
}

func (thmb *Anim) wait() {
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

	// empty the render queue
	var drained bool
	for !drained {
		select {
		case <-thmb.Render:
		default:
			drained = true
		}
	}
}

// Notify implements the notifications.Notify interface
func (thmb *Anim) Notify(notice notifications.Notice) error {
	switch notice {
	case notifications.NotifySuperchargerFastload:
		// the supercharger ROM will eventually start execution from the PC
		// address given in the supercharger file

		// CPU execution has been interrupted. update state of CPU
		thmb.vcs.CPU.Interrupted = true

		// the interrupted CPU means it never got a chance to
		// finalise the result. we force that here by simply
		// setting the Final flag to true.
		thmb.vcs.CPU.LastResult.Final = true

		// call function to complete tape loading procedure
		fastload := thmb.vcs.Mem.Cart.GetSuperchargerFastLoad()
		err := fastload.Fastload(thmb.vcs.CPU, thmb.vcs.Mem.RAM, thmb.vcs.RIOT.Timer)
		if err != nil {
			return err
		}
	case notifications.NotifySuperchargerSoundloadEnded:
		return thmb.vcs.TV.Reset(true)
	}
	return nil
}

// Create will cause images to be generated by a running emulation initialised
// with the specified cartridge loader. The emulation will run for a number of
// frames before ending
//
// Returns when the preview has completed (so PreviewResults() is safe to call
// once the function has returned)
func (thmb *Anim) Create(cartload cartridgeloader.Loader, spec string, numFrames int) {
	thmb.wait()

	// reset fields
	thmb.previewResults = nil
	thmb.monitorActive = true
	thmb.monitorCount = 0
	thmb.monitorInput = nil
	thmb.monitorInputDelay = 0

	// update TV specification in case it has changed
	thmb.vcs.TV.SetSpec(spec, true)

	// reset function is usually called from the television. we call it here
	// because it's useful for clearing the image and to put the now empty
	// image in the render queue at the start of the animation
	thmb.Reset()

	go func() {
		defer func() {
			thmb.emulationCompleted <- true
			thmb.isEmulating.Store(false)
		}()

		// attach cartridge using the setup system
		err := setup.AttachCartridge(thmb.vcs, cartload, true)
		if err != nil {
			logger.Logf(logger.Allow, "thumbnailer", err.Error())
			return
		}

		// run preview for just one frame. this is enough to give us basic
		// information like the cartridge mapper and detected controllers
		_ = thmb.preview.RunN(cartload, 1)

		// indicate that the first part of the preview has completed and that
		// the preview results should be updated
		select {
		case thmb.previewUpdate <- thmb.preview.Results():
		default:
		}

		// run preview some more in order to get excellent frame information
		err = thmb.preview.Run(cartload)
		if err == nil || errors.Is(err, cartridgeloader.NoFilename) {
			thmb.vcs.TV.SetResizer(thmb.preview.Results().Resizer, thmb.preview.Results().FrameNum)
		}

		// indicate that the second part of the preview has completed
		select {
		case thmb.previewUpdate <- thmb.preview.Results():
		default:
		}

		// if we get to this point then we can be reasonably sure that the
		// cartridgeloader is emulatable
		thmb.isEmulating.Store(true)

		// run until target frame has been generated
		tgtFrame := thmb.vcs.TV.GetCoords().Frame + numFrames

		err = thmb.vcs.Run(func() (govern.State, error) {
			select {
			case <-thmb.emulationQuit:
				return govern.Ending, nil
			default:
			}

			if numFrames != UndefinedNumFrames && thmb.vcs.TV.GetCoords().Frame >= tgtFrame {
				return govern.Ending, nil
			}
			return govern.Running, nil
		})
		if err != nil {
			logger.Logf(logger.Allow, "thumbnailer", err.Error())
			return
		}
	}()
}

func (thmb *Anim) resize(frameInfo television.FrameInfo) {
	if thmb.frameInfo.IsDifferent(frameInfo) {
		thmb.cropImg = thmb.img.SubImage(frameInfo.Crop()).(*image.RGBA)
	}
	thmb.frameInfo = frameInfo
}

// NewFrame implements the television.PixelRenderer interface
func (thmb *Anim) NewFrame(frameInfo television.FrameInfo) error {
	// act on monitor input
	if thmb.monitorActive && thmb.monitorInputDelay > 0 {
		thmb.monitorInputDelay--
		if thmb.monitorInputDelay == 0 {
			thmb.monitorInput()
		}
	}

	thmb.resize(frameInfo)

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
func (thmb *Anim) NewScanline(scanline int) error {
	return nil
}

// SetPixels implements the television.PixelRenderer interface
func (thmb *Anim) SetPixels(sig []signal.SignalAttributes, last int) error {
	var col color.RGBA

	// this adhoc "monitor" system looks for changes in pixels and uses that
	// information to insert input into the emulation
	var monitorChanges bool
	var monitorPixels int
	var monitorBoringPixels int
	var monitorPrev color.RGBA

	var offset int
	for i := range sig {
		// note vblank signal for later
		vblank := sig[i]&signal.VBlank == signal.VBlank

		// handle VBLANK by setting pixels to black
		if vblank {
			col = color.RGBA{R: 0, G: 0, B: 0}
		} else {
			px := signal.ColorSignal((sig[i] & signal.Color) >> signal.ColorShift)
			col = thmb.frameInfo.Spec.GetColor(px)
		}

		// small cap improves performance, see https://golang.org/issue/27857
		s := thmb.img.Pix[offset : offset+3 : offset+3]
		offset += 4

		// monitor pixels
		if thmb.monitorActive && !vblank && i%specification.ClksScanline > specification.ClksHBlank {
			monitorPixels++
			if s[0] != col.R || s[1] != col.G || s[2] != col.B {
				monitorChanges = true
			}
			if col == monitorPrev {
				monitorBoringPixels++
			}
		}
		monitorPrev = col

		// set new color
		s[0] = col.R
		s[1] = col.G
		s[2] = col.B
	}

	const (
		monitorBoringThreshold = 0.98
		monitorCountThreshold  = 60
	)

	// act on monitor information
	if thmb.monitorActive && !monitorChanges &&
		float32(monitorBoringPixels)/float32(monitorPixels) > monitorBoringThreshold {

		// increase monitor count and once threshold has been reached insert the input
		thmb.monitorCount++
		if thmb.monitorCount >= monitorCountThreshold {
			thmb.monitorCount = 0

			//  monitorInput works best as a chain. the input function is run
			//  after delay number of frames. the function issues the input and
			//  sets the delay and function to indicate the next input
			thmb.monitorInputDelay = 1
			thmb.monitorInput = func() {
				thmb.vcs.RIOT.Ports.HandleInputEvent(
					ports.InputEvent{Port: plugging.PortPanel,
						Ev: ports.PanelReset,
						D:  true})

				// add the next link in the input chain
				thmb.monitorInputDelay = 1
				thmb.monitorInput = func() {
					thmb.vcs.RIOT.Ports.HandleInputEvent(
						ports.InputEvent{Port: plugging.PortPanel,
							Ev: ports.PanelReset,
							D:  false})

					// disable monitor at end of input chain
					thmb.monitorActive = false
				}
			}
		}
	} else {
		thmb.monitorCount = 0
	}

	return nil
}

// Reset implements the television.PixelRenderer interface
func (thmb *Anim) Reset() {
	// start with a NTSC television as default
	thmb.resize(television.NewFrameInfo(specification.SpecNTSC))

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
func (thmb *Anim) EndRendering() error {
	return nil
}

// PreviewResults returns the results of the preview emulation
func (thmb *Anim) PreviewResults() *preview.Results {
	select {
	case thmb.previewResults = <-thmb.previewUpdate:
	default:
	}
	return thmb.previewResults
}

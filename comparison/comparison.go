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
	"fmt"
	"image"
	"image/color"
	"strings"
	"sync/atomic"

	"github.com/jetsetilly/gopher2600/cartridgeloader"
	"github.com/jetsetilly/gopher2600/debugger/govern"
	"github.com/jetsetilly/gopher2600/environment"
	"github.com/jetsetilly/gopher2600/hardware"
	"github.com/jetsetilly/gopher2600/hardware/riot/ports"
	"github.com/jetsetilly/gopher2600/hardware/television"
	"github.com/jetsetilly/gopher2600/hardware/television/frameinfo"
	"github.com/jetsetilly/gopher2600/hardware/television/signal"
	"github.com/jetsetilly/gopher2600/hardware/television/specification"
	"github.com/jetsetilly/gopher2600/notifications"
)

// Comparison type runs a parallel emulation with the intention of comparing
// the output with the driver emulation.
type Comparison struct {
	VCS *hardware.VCS

	frameInfo frameinfo.Current

	img         *image.RGBA
	cropImg     *image.RGBA
	diffImg     *image.RGBA
	cropDiffImg *image.RGBA

	isEmulating   atomic.Value
	emulationQuit chan bool

	Render     chan *image.RGBA
	DiffRender chan *image.RGBA
	AudioDiff  chan bool

	audio []uint8

	// pixel renderer implementation for the "driver" emulation. ie. the
	// emulation we'll be comparing against
	driver driver
}

const comparisonLabel = environment.Label("comparison")

// NewComparison is the preferred method of initialisation for the Comparison type.
func NewComparison(driverVCS *hardware.VCS) (*Comparison, error) {
	cmp := &Comparison{
		emulationQuit: make(chan bool, 1),
		Render:        make(chan *image.RGBA, 1),
		DiffRender:    make(chan *image.RGBA, 1),
		AudioDiff:     make(chan bool, 1),
	}

	// set isEmulating atomic as a boolean
	cmp.isEmulating.Store(false)

	// create a new television. this will be used during the initialisation of
	// the VCS and not referred to directly again
	tv, err := television.NewTelevision(driverVCS.TV.GetResetSpecID())
	if err != nil {
		return nil, fmt.Errorf("comparison: %w", err)
	}
	tv.AddPixelRenderer(cmp)
	tv.AddAudioMixer(cmp)
	tv.SetFPSCap(false)

	// create a new VCS emulation
	cmp.VCS, err = hardware.NewVCS(comparisonLabel, tv, cmp, driverVCS.Env.Prefs)
	if err != nil {
		return nil, fmt.Errorf("comparison: %w", err)
	}

	cmp.img = image.NewRGBA(image.Rect(0, 0, specification.ClksScanline, specification.AbsoluteMaxScanlines))
	cmp.diffImg = image.NewRGBA(image.Rect(0, 0, specification.ClksScanline, specification.AbsoluteMaxScanlines))

	// start with a NTSC television as default
	cmp.Resize(frameinfo.NewCurrent(specification.SpecNTSC))
	cmp.Reset()

	// create driver
	cmp.driver = newDriver(driverVCS.TV)
	driverVCS.TV.AddPixelRenderer(&cmp.driver)
	driverVCS.TV.AddAudioMixer(&cmp.driver)

	// synchronise RIOT ports
	sync := make(chan ports.TimedInputEvent, 32)
	err = cmp.VCS.Input.AttachPassenger(sync)
	if err != nil {
		return nil, fmt.Errorf("comparison: %w", err)
	}
	err = driverVCS.Input.AttachDriver(sync)
	if err != nil {
		return nil, fmt.Errorf("comparison: %w", err)
	}

	return cmp, nil
}

func (cmp *Comparison) String() string {
	cart := cmp.VCS.Mem.Cart
	s := strings.Builder{}
	s.WriteString(fmt.Sprintf("%s (%s cartridge)", cart.ShortName, cart.ID()))
	if cc := cart.GetContainer(); cc != nil {
		s.WriteString(fmt.Sprintf(" [in %s]", cc.ContainerID()))
	}
	return s.String()
}

// IsEmulating returns true if the comparison emulator is working. Useful for
// testing whether the cartridgeloader was an emulatable file.
func (cmp *Comparison) IsEmulating() bool {
	return cmp.isEmulating.Load().(bool)
}

// Quit ends the running comparison emulation.
func (cmp *Comparison) Quit() {
	// very important that we remove the pixel renderer from the main
	// emulation's television that we added as part of the newDriver()
	// function. if we don't then the renderer will continue firing and get
	// jammed waiting on a channel that has been abandone
	cmp.driver.tv.RemovePixelRenderer(&cmp.driver)

	// send quit signal to the comparison emulation
	cmp.emulationQuit <- true
}

// PushNotify implements the notifications.Notify interface
func (cmp *Comparison) PushNotify(notice notifications.Notice, data ...string) error {
	return nil
}

// Notify implements the notifications.Notify interface
func (cmp *Comparison) Notify(notice notifications.Notice, data ...string) error {
	switch notice {
	case notifications.NotifySuperchargerFastload:
		// the supercharger ROM will eventually start execution from the PC
		// address given in the supercharger file

		// the interrupted CPU means it never got a chance to
		// finalise the result. we force that here by simply
		// setting the Final flag to true.
		cmp.VCS.CPU.LastResult.Final = true

		// call function to complete tape loading procedure
		fastload := cmp.VCS.Mem.Cart.GetSuperchargerFastLoad()
		err := fastload.Fastload(cmp.VCS.CPU, cmp.VCS.Mem.RAM, cmp.VCS.RIOT.Timer)
		if err != nil {
			return err
		}
	case notifications.NotifySuperchargerSoundloadEnded:
		return cmp.VCS.TV.Reset(true)
	}

	return nil
}

// CreateFromLoader will cause images to be generated from a running emulation
// initialised with the specified cartridge loader.
func (cmp *Comparison) CreateFromLoader(cartload cartridgeloader.Loader) error {
	if cmp.IsEmulating() {
		return fmt.Errorf("comparison: emulation already running")
	}

	go func() {
		defer func() {
			cmp.driver.quit <- nil
			cmp.isEmulating.Store(false)
			cartload.Close()
		}()

		// not using setup system to attach cartridge. maybe we should?
		err := cmp.VCS.AttachCartridge(cartload)
		if err != nil {
			cmp.driver.quit <- err
			return
		}

		// if we get to this point then we can be reasonably sure that the
		// cartridgeloader is emulatable
		cmp.isEmulating.Store(true)

		// we need the main emulation to be exactly one frame ahead of the
		// comparison emulation. the best way of doing this is to not start the
		// comparison emulation immediately
		//
		// the reason for the one frame ahead rule is so we can feed the input
		// from the driver emulation to the comparison emulation

		select {
		case <-cmp.driver.sync:
		case <-cmp.emulationQuit:
			return
		}

		select {
		case cmp.driver.ack <- true:
		case <-cmp.emulationQuit:
			return
		}

		err = cmp.VCS.Run(func() (govern.State, error) {
			select {
			case <-cmp.emulationQuit:
				return govern.Ending, nil
			default:
			}

			return govern.Running, nil
		})
		if err != nil {
			cmp.driver.quit <- err
			return
		}
	}()

	return nil
}

// Resize implements the television.PixelRenderer interface.
func (cmp *Comparison) Resize(frameInfo frameinfo.Current) error {
	cmp.frameInfo = frameInfo
	crop := cmp.frameInfo.Crop()
	cmp.cropImg = cmp.img.SubImage(crop).(*image.RGBA)
	cmp.cropDiffImg = cmp.diffImg.SubImage(crop).(*image.RGBA)

	return nil
}

// NewFrame implements the television.PixelRenderer interface.
func (cmp *Comparison) NewFrame(frameInfo frameinfo.Current) error {
	cmp.frameInfo = frameInfo

	img := *cmp.cropImg
	img.Pix = make([]uint8, len(cmp.cropImg.Pix))
	copy(img.Pix, cmp.cropImg.Pix)

	select {
	case cmp.Render <- &img:
	default:
	}

	select {
	case <-cmp.driver.sync:
	case <-cmp.emulationQuit:
		return nil
	}

	select {
	case cmp.driver.ack <- true:
	case <-cmp.emulationQuit:
		return nil
	}

	if !cmp.frameInfo.Stable || !cmp.driver.frameInfo.Stable {
		return nil
	}

	// comparison of frame numbers takes into account that the driver emulation
	// is one ahead of the comparison emulation (for user-input purposes)
	if cmp.frameInfo.FrameNum > cmp.driver.frameInfo.FrameNum-1 {
		return fmt.Errorf("comparison: comparison emulation is running AHEAD of the driver emulation")
	}
	if cmp.frameInfo.FrameNum < cmp.driver.frameInfo.FrameNum-1 {
		return fmt.Errorf("comparison: comparison emulation is running BEHIND of the driver emulation")
	}

	var drvImg *image.RGBA
	if cmp.driver.swapIdx {
		drvImg = cmp.driver.cropImg[0]
	} else {
		drvImg = cmp.driver.cropImg[1]
	}

	if len(cmp.cropImg.Pix) != len(drvImg.Pix) {
		return fmt.Errorf("comparison: frames are different sizes")
	}

	for i := 0; i < len(cmp.cropImg.Pix); i += 4 {
		a := cmp.cropImg.Pix[i : i+3 : i+3]
		b := drvImg.Pix[i : i+3 : i+3]
		c := cmp.cropDiffImg.Pix[i : i+4 : i+4]
		if a[0] != b[0] || a[1] != b[1] || a[2] != b[2] {
			c[0] = 0xff
			c[1] = 0xff
			c[2] = 0xff
		} else {
			c[0] = 0x00
			c[1] = 0x00
			c[2] = 0x00
		}
		c[3] = 0xff
	}

	// indicate visual differences
	cmp.DiffRender <- cmp.cropDiffImg

	// find differences in the two audio buffers
	var audioIsDifferent bool

	var drvAudioIdx int
	if cmp.driver.swapIdx {
		drvAudioIdx = 0
	} else {
		drvAudioIdx = 1
	}

	if len(cmp.audio) != len(cmp.driver.audio[drvAudioIdx]) {
		audioIsDifferent = true
	} else {
		for i := range cmp.driver.audio[drvAudioIdx] {
			if cmp.audio[i] != cmp.driver.audio[drvAudioIdx][i] {
				audioIsDifferent = true
				break
			}
		}
	}

	// indicate audio differences
	cmp.AudioDiff <- audioIsDifferent

	// clear audio buffers
	cmp.audio = cmp.audio[:0]
	cmp.driver.audio[drvAudioIdx] = cmp.driver.audio[drvAudioIdx][:0]

	return nil
}

// NewScanline implements the television.PixelRenderer interface.
func (cmp *Comparison) NewScanline(scanline int) error {
	return nil
}

// SetPixels implements the television.PixelRenderer interface.
func (cmp *Comparison) SetPixels(sig []signal.SignalAttributes, last int) error {
	var col color.RGBA
	var offset int

	for i := range sig {
		// handle VBLANK by setting pixels to black. we also manually handle
		// NoSignal in the same way
		if sig[i].VBlank || sig[i].Index == signal.NoSignal {
			col = cmp.frameInfo.Spec.GetColor(signal.ZeroBlack)
		} else {
			col = cmp.frameInfo.Spec.GetColor(sig[i].Color)
		}

		// small cap improves performance, see https://golang.org/issue/27857
		s := cmp.img.Pix[offset : offset+3 : offset+3]
		s[0] = col.R
		s[1] = col.G
		s[2] = col.B

		offset += 4
	}
	return nil
}

// Reset implements the television.PixelRenderer interface.
func (cmp *Comparison) Reset() {
	// clear pixels. setting the alpha channel so we don't have to later (the
	// alpha channel never changes)
	for y := 0; y < cmp.img.Bounds().Size().Y; y++ {
		for x := 0; x < cmp.img.Bounds().Size().X; x++ {
			cmp.img.SetRGBA(x, y, color.RGBA{0, 0, 0, 255})
		}
	}

	for y := 0; y < cmp.diffImg.Bounds().Size().Y; y++ {
		for x := 0; x < cmp.diffImg.Bounds().Size().X; x++ {
			cmp.diffImg.SetRGBA(x, y, color.RGBA{0, 0, 0, 255})
		}
	}

	img := *cmp.cropImg
	img.Pix = make([]uint8, len(cmp.cropImg.Pix))
	copy(img.Pix, cmp.cropImg.Pix)

	select {
	case cmp.Render <- &img:
	default:
	}
}

// EndRendering implements the television.PixelRenderer interface.
func (cmp *Comparison) EndRendering() error {

	return nil
}

// SetAudio implements the television.AudioMixer interface.
func (cmp *Comparison) SetAudio(sig []signal.AudioSignalAttributes) error {
	for _, s := range sig {
		cmp.audio = append(cmp.audio, s.AudioChannel0, s.AudioChannel1)
	}
	return nil
}

// EndMixing implements the television.AudioMixer interface.
func (cmp *Comparison) EndMixing() error {
	return nil
}

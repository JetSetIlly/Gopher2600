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

package headless

import (
	"fmt"
	"image"
	"image/color"

	"github.com/jetsetilly/gopher2600/debugger"
	"github.com/jetsetilly/gopher2600/gui"
	"github.com/jetsetilly/gopher2600/gui/screenshot"
	"github.com/jetsetilly/gopher2600/hardware/television/frameinfo"
	"github.com/jetsetilly/gopher2600/hardware/television/signal"
	"github.com/jetsetilly/gopher2600/hardware/television/specification"
)

// Headless implements what the GUI for Gopher2600's HEADLESS mode. This only exists so that
// screenshotting can be supported from HEADLESS mode.
type Headless struct {
	dbg       *debugger.Debugger
	frameInfo frameinfo.Current
	sig       []signal.SignalAttributes
	prevSig   []signal.SignalAttributes
	pixels    *image.RGBA
}

func NewHeadless(dbg *debugger.Debugger) *Headless {
	hdls := &Headless{
		dbg: dbg,
	}
	hdls.pixels = image.NewRGBA(image.Rect(0, 0, specification.ClksScanline, specification.AbsoluteMaxScanlines))
	for i := 0; i < len(hdls.pixels.Pix); i += 4 {
		hdls.pixels.Pix[i+3] = 255
	}
	dbg.TV().AddPixelRenderer(hdls)
	return hdls
}

// SetFeature implements the GUI interface.
func (hdls *Headless) SetFeature(request gui.FeatureReq, args ...gui.FeatureReqData) error {
	if request == gui.ReqScreenshot {
		switch len(args) {
		case 0:
			hdls.screenshot("")
		case 1:
			hdls.screenshot(args[0].(string))
		default:
			return fmt.Errorf("wrong number of arguments for %s", request)
		}
	}
	return nil
}

func (hdls *Headless) screenshot(filename string) {
	// clear image first (keeping alpha channel unchanged)
	for i := 0; i < len(hdls.pixels.Pix); i += 4 {
		s := hdls.pixels.Pix[i : i+3 : i+3]
		s[0] = 0
		s[1] = 0
		s[2] = 0
	}

	// decide which set of signals to use. the current set or if there are less than one scanlines
	// worth of signals, use the previous set
	sig := hdls.sig
	if len(sig) <= specification.ClksScanline {
		sig = hdls.prevSig
	}

	var col color.RGBA
	var offset int

	for i := range sig {
		// handle VBLANK by setting pixels to black. we also manually handle
		// NoSignal in the same way
		if sig[i].VBlank || sig[i].Index == signal.NoSignal {
			col = hdls.frameInfo.Spec.GetColor(signal.ZeroBlack)
		} else {
			col = hdls.frameInfo.Spec.GetColorScreen(sig, i, specification.ClksScanline)
		}

		// small cap improves performance, see https://golang.org/issue/27857
		// alpha channel never changes
		s := hdls.pixels.Pix[offset : offset+3 : offset+3]
		s[0] = col.R
		s[1] = col.G
		s[2] = col.B

		offset += 4
	}

	crop := hdls.frameInfo.Crop()
	cropped := hdls.pixels.SubImage(crop).(*image.RGBA)
	scaled := screenshot.ScaleRawPixels(cropped)

	// save image to file as a JPEG
	if filename == "" {
		filename = screenshot.GenerateFilename(hdls.dbg.VCS().Mem.Cart.ShortName, "", "headless")
	}
	screenshot.Save(scaled, filename)
}

// NewFrame implements the television.PixelRenderer interface.
func (hdls *Headless) NewFrame(frameinfo frameinfo.Current) error {
	hdls.frameInfo = frameinfo
	hdls.prevSig = hdls.sig
	return nil
}

// NewScanline implements the television.PixelRenderer interface.
func (hdls *Headless) NewScanline(scanline int) error {
	return nil
}

// SetPixels implements the television.PixelRenderer interface.
func (hdls *Headless) SetPixels(sig []signal.SignalAttributes, last int) error {
	hdls.sig = sig[:last]
	return nil
}

// Reset implements the television.PixelRenderer interface.
func (hdls *Headless) Reset() {
}

// EndRendering implements the television.PixelRenderer interface.
func (hdls *Headless) EndRendering() error {
	return nil
}

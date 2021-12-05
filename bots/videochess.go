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

package bot

import (
	"crypto/sha1"
	"fmt"
	"image"
	"image/color"
	"image/draw"

	"github.com/jetsetilly/gopher2600/bots/uci"
	"github.com/jetsetilly/gopher2600/curated"
	"github.com/jetsetilly/gopher2600/hardware/riot/ports"
	"github.com/jetsetilly/gopher2600/hardware/riot/ports/plugging"
	"github.com/jetsetilly/gopher2600/hardware/television"
	"github.com/jetsetilly/gopher2600/hardware/television/coords"
	"github.com/jetsetilly/gopher2600/hardware/television/signal"
	"github.com/jetsetilly/gopher2600/hardware/television/specification"
)

const (
	boardMarginLeft = 44
	boardMarginTop  = 70
	squareWidth     = 8
	squareHeight    = 17
)

var startingImage = [sha1.Size]byte{206, 150, 33, 111, 96, 82, 105, 37, 18, 218, 111, 99, 43, 70, 231, 84, 152, 24, 218, 150}

var cursorSampleDataEvenColumns = [...]uint8{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 172, 120, 60, 255, 172, 120, 60, 255, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 172, 120, 60, 255, 172, 120, 60, 255, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 172, 120, 60, 255, 172, 120, 60, 255, 0, 0, 0, 0, 172, 120, 60, 255, 172, 120, 60, 255, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 172, 120, 60, 255, 172, 120, 60, 255, 172, 120, 60, 255, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 172, 120, 60, 255, 172, 120, 60, 255, 0, 0, 0, 0, 172, 120, 60, 255, 172, 120, 60, 255, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 172, 120, 60, 255, 172, 120, 60, 255, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 172, 120, 60, 255, 172, 120, 60, 255, 0, 0, 0, 0}

var cursorSampleDataOddColumns = [...]uint8{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 172, 120, 60, 255, 172, 120, 60, 255, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 172, 120, 60, 255, 172, 120, 60, 255, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 172, 120, 60, 255, 172, 120, 60, 255, 0, 0, 0, 0, 172, 120, 60, 255, 172, 120, 60, 255, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 172, 120, 60, 255, 172, 120, 60, 255, 172, 120, 60, 255, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 172, 120, 60, 255, 172, 120, 60, 255, 0, 0, 0, 0, 172, 120, 60, 255, 172, 120, 60, 255, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 172, 120, 60, 255, 172, 120, 60, 255, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 172, 120, 60, 255, 172, 120, 60, 255, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}

var boardIndicator = [...]uint8{28, 32, 156, 255, 28, 32, 156, 255, 28, 32, 156, 255, 28, 32, 156, 255, 28, 32, 156, 255, 28, 32, 156, 255, 28, 32, 156, 255, 28, 32, 156, 255, 28, 32, 156, 255, 28, 32, 156, 255, 28, 32, 156, 255, 28, 32, 156, 255, 28, 32, 156, 255, 28, 32, 156, 255, 28, 32, 156, 255, 28, 32, 156, 255, 28, 32, 156, 255, 28, 32, 156, 255, 28, 32, 156, 255, 28, 32, 156, 255, 28, 32, 156, 255, 28, 32, 156, 255, 28, 32, 156, 255, 28, 32, 156, 255, 28, 32, 156, 255, 28, 32, 156, 255, 28, 32, 156, 255, 28, 32, 156, 255, 28, 32, 156, 255, 28, 32, 156, 255, 28, 32, 156, 255, 28, 32, 156, 255, 28, 32, 156, 255, 28, 32, 156, 255, 28, 32, 156, 255, 28, 32, 156, 255, 28, 32, 156, 255, 28, 32, 156, 255, 28, 32, 156, 255, 28, 32, 156, 255, 28, 32, 156, 255, 28, 32, 156, 255, 28, 32, 156, 255, 28, 32, 156, 255, 28, 32, 156, 255, 28, 32, 156, 255, 28, 32, 156, 255, 28, 32, 156, 255, 28, 32, 156, 255, 28, 32, 156, 255, 28, 32, 156, 255, 28, 32, 156, 255, 28, 32, 156, 255, 28, 32, 156, 255, 28, 32, 156, 255, 28, 32, 156, 255, 28, 32, 156, 255, 28, 32, 156, 255, 28, 32, 156, 255, 28, 32, 156, 255, 28, 32, 156, 255, 28, 32, 156, 255, 28, 32, 156, 255, 28, 32, 156, 255, 28, 32, 156, 255, 28, 32, 156, 255, 164, 184, 252, 255, 164, 184, 252, 255, 28, 32, 156, 255, 28, 32, 156, 255, 28, 32, 156, 255, 28, 32, 156, 255, 28, 32, 156, 255, 28, 32, 156, 255, 28, 32, 156, 255, 28, 32, 156, 255, 28, 32, 156, 255, 28, 32, 156, 255, 28, 32, 156, 255, 28, 32, 156, 255, 28, 32, 156, 255, 28, 32, 156, 255, 164, 184, 252, 255, 164, 184, 252, 255, 28, 32, 156, 255, 28, 32, 156, 255, 28, 32, 156, 255, 28, 32, 156, 255, 28, 32, 156, 255, 28, 32, 156, 255, 28, 32, 156, 255, 28, 32, 156, 255, 28, 32, 156, 255, 28, 32, 156, 255, 28, 32, 156, 255, 28, 32, 156, 255, 28, 32, 156, 255, 28, 32, 156, 255, 164, 184, 252, 255, 164, 184, 252, 255, 28, 32, 156, 255, 28, 32, 156, 255, 28, 32, 156, 255, 28, 32, 156, 255, 28, 32, 156, 255, 28, 32, 156, 255, 28, 32, 156, 255, 28, 32, 156, 255, 28, 32, 156, 255, 28, 32, 156, 255, 28, 32, 156, 255, 28, 32, 156, 255, 28, 32, 156, 255, 28, 32, 156, 255, 164, 184, 252, 255, 164, 184, 252, 255, 28, 32, 156, 255, 28, 32, 156, 255, 28, 32, 156, 255, 28, 32, 156, 255, 28, 32, 156, 255, 28, 32, 156, 255, 28, 32, 156, 255, 28, 32, 156, 255, 28, 32, 156, 255, 28, 32, 156, 255, 28, 32, 156, 255, 28, 32, 156, 255, 28, 32, 156, 255, 28, 32, 156, 255, 164, 184, 252, 255, 164, 184, 252, 255, 28, 32, 156, 255, 28, 32, 156, 255, 28, 32, 156, 255, 28, 32, 156, 255}

type observer struct {
	frameInfo television.FrameInfo
	img       *image.RGBA
	analysis  chan *image.RGBA

	audioFeedback chan bool
}

func newObserver() *observer {
	obs := &observer{
		img:           image.NewRGBA(image.Rect(0, 0, specification.ClksScanline, specification.AbsoluteMaxScanlines)),
		analysis:      make(chan *image.RGBA, 1),
		audioFeedback: make(chan bool, 1),
	}

	obs.Resize(television.NewFrameInfo(specification.SpecNTSC))
	obs.Reset()

	return obs
}

const audioSilence = 15

func (o *observer) SetAudio(sig []signal.SignalAttributes) error {
	for _, s := range sig {
		if uint8((s&signal.AudioChannel0)>>signal.AudioChannel0Shift) != audioSilence {
			select {
			case o.audioFeedback <- true:
			default:
			}
		}
	}
	return nil
}

func (o *observer) EndMixing() error {
	return nil
}

func (obs *observer) Resize(frameInfo television.FrameInfo) error {
	obs.frameInfo = frameInfo
	return nil
}

func (obs *observer) NewFrame(frameInfo television.FrameInfo) error {
	obs.frameInfo = frameInfo

	img := *obs.img
	img.Pix = make([]uint8, len(obs.img.Pix))
	copy(img.Pix, obs.img.Pix)

	select {
	case obs.analysis <- &img:
	default:
	}

	return nil
}

func (obs *observer) NewScanline(scanline int) error {
	return nil
}

func (obs *observer) SetPixels(sig []signal.SignalAttributes, last int) error {
	var col color.RGBA
	var offset int

	for i := range sig {
		// handle VBLANK by setting pixels to black
		if sig[i]&signal.VBlank == signal.VBlank {
			col = color.RGBA{R: 0, G: 0, B: 0}
		} else {
			px := signal.ColorSignal((sig[i] & signal.Color) >> signal.ColorShift)
			col = obs.frameInfo.Spec.GetColor(px)
		}

		// small cap improves performance, see https://golang.org/issue/27857
		s := obs.img.Pix[offset : offset+3 : offset+3]
		s[0] = col.R
		s[1] = col.G
		s[2] = col.B

		offset += 4
	}
	return nil
}

func (obs *observer) Reset() {
	// clear pixels. setting the alpha channel so we don't have to later (the
	// alpha channel never changes)
	for y := 0; y < obs.img.Bounds().Size().Y; y++ {
		for x := 0; x < obs.img.Bounds().Size().X; x++ {
			obs.img.SetRGBA(x, y, color.RGBA{0, 0, 0, 255})
		}
	}
}

func (obs *observer) EndRendering() error {
	return nil
}

// TV defines the functions required by a bot.
type TV interface {
	AddPixelRenderer(television.PixelRenderer)
	AddAudioMixer(television.AudioMixer)
	GetCoords() coords.TelevisionCoords
}

// VCS defines the functions required by a bot.
type VCS interface {
	QueueEvent(ports.InputEvent) error
}

type videoChessBot struct {
	obs *observer

	vcs VCS
	tv  TV

	// the most recent position
	currentPosition *image.RGBA

	// a copy of the board at the end of the previous move by white
	prevPosition *image.RGBA

	// the square last looked at in detail
	inspectionSquare *image.RGBA
	cmpSquare        *image.RGBA

	moveFrom image.Rectangle
	moveTo   image.Rectangle

	debuggingRender chan *image.RGBA
}

// composites a simple debugging image to be sent over the debuggingRender channel
func (bot *videoChessBot) commitDebuggingRender() {
	img := *bot.currentPosition
	img.Pix = make([]uint8, len(bot.currentPosition.Pix))
	copy(img.Pix, bot.currentPosition.Pix)

	col := color.RGBA{200, 50, 50, 100}
	draw.Draw(&img, bot.moveFrom, &image.Uniform{col}, image.Point{}, draw.Over)
	col = color.RGBA{50, 200, 50, 100}
	draw.Draw(&img, bot.moveTo, &image.Uniform{col}, image.Point{}, draw.Over)

	draw.Draw(&img, bot.inspectionSquare.Bounds().Bounds().Add(image.Point{X: 10, Y: 10}), bot.inspectionSquare, image.Point{}, draw.Src)

	select {
	case bot.debuggingRender <- &img:
	default:
	}
}

// returns the image.Rectangle for a square on the chess board
func (bot *videoChessBot) getRect(col int, row int) image.Rectangle {
	minX := specification.ClksHBlank + boardMarginLeft + (col * squareWidth)
	minY := boardMarginTop + (row * squareHeight)
	maxX := minX + squareWidth
	maxY := minY + squareHeight
	return image.Rect(minX, minY, maxX, maxY)
}

// checks that board is present - board disappears during "thinking" so it's
// important we only continue once the board reappears
//
// returns true if board is visible
func (bot *videoChessBot) lookForBoard() bool {
	clip := bot.getRect(2, -1)
	draw.Draw(bot.inspectionSquare, bot.inspectionSquare.Bounds(), bot.currentPosition.SubImage(clip), clip.Min, draw.Src)
	for i := range boardIndicator {
		if boardIndicator[i] != bot.inspectionSquare.Pix[i] {
			return false
		}
	}
	return true
}

// compares currentPosition with prevPosition for a change. returns fromCol,
// fromRow and toCol, toRow - indicating the move
//
// rows are flipped to normal chess orientation. rows are counted from the
// bottom (white side) in chess.
func (bot *videoChessBot) lookForVCSMove() (int, int, int, int) {
	fromCol := -1
	fromRow := -1
	toCol := -1
	toRow := -1

	for row := 0; row < 8; row++ {
		for col := 0; col < 8; col++ {
			b := bot.isCursor(bot.prevPosition, col, row)
			a := bot.isCursor(bot.currentPosition, col, row)
			if a && !b {
				fromCol = col
				fromRow = row

				// end for loops
				col = 8
				row = 8
			}
		}
	}

	if fromCol == -1 && fromRow == -1 {
		return -1, -1, -1, -1
	}

	for row := 0; row < 8; row++ {
		for col := 0; col < 8; col++ {
			if !bot.isCursor(bot.currentPosition, col, row) {
				if !bot.isCursor(bot.prevPosition, col, row) {
					if !bot.cmpPositions(col, row) {
						toCol = col
						toRow = row

						// end for loops
						col = 8
						row = 8
					}
				}
			}
		}
	}

	return fromCol, 8 - fromRow, toCol, 8 - toRow
}

// returns false if equivalent squares in the currentPosition and prevPosition images are different.
func (bot *videoChessBot) cmpPositions(col int, row int) bool {
	clip := bot.getRect(col, row)

	draw.Draw(bot.inspectionSquare, bot.inspectionSquare.Bounds(), bot.currentPosition.SubImage(clip), clip.Min, draw.Src)
	draw.Draw(bot.cmpSquare, bot.inspectionSquare.Bounds(), bot.prevPosition.SubImage(clip), clip.Min, draw.Src)

	for i := 4; i < len(bot.inspectionSquare.Pix); i += 4 {
		if bot.inspectionSquare.Pix[i] != bot.cmpSquare.Pix[i] {
			return false
		}
	}

	return true
}

// searches currentPosition image for the cursor. returns col and row of found
// cursor. -1 if it is not found.
//
// rows are flipped to normal chess orientation. rows are counted from the
// bottom (white side) in chess.
func (bot *videoChessBot) lookForCursor() (int, int) {
	// counting from top-to-bottom and left-to-right
	for row := 0; row < 8; row++ {
		for col := 0; col < 8; col++ {
			if bot.isCursor(bot.currentPosition, col, row) {
				return col, 8 - row
			}
		}
	}

	return -1, -1
}

// checks the seearchImage at the square indicated by col/row to see if it is
// contains the cursor
func (bot *videoChessBot) isCursor(searchImage *image.RGBA, col int, row int) bool {
	clip := bot.getRect(col, row)

	draw.Draw(bot.inspectionSquare, bot.inspectionSquare.Bounds(), searchImage.SubImage(clip), clip.Min, draw.Src)

	m := 0
	if col%2 == 0 {
		for i := range bot.inspectionSquare.Pix {
			if cursorSampleDataEvenColumns[i] > 0 && bot.inspectionSquare.Pix[i] == cursorSampleDataEvenColumns[i] {
				m++
			}
		}
	} else {
		for i := range bot.inspectionSquare.Pix {
			if cursorSampleDataOddColumns[i] > 0 && bot.inspectionSquare.Pix[i] == cursorSampleDataOddColumns[i] {
				m++
			}
		}
	}

	return m == 76
}

func (bot *videoChessBot) isEmptySquare(searchImage *image.RGBA, col int, row int) bool {
	clip := bot.getRect(col, row)

	draw.Draw(bot.inspectionSquare, bot.inspectionSquare.Bounds(), searchImage.SubImage(clip), clip.Min, draw.Src)
	seq := bot.inspectionSquare.Pix[:4]

	for i := 4; i < len(bot.inspectionSquare.Pix); i += 4 {
		if bot.inspectionSquare.Pix[i] != seq[0] || bot.inspectionSquare.Pix[i+1] != seq[1] || bot.inspectionSquare.Pix[i+2] != seq[2] {
			return false
		}
	}

	return true
}

// number of frames to leave stick down. too long and the cursor will move
// more than one square
const downDuration = 20

// moves cursor once in the suppied direction. tries as often as necessary
// until it hears the audible feedback.
func (bot *videoChessBot) moveCursorOnceStep(portid plugging.PortID, direction ports.Event) {
	bot.waitForFrames(1)

	draining := true
	for draining {
		select {
		case <-bot.obs.audioFeedback:
		default:
			draining = false
		}
	}

	waiting := true
	for waiting {
		bot.vcs.QueueEvent(ports.InputEvent{Time: bot.tv.GetCoords(), Port: portid, Ev: direction, D: ports.DataStickTrue})
		bot.waitForFrames(downDuration)
		bot.vcs.QueueEvent(ports.InputEvent{Time: bot.tv.GetCoords(), Port: portid, Ev: direction, D: ports.DataStickFalse})
		select {
		case <-bot.obs.audioFeedback:
			waiting = false
		default:
		}
	}
}

// move cursor by the number of columns/rows indicated. negative columns
// indicate right and negative rows indicate up.
func (bot *videoChessBot) moveCursor(moveCol int, moveRow int, shortcut bool) {
	if moveCol == moveRow || -moveCol == moveRow {
		fmt.Printf("* moving cursor (diagonally) %d %d\n", moveCol, moveRow)

		move := moveCol
		if move < 0 {
			move = -move
		}

		var direction ports.Event

		if moveCol > 0 && moveRow > 0 {
			direction = ports.LeftDown
		} else if moveCol > 0 && moveRow < 0 {
			direction = ports.LeftUp
		} else if moveCol < 0 && moveRow > 0 {
			direction = ports.RightDown
		} else if moveCol < 0 && moveRow < 0 {
			direction = ports.RightUp
		}

		for i := 0; i < move; i++ {
			bot.moveCursorOnceStep(plugging.PortLeftPlayer, direction)
		}
	} else {
		fmt.Printf("* moving cursor %d %d\n", moveCol, moveRow)

		if shortcut {
			if moveCol > 4 {
				moveCol -= 8
				moveRow-- // correct moveRow caused by board wrap
			} else if moveCol < -4 {
				moveCol += 8
				moveRow++ // correct moveRow caused by board wrap
			}
			if moveRow > 4 {
				moveRow -= 8
			} else if moveRow < -4 {
				moveRow += 8
			}
		}

		if moveCol > 0 {
			for i := 0; i < moveCol; i++ {
				bot.moveCursorOnceStep(plugging.PortLeftPlayer, ports.Left)
			}
		} else if moveCol < 0 {
			for i := 0; i > moveCol; i-- {
				bot.moveCursorOnceStep(plugging.PortLeftPlayer, ports.Right)
			}
		}

		if moveRow > 0 {
			for i := 0; i < moveRow; i++ {
				bot.moveCursorOnceStep(plugging.PortLeftPlayer, ports.Down)
			}
		} else if moveRow < 0 {
			for i := 0; i > moveRow; i-- {
				bot.moveCursorOnceStep(plugging.PortLeftPlayer, ports.Up)
			}
		}
	}

	bot.waitForFrames(1)

	draining := true
	for draining {
		select {
		case <-bot.obs.audioFeedback:
		default:
			draining = false
		}
	}

	waiting := true
	for waiting {
		bot.vcs.QueueEvent(ports.InputEvent{Time: bot.tv.GetCoords(), Port: plugging.PortLeftPlayer, Ev: ports.Fire, D: true})
		bot.waitForFrames(downDuration)
		bot.vcs.QueueEvent(ports.InputEvent{Time: bot.tv.GetCoords(), Port: plugging.PortLeftPlayer, Ev: ports.Fire, D: false})
		select {
		case <-bot.obs.audioFeedback:
			waiting = false
		default:
		}
	}
}

// block goroutine until n frames have passed
func (bot *videoChessBot) waitForFrames(n int) {
	for i := 0; i < n; i++ {
		<-bot.obs.analysis
	}
}

// VideoChessBot creates a new bot able to play chess (via a UCI engine).
func VideoChessBot(vcs VCS, tv TV) (chan *image.RGBA, error) {
	bot := videoChessBot{
		obs:              newObserver(),
		vcs:              vcs,
		tv:               tv,
		prevPosition:     image.NewRGBA(image.Rect(0, 0, specification.ClksScanline, specification.AbsoluteMaxScanlines)),
		inspectionSquare: image.NewRGBA(image.Rect(0, 0, squareWidth, squareHeight)),
		cmpSquare:        image.NewRGBA(image.Rect(0, 0, squareWidth, squareHeight)),
		debuggingRender:  make(chan *image.RGBA, 1),
	}

	tv.AddPixelRenderer(bot.obs)
	tv.AddAudioMixer(bot.obs)

	uci, err := uci.NewUCI("/usr/local/bin/stockfish")
	if err != nil {
		return nil, curated.Errorf("bot: %v", err)
	}

	uci.Start()

	go func() {
		started := false
		for !started {
			bot.currentPosition = <-bot.obs.analysis
			imgHash := sha1.Sum(bot.currentPosition.Pix)
			started = startingImage == imgHash
		}

		fmt.Println("* Video Chess recognised")

		// bot is playing white so ask for first move by submitting an empty move
		uci.SubmitMove <- ""

		for {
			var cursorCol, cursorRow int

			waiting := true
			for waiting {
				bot.currentPosition = <-bot.obs.analysis
				cursorCol, cursorRow = bot.lookForCursor()
				waiting = cursorCol == -1 || cursorRow == -1
				bot.commitDebuggingRender()
			}

			// wait for move from UCI engine
			move := <-uci.GetMove
			fmt.Printf("* playing move %s\n", move)

			// convert move into row/col numbers
			fromCol := int(move[0]) - 97
			fromRow := int(move[1]) - 48
			toCol := int(move[2]) - 97
			toRow := int(move[3]) - 48

			// move to piece being moved
			moveCol := cursorCol - fromCol
			moveRow := cursorRow - fromRow
			bot.moveCursor(moveCol, moveRow, true)
			bot.commitDebuggingRender()

			// move piece to new position
			moveCol = fromCol - toCol
			moveRow = fromRow - toRow
			bot.moveCursor(moveCol, moveRow, false)

			// correct from/to row values after moving
			fromRow = 8 - fromRow
			toRow = 8 - toRow

			// move rectangles for move we just made
			bot.moveFrom = bot.getRect(fromCol, fromRow)
			bot.moveTo = bot.getRect(toCol, toRow)
			bot.commitDebuggingRender()

			// moved piece flashes so we should wait for the correct frame and
			// then remember the white position
			waiting = true
			for waiting {
				bot.currentPosition = <-bot.obs.analysis
				waiting = bot.isEmptySquare(bot.currentPosition, toCol, toRow)
				bot.commitDebuggingRender()
			}
			copy(bot.prevPosition.Pix, bot.currentPosition.Pix)

			// wait for board to disappear to indicate the the VCS is thinking
			waiting = true
			for waiting {
				bot.currentPosition = <-bot.obs.analysis
				waiting = bot.lookForBoard()
			}

			fmt.Println("* vcs is thinking")

			// wait for it to reappear to indicate that the VCS move has been made
			waiting = true
			for waiting {
				bot.currentPosition = <-bot.obs.analysis
				waiting = !bot.lookForBoard()
			}

			// figure out which piece has been moved
			waiting = true
			for waiting {
				bot.currentPosition = <-bot.obs.analysis
				fromCol, fromRow, toCol, toRow = bot.lookForVCSMove()
				waiting = fromCol == -1 || fromRow == -1 || toCol == -1 || toRow == -1
			}

			// highlight 'from' and 'to' squanre
			bot.moveFrom = bot.getRect(fromCol, fromRow)
			bot.moveTo = bot.getRect(toCol, toRow)

			// convert VCS move information to UCI notation
			move = fmt.Sprintf("%c%d%c%d", rune(fromCol+97), fromRow, rune(toCol+97), toRow)
			fmt.Printf("* vcs played %s\n", move)

			// submit to UCI engine
			uci.SubmitMove <- move

			// draw move just made by the VCS
			bot.moveFrom = bot.getRect(fromCol, 8-fromRow)
			bot.moveTo = bot.getRect(toCol, 8-toRow)
			bot.commitDebuggingRender()
		}
	}()

	return bot.debuggingRender, nil
}

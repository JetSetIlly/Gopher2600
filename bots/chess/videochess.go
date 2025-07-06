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

package chess

import (
	"crypto/sha1"
	"fmt"
	"image"
	"image/color"
	"time"

	"golang.org/x/image/draw"

	"github.com/jetsetilly/gopher2600/bots"
	"github.com/jetsetilly/gopher2600/bots/chess/uci"
	"github.com/jetsetilly/gopher2600/hardware/riot/ports"
	"github.com/jetsetilly/gopher2600/hardware/riot/ports/plugging"
	"github.com/jetsetilly/gopher2600/hardware/television/frameinfo"
	"github.com/jetsetilly/gopher2600/hardware/television/signal"
	"github.com/jetsetilly/gopher2600/hardware/television/specification"
)

const (
	boardMarginLeft = 44
	boardMarginTop  = 69
	squareWidth     = 8
	squareHeight    = 17
)

// this is a hash of the entire screen image, including VBLANK/Overscan areas.
// the size of the screen image is based on the AbsoluteMaxScanlines value in
// the specification package. if that value changes then this hash value must
// also change - or the videochess start screen will not be recognised
var startingImage = [sha1.Size]byte{60, 51, 34, 16, 251, 1, 70, 196, 147, 246, 123, 2, 183, 25, 111, 234, 48, 23, 80, 225}

// there are four cursor images. one for each coloured square on both odd and
// even numbered rows. don't forget that column and row number are counted from
// the top-left corner
var cursorImage = [...][sha1.Size]byte{
	{155, 61, 232, 80, 214, 19, 81, 163, 184, 45, 167, 162, 27, 35, 73, 209, 29, 227, 184, 0},      // d4 (col 3, row 4)
	{142, 201, 83, 191, 169, 64, 92, 209, 204, 138, 6, 136, 45, 242, 238, 145, 109, 122, 109, 125}, // e4 (col 4, row 4)
	{27, 156, 79, 167, 22, 186, 233, 87, 90, 57, 156, 255, 221, 83, 57, 70, 176, 201, 35, 210},     // e5 (col 4, row 3)
	{58, 9, 135, 213, 96, 111, 211, 137, 108, 55, 220, 58, 50, 117, 199, 15, 76, 166, 31, 220},     // d5 (col 3, row 3)
}

var levelIndicator = [...][sha1.Size]byte{
	{163, 4, 212, 204, 243, 94, 121, 244, 103, 104, 169, 179, 24, 114, 50, 157, 59, 207, 63, 154}, // level1
	{255, 62, 159, 165, 200, 135, 129, 79, 122, 29, 145, 216, 152, 163, 94, 67, 41, 241, 153, 19},
	{68, 38, 186, 189, 190, 20, 33, 167, 160, 1, 59, 190, 0, 114, 152, 141, 142, 228, 233, 223},
	{4, 197, 14, 9, 146, 2, 158, 170, 9, 10, 9, 147, 190, 172, 243, 170, 132, 30, 131, 229},
	{69, 124, 66, 248, 157, 212, 184, 208, 13, 169, 58, 94, 110, 149, 164, 93, 184, 43, 91, 74},
	{108, 122, 47, 188, 248, 114, 188, 89, 24, 177, 146, 188, 8, 185, 44, 213, 242, 181, 152, 115},
	{241, 88, 253, 41, 69, 140, 246, 53, 128, 26, 97, 239, 78, 155, 61, 181, 233, 239, 164, 136},
	{189, 142, 128, 67, 62, 140, 105, 150, 8, 154, 196, 80, 86, 234, 187, 44, 187, 229, 158, 108}, // level 8
}

type observer struct {
	frameInfo     frameinfo.Current
	img           *image.RGBA
	analysis      chan *image.RGBA
	audioFeedback chan bool
}

func newObserver() *observer {
	obs := &observer{
		img:           image.NewRGBA(image.Rect(0, 0, specification.ClksScanline, specification.AbsoluteMaxScanlines)),
		analysis:      make(chan *image.RGBA, 1),
		audioFeedback: make(chan bool, 1),
	}

	obs.Resize(frameinfo.NewCurrent(specification.SpecNTSC))
	obs.Reset()

	return obs
}

const audioSilence = 15

func (o *observer) SetAudio(sig []signal.AudioSignalAttributes) error {
	for _, s := range sig {
		if s.AudioChannel0 != audioSilence {
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

func (obs *observer) Resize(frameInfo frameinfo.Current) error {
	obs.frameInfo = frameInfo
	return nil
}

func (obs *observer) NewFrame(frameInfo frameinfo.Current) error {
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
		// handle VBLANK by setting pixels to black. we also manually handle
		// NoSignal in the same way
		if sig[i].VBlank || sig[i].Index == signal.NoSignal {
			col = obs.frameInfo.Spec.GetColor(signal.ZeroBlack)
		} else {
			col = obs.frameInfo.Spec.GetColor(sig[i].Color)
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

type chessColors struct {
	blackSquare color.RGBA
	whiteSquare color.RGBA
	blackPiece  color.RGBA
	whitePiece  color.RGBA
}

type videoChessBot struct {
	obs *observer

	input bots.Input
	tv    bots.TV

	// quit as soon as possible when a value appears on the channel
	quit     chan bool
	quitting bool

	// the various colors used by video chess
	colors chessColors

	// the most recent position
	currentPosition *image.RGBA

	// a copy of the board at the end of the previous move by white
	prevPosition *image.RGBA

	// the square last looked at in detail
	inspectionSquare *image.RGBA
	cmpSquare        *image.RGBA

	moveFrom image.Rectangle
	moveTo   image.Rectangle

	feedback bots.Feedback

	// the detected video chess level
	level int
}

// composites a simple debugging image and forward it onto the feeback.Images channel.
func (bot *videoChessBot) commitDebuggingRender() {
	img := *bot.currentPosition
	img.Pix = make([]uint8, len(bot.currentPosition.Pix))
	copy(img.Pix, bot.currentPosition.Pix)

	col := color.RGBA{200, 50, 50, 100}
	draw.Draw(&img, bot.moveFrom, &image.Uniform{col}, image.Point{}, draw.Over)
	col = color.RGBA{50, 200, 50, 100}
	draw.Draw(&img, bot.moveTo, &image.Uniform{col}, image.Point{}, draw.Over)

	r := bot.inspectionSquare.Bounds()
	r.Max = r.Max.Mul(3)
	draw.NearestNeighbor.Scale(&img, r.Add(image.Point{X: 10, Y: 10}), bot.inspectionSquare, bot.inspectionSquare.Bounds(), draw.Src, nil)

	select {
	case bot.feedback.Images <- &img:
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

// checks for board by looking for level indicator - board disappears during
// "thinking" so it's important that we can detect the presence of the board,
// indicating that it's our turn
//
// returns true if level indicator is visible
func (bot *videoChessBot) lookForBoard() bool {
	clip := bot.getRect(2, -1)
	draw.Draw(bot.inspectionSquare, bot.inspectionSquare.Bounds(), bot.currentPosition.SubImage(clip), clip.Min, draw.Src)

	return levelIndicator[bot.level] == sha1.Sum(bot.inspectionSquare.Pix)
}

// get level indicator for future calls to lookForBoard
func (bot *videoChessBot) getLevel() bool {
	clip := bot.getRect(2, -1)
	draw.Draw(bot.inspectionSquare, bot.inspectionSquare.Bounds(), bot.currentPosition.SubImage(clip), clip.Min, draw.Src)

	hash := sha1.Sum(bot.inspectionSquare.Pix)

	for b := range levelIndicator {
		if levelIndicator[b] == hash {
			bot.level = b
			return true
		}
	}

	return false
}

// compares currentPosition with prevPosition for a change. returns fromCol,
// fromRow and toCol, toRow - indicating the move
//
// returned row value is flipped to normal chess orientation. rows are counted from the
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
					if !bot.cmpBlackPosition(col, row) {
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

// returns false if equivalent squares in the currentPosition and prevPosition
// images are different. only empty sqaures and black pieces are considered.
func (bot *videoChessBot) cmpBlackPosition(col int, row int) bool {
	clip := bot.getRect(col, row)

	draw.Draw(bot.inspectionSquare, bot.inspectionSquare.Bounds(), bot.currentPosition.SubImage(clip), clip.Min, draw.Src)
	draw.Draw(bot.cmpSquare, bot.inspectionSquare.Bounds(), bot.prevPosition.SubImage(clip), clip.Min, draw.Src)

	// it just happens to be that the red component of all the possible colors
	// are different so we can restrict our inspection just to that component.
	for i := 0; i < len(bot.inspectionSquare.Pix); i += 4 {

		// limit inspection to the black pieces. we do this by returning true,
		// indicating that the square hasn't changed
		r := bot.inspectionSquare.Pix[i]
		if r == bot.colors.whitePiece.R {
			return true
		}

		// value is different. return false to indicate difference
		if bot.inspectionSquare.Pix[i] != bot.cmpSquare.Pix[i] {
			return false
		}
	}

	// nothing has changed
	return true
}

// searches currentPosition image for the cursor. returns col and row of found
// cursor. -1 if it is not found.
//
// returned row value is flipped to normal chess orientation. rows are counted
// from the bottom (white side) in chess.
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
	hash := sha1.Sum(bot.inspectionSquare.Pix)

	for _, c := range cursorImage {
		if c == hash {
			return true
		}
	}

	return false
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
		bot.input.PushEvent(ports.InputEvent{Port: portid, Ev: direction, D: ports.DataStickTrue})
		bot.waitForFrames(downDuration)
		bot.input.PushEvent(ports.InputEvent{Port: portid, Ev: direction, D: ports.DataStickFalse})
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
		bot.feedback.Diagnostic <- bots.Diagnostic{
			Group:      "VideoChess",
			Diagnostic: fmt.Sprintf("moving cursor (diagonally) %d %d\n", moveCol, moveRow),
		}

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
			bot.moveCursorOnceStep(plugging.PortLeft, direction)
		}
	} else {
		bot.feedback.Diagnostic <- bots.Diagnostic{
			Group:      "VideoChess",
			Diagnostic: fmt.Sprintf("moving cursor %d %d\n", moveCol, moveRow),
		}

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
				bot.moveCursorOnceStep(plugging.PortLeft, ports.Left)
			}
		} else if moveCol < 0 {
			for i := 0; i > moveCol; i-- {
				bot.moveCursorOnceStep(plugging.PortLeft, ports.Right)
			}
		}

		if moveRow > 0 {
			for i := 0; i < moveRow; i++ {
				bot.moveCursorOnceStep(plugging.PortLeft, ports.Down)
			}
		} else if moveRow < 0 {
			for i := 0; i > moveRow; i-- {
				bot.moveCursorOnceStep(plugging.PortLeft, ports.Up)
			}
		}
	}

	bot.waitForFrames(1)

	draining := true
	for draining {
		select {
		case <-bot.obs.audioFeedback:
		case <-bot.quit:
			bot.quitting = true
			return
		default:
			draining = false
		}
	}

	waiting := true
	for waiting {
		bot.input.PushEvent(ports.InputEvent{Port: plugging.PortLeft, Ev: ports.Fire, D: true})
		bot.waitForFrames(downDuration)
		bot.input.PushEvent(ports.InputEvent{Port: plugging.PortLeft, Ev: ports.Fire, D: false})
		select {
		case <-bot.obs.audioFeedback:
			waiting = false
		case <-bot.quit:
			bot.quitting = true
			return
		default:
		}
	}
}

// block goroutine until n frames have passed
func (bot *videoChessBot) waitForFrames(n int) {
	for i := 0; i < n; i++ {
		select {
		case <-bot.obs.analysis:
		case <-bot.quit:
			bot.quitting = true
			return
		}
	}
}

// NewVideoChess creates a new bot able to play chess (via a UCI engine).
func NewVideoChess(vcs bots.Input, tv bots.TV, specID string) (bots.Bot, error) {
	if specID != "NTSC" {
		return nil, fmt.Errorf("videochess: television spec %s is unsupported", specID)
	}

	bot := &videoChessBot{
		obs:              newObserver(),
		input:            vcs,
		tv:               tv,
		quit:             make(chan bool),
		prevPosition:     image.NewRGBA(image.Rect(0, 0, specification.ClksScanline, specification.AbsoluteMaxScanlines)),
		inspectionSquare: image.NewRGBA(image.Rect(0, 0, squareWidth, squareHeight)),
		cmpSquare:        image.NewRGBA(image.Rect(0, 0, squareWidth, squareHeight)),
		feedback: bots.Feedback{
			Images:     make(chan *image.RGBA, 1),
			Diagnostic: make(chan bots.Diagnostic, 64),
		},
	}

	// the colors used by VideoChess. this is for the NTSC version of VideoChess
	//
	// 130 = black squares
	// 132 = white squares
	// 38 = black pieces
	// 142 = white pieces
	bot.colors.blackSquare = specification.SpecNTSC.GetColor(130)
	bot.colors.whiteSquare = specification.SpecNTSC.GetColor(132)
	bot.colors.blackPiece = specification.SpecNTSC.GetColor(38)
	bot.colors.whitePiece = specification.SpecNTSC.GetColor(142)

	tv.AddPixelRenderer(bot.obs)
	tv.AddAudioMixer(bot.obs)

	ucii, err := uci.NewUCI("/usr/local/bin/stockfish", bot.feedback.Diagnostic)
	if err != nil {
		ucii, err = uci.NewUCI("/usr/bin/stockfish", bot.feedback.Diagnostic)
		if err != nil {
			ucii, err = uci.NewUCI("/usr/games/stockfish", bot.feedback.Diagnostic)
			if err != nil {
				return nil, fmt.Errorf("videochess: %w", err)
			}
		}
	}

	ucii.Start()

	go func() {
		defer func() {
			ucii.Quit <- true
			<-bot.quit
		}()

		// look for board
		started := false
		for !started {
			select {
			case bot.currentPosition = <-bot.obs.analysis:
			case <-bot.quit:
				return
			}
			imgHash := sha1.Sum(bot.currentPosition.Pix)
			started = startingImage == imgHash
		}

		bot.feedback.Diagnostic <- bots.Diagnostic{
			Group:      "VideoChess",
			Diagnostic: "video chess recognised",
		}

		bot.commitDebuggingRender()

		// wait for a short period to give time for user to set play level
		bot.feedback.Diagnostic <- bots.Diagnostic{
			Group:      "VideoChess",
			Diagnostic: "waiting 5 seconds before beginning",
		}

		dur, _ := time.ParseDuration("5s")
		select {
		case <-time.After(dur):
		case <-bot.quit:
			return
		}

		// consume two frames before checking video chess level. the first one
		// will have been waiting for a while and will be stale
		select {
		case bot.currentPosition = <-bot.obs.analysis:
		case <-bot.quit:
			return
		}
		select {
		case bot.currentPosition = <-bot.obs.analysis:
		case <-bot.quit:
			return
		}

		// get video chess level
		if bot.getLevel() {
			bot.feedback.Diagnostic <- bots.Diagnostic{
				Group:      "VideoChess",
				Diagnostic: fmt.Sprintf("playing at level %d", bot.level+1),
			}
		} else {
			bot.feedback.Diagnostic <- bots.Diagnostic{
				Group:      "VideoChess",
				Diagnostic: "unrecognised level",
			}
			return
		}

		// bot is playing white so ask for first move by submitting an empty move
		ucii.SubmitMove <- ""

		for {
			var cursorCol, cursorRow int

			waiting := true
			for waiting {
				select {
				case bot.currentPosition = <-bot.obs.analysis:
				case <-bot.quit:
					return
				}
				cursorCol, cursorRow = bot.lookForCursor()
				waiting = cursorCol == -1 || cursorRow == -1
				bot.commitDebuggingRender()
			}

			// wait for move from UCI engine
			var move string
			select {
			case move = <-ucii.GetMove:
				bot.feedback.Diagnostic <- bots.Diagnostic{
					Group:      "VideoChess",
					Diagnostic: fmt.Sprintf("playing move %s\n", move),
				}
			case <-bot.quit:
				return
			}

			// convert move into row/col numbers
			fromCol := int(move[0]) - 97
			fromRow := int(move[1]) - 48
			toCol := int(move[2]) - 97
			toRow := int(move[3]) - 48

			// move to piece being moved
			moveCol := cursorCol - fromCol
			moveRow := cursorRow - fromRow
			bot.moveCursor(moveCol, moveRow, true)
			if bot.quitting {
				return
			}
			bot.commitDebuggingRender()

			// move piece to new position
			moveCol = fromCol - toCol
			moveRow = fromRow - toRow
			bot.moveCursor(moveCol, moveRow, false)
			if bot.quitting {
				return
			}

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
				select {
				case bot.currentPosition = <-bot.obs.analysis:
				case <-bot.quit:
					return
				}
				waiting = bot.isEmptySquare(bot.currentPosition, toCol, toRow)
				bot.commitDebuggingRender()
			}
			copy(bot.prevPosition.Pix, bot.currentPosition.Pix)

			// wait for board to disappear to indicate the the VCS is thinking
			waiting = true
			for waiting {
				select {
				case bot.currentPosition = <-bot.obs.analysis:
				case <-bot.quit:
					return
				}
				waiting = bot.lookForBoard()
			}

			bot.feedback.Diagnostic <- bots.Diagnostic{
				Group:      "VideoChess",
				Diagnostic: "vcs is thinking",
			}

			// wait for it to reappear to indicate that the VCS move has been made
			waiting = true
			for waiting {
				select {
				case bot.currentPosition = <-bot.obs.analysis:
				case <-bot.quit:
					return
				}
				waiting = !bot.lookForBoard()
			}

			// figure out which piece has been moved
			waiting = true
			for waiting {
				select {
				case bot.currentPosition = <-bot.obs.analysis:
				case <-bot.quit:
					return
				}
				fromCol, fromRow, toCol, toRow = bot.lookForVCSMove()
				waiting = fromCol == -1 || fromRow == -1 || toCol == -1 || toRow == -1
			}

			// highlight 'from' and 'to' squanre
			bot.moveFrom = bot.getRect(fromCol, fromRow)
			bot.moveTo = bot.getRect(toCol, toRow)

			move = fmt.Sprintf("%c%d%c%d", rune(fromCol+97), fromRow, rune(toCol+97), toRow)

			// convert VCS move information to UCI notation
			bot.feedback.Diagnostic <- bots.Diagnostic{
				Group:      "VideoChess",
				Diagnostic: fmt.Sprintf("vcs played %s\n", move),
			}

			// submit to UCI engine
			ucii.SubmitMove <- move

			// draw move just made by the VCS
			bot.moveFrom = bot.getRect(fromCol, 8-fromRow)
			bot.moveTo = bot.getRect(toCol, 8-toRow)
			bot.commitDebuggingRender()
		}
	}()

	return bot, nil
}

// BotID implements the bots.Bot interface.
func (bot *videoChessBot) BotID() string {
	return "videochess"
}

// Quit implements the bots.Bot interface.
func (bot *videoChessBot) Quit() {
	// wait until quit has been honoured
	bot.quit <- true
	bot.tv.RemovePixelRenderer(bot.obs)
	bot.tv.RemoveAudioMixer(bot.obs)
}

// Feedback implements the bots.Bot interface.
func (bot *videoChessBot) Feedback() *bots.Feedback {
	return &bot.feedback
}

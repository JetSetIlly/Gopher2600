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

// Package spacejockey is a bot capable of playing Space Jockey.
package spacejockey

import (
	"crypto/sha1"
	"fmt"
	"image"
	"image/color"
	"time"

	"golang.org/x/image/draw"

	"github.com/jetsetilly/gopher2600/bots"
	"github.com/jetsetilly/gopher2600/hardware/riot/ports"
	"github.com/jetsetilly/gopher2600/hardware/riot/ports/plugging"
	"github.com/jetsetilly/gopher2600/hardware/television"
	"github.com/jetsetilly/gopher2600/hardware/television/colourgen"
	"github.com/jetsetilly/gopher2600/hardware/television/signal"
	"github.com/jetsetilly/gopher2600/hardware/television/specification"
)

type observer struct {
	frameInfo     television.FrameInfo
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

	obs.Resize(television.NewFrameInfo(specification.SpecNTSC))
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
		// handle VBLANK by setting pixels to black. we also manually handle
		// NoSignal in the same way
		if sig[i].VBlank || sig[i].Index == signal.NoSignal {
			col = obs.frameInfo.Spec.GetColor(signal.VideoBlack)
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

var startRect = image.Rectangle{Min: image.Point{X: 117, Y: 206}, Max: image.Point{X: 174, Y: 239}}
var startHash = [sha1.Size]byte{0xc8, 0x78, 0x33, 0x7, 0x3b, 0x5a, 0x4b, 0xab, 0xfe, 0x81, 0xbe, 0x54, 0x89, 0xfb, 0xfd, 0x4, 0x40, 0x19, 0x8f, 0xec}

var playerLimitRect = image.Rectangle{Min: image.Point{X: 80, Y: 62}, Max: image.Point{X: 96, Y: 213}}
var playerRect = image.Rectangle{Min: image.Point{X: 80, Y: 194}, Max: image.Point{X: 96, Y: 199}}
var playerHash = [sha1.Size]uint8{0x23, 0x56, 0x8d, 0x92, 0x83, 0xd, 0x98, 0x33, 0x55, 0xc, 0x0, 0x24, 0x56, 0xfc, 0x70, 0xc1, 0xac, 0xf4, 0x8c, 0xe9}

var enemyWindow = image.Rectangle{Min: image.Point{X: 101, Y: 61}, Max: image.Point{X: 223, Y: 202}}

// this is the same as: (playerLimitRect.Max().Y - playerLimitRect.Min().Y) / 2
var playerMidPoint = 137

// the amount of leeway in player move accuracy
var playerFuzz = 5

func cmpSubImage(src *image.RGBA, sub image.Rectangle, hash [sha1.Size]byte) bool {
	searchImage := image.NewRGBA(sub)
	draw.Draw(searchImage, sub, src.SubImage(sub), sub.Min, draw.Src)
	return sha1.Sum(searchImage.Pix) == hash
}

func (bot *spaceJockeyBot) findEnemy() int {
	bot.findEnemyImage = image.NewRGBA(enemyWindow)
	draw.Draw(bot.findEnemyImage, enemyWindow, bot.image.SubImage(enemyWindow), enemyWindow.Min, draw.Src)

	for y := enemyWindow.Min.Y; y < enemyWindow.Max.Y; y++ {
		ct := 0
		for x := enemyWindow.Min.X; x < enemyWindow.Max.X; x++ {
			if bot.findEnemyImage.At(x, y) != colourgen.VideoBlack {
				ct++
			}

			if ct > 4 {
				return y
			}
		}
	}

	return -1
}

func (bot *spaceJockeyBot) findPlayer() int {
	for y := playerLimitRect.Min.Y; y <= playerLimitRect.Max.Y; y++ {
		r := playerRect
		r.Min.Y = y
		r.Max.Y = y + playerRect.Size().Y
		if cmpSubImage(bot.image, r, playerHash) {
			return y
		}
	}
	return -1
}

func (bot *spaceJockeyBot) movePlayerUpToY(y int) {
	py := bot.findPlayer()
	if py == -1 {
		return
	}

	bot.input.PushEvent(ports.InputEvent{Port: plugging.PortLeft, Ev: ports.Up, D: ports.DataStickTrue})
	for y < py && y < py+playerFuzz && py > playerLimitRect.Min.Y {
		select {
		case bot.image = <-bot.obs.analysis:
		case <-bot.quit:
			bot.quitting = true
			return
		}
		py = bot.findPlayer()
		if py == -1 {
			return
		}
	}
	bot.input.PushEvent(ports.InputEvent{Port: plugging.PortLeft, Ev: ports.Up, D: ports.DataStickFalse})

	bot.input.PushEvent(ports.InputEvent{Port: plugging.PortLeft, Ev: ports.Fire, D: true})
	bot.waitForFrames(1)
	bot.input.PushEvent(ports.InputEvent{Port: plugging.PortLeft, Ev: ports.Fire, D: false})
}

func (bot *spaceJockeyBot) movePlayerDownToY(y int) {
	py := bot.findPlayer()
	if py == -1 {
		return
	}

	bot.input.PushEvent(ports.InputEvent{Port: plugging.PortLeft, Ev: ports.Down, D: ports.DataStickTrue})
	for y > py && y > py+playerFuzz && py < playerLimitRect.Max.Y {
		select {
		case bot.image = <-bot.obs.analysis:
		case <-bot.quit:
			bot.quitting = true
			return
		}
		py = bot.findPlayer()
		if py == -1 {
			return
		}
	}
	bot.input.PushEvent(ports.InputEvent{Port: plugging.PortLeft, Ev: ports.Down, D: ports.DataStickFalse})

	bot.input.PushEvent(ports.InputEvent{Port: plugging.PortLeft, Ev: ports.Fire, D: true})
	bot.waitForFrames(1)
	bot.input.PushEvent(ports.InputEvent{Port: plugging.PortLeft, Ev: ports.Fire, D: false})
}

type spaceJockeyBot struct {
	obs *observer

	input bots.Input
	tv    bots.TV

	// quit as soon as possible when a value appears on the channel
	quit     chan bool
	quitting bool

	// the most recent image from the observer
	image *image.RGBA

	findEnemyImage *image.RGBA

	feedback bots.Feedback
}

// NewSpaceJockey creates a new bot able to play Space Jockey.
func NewSpaceJockey(vcs bots.Input, tv bots.TV, specID string) (bots.Bot, error) {
	if specID != "NTSC" {
		return nil, fmt.Errorf("spacejockey: television spec %s is unsupported", specID)
	}

	bot := &spaceJockeyBot{
		obs:   newObserver(),
		input: vcs,
		tv:    tv,
		quit:  make(chan bool),
		feedback: bots.Feedback{
			Images:     make(chan *image.RGBA, 1),
			Diagnostic: make(chan bots.Diagnostic, 64),
		},
	}

	tv.AddPixelRenderer(bot.obs)
	tv.AddAudioMixer(bot.obs)

	go func() {
		started := false
		for !started {
			select {
			case bot.image = <-bot.obs.analysis:
			case <-bot.quit:
				return
			}
			started = cmpSubImage(bot.image, startRect, startHash)
		}

		bot.feedback.Diagnostic <- bots.Diagnostic{
			Group:      bot.BotID(),
			Diagnostic: "space jockey recognised",
		}

		for {
			bot.commitDebuggingRender()

			// wait for new image
			select {
			case bot.image = <-bot.obs.analysis:
			case <-bot.quit:
				return
			}

			// start game if necessary
			if cmpSubImage(bot.image, startRect, startHash) {
				select {
				case <-time.After(2 * time.Second):
					bot.input.PushEvent(ports.InputEvent{Port: plugging.PortLeft, Ev: ports.Fire, D: true})
					bot.waitForFrames(10)
					bot.input.PushEvent(ports.InputEvent{Port: plugging.PortLeft, Ev: ports.Fire, D: false})

					bot.feedback.Diagnostic <- bots.Diagnostic{
						Group:      bot.BotID(),
						Diagnostic: "started game",
					}
				case <-bot.quit:
					return
				}
			}

			enemyY := bot.findEnemy()
			if enemyY == -1 {
				enemyY = playerMidPoint
			}

			playerY := bot.findPlayer()
			if playerY > enemyY {
				bot.movePlayerUpToY(enemyY)
			} else if playerY < enemyY {
				bot.movePlayerDownToY(enemyY)
			}
			if bot.quitting {
				return
			}
		}
	}()

	return bot, nil
}

// BotID implements the bots.Bot interface.
func (bot *spaceJockeyBot) BotID() string {
	return "SpaceJockey"
}

// Quit implements the bots.Bot interface.
func (bot *spaceJockeyBot) Quit() {
	// wait until quit has been honoured
	bot.quit <- true
	bot.tv.RemovePixelRenderer(bot.obs)
	bot.tv.RemoveAudioMixer(bot.obs)
}

// Feedback implements the bots.Bot interface.
func (bot *spaceJockeyBot) Feedback() *bots.Feedback {
	return &bot.feedback
}

// block goroutine until n frames have passed
func (bot *spaceJockeyBot) waitForFrames(n int) {
	for i := 0; i < n; i++ {
		select {
		case <-bot.obs.analysis:
		case <-bot.quit:
			bot.quitting = true
			return
		}
	}
}

// composites a simple debugging image and forward it onto the feeback.Images channel.
func (bot *spaceJockeyBot) commitDebuggingRender() {
	img := *bot.image
	img.Pix = make([]uint8, len(bot.image.Pix))
	copy(img.Pix, bot.image.Pix)

	select {
	case bot.feedback.Images <- &img:
	default:
	}
}

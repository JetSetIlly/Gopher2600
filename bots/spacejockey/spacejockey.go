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
	"math/rand/v2"
	"time"

	"golang.org/x/image/draw"

	"github.com/jetsetilly/gopher2600/bots"
	"github.com/jetsetilly/gopher2600/hardware/riot/ports"
	"github.com/jetsetilly/gopher2600/hardware/riot/ports/plugging"
	"github.com/jetsetilly/gopher2600/hardware/television/frameinfo"
	"github.com/jetsetilly/gopher2600/hardware/television/signal"
	"github.com/jetsetilly/gopher2600/hardware/television/specification"
)

// the hash and the rectangle used to create the hash for the starting state.
// the rectangle only needs to cover the "SPACE JOCKEY" banner at the bottom
var startRect = image.Rectangle{Min: image.Point{X: 116, Y: 213}, Max: image.Point{X: 174, Y: 233}}
var startHash = [sha1.Size]uint8{0x3a, 0xda, 0xec, 0xe4, 0x8a, 0x41, 0x35, 0x49, 0x18, 0xd, 0xbf, 0xf, 0xf2, 0x5d, 0x26, 0x50, 0x83, 0xad, 0xd5, 0x58}

// player is the just the top half of the ship because it never changes. if we
// use the whole ship we would need to have multiple hashes to compare against
var playerHeight = 5
var playerHash = [sha1.Size]uint8{0xe5, 0x3d, 0xb4, 0xe3, 0x5d, 0xa4, 0xed, 0xea, 0x9e, 0xfb, 0xec, 0xeb, 0xa6, 0x65, 0x6d, 0x9b, 0x84, 0x60, 0x59, 0x55}

// the X coords of the player limit rectangle should be exactly the width of the
// rectangle used to create the playerHash
var playerLimitRect = image.Rectangle{Min: image.Point{X: 79, Y: 62}, Max: image.Point{X: 95, Y: 213}}

// the area in which to look for enemies
var enemyWindow = image.Rectangle{Min: image.Point{X: 101, Y: 61}, Max: image.Point{X: 223, Y: 202}}

// mid point of player limit rectangle
var playerMidPoint int

func init() {
	playerMidPoint = (playerLimitRect.Max.Y - playerLimitRect.Min.Y) / 2
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

func cmpSubImage(src *image.RGBA, sub image.Rectangle, hash [sha1.Size]byte) bool {
	searchImage := image.NewRGBA(sub)
	draw.Draw(searchImage, sub, src.SubImage(sub), sub.Min, draw.Src)
	return sha1.Sum(searchImage.Pix) == hash
}

func (bot *spaceJockeyBot) findEnemy() int {
	findEnemyImage := image.NewRGBA(enemyWindow)
	draw.Draw(findEnemyImage, enemyWindow, bot.image.SubImage(enemyWindow), enemyWindow.Min, draw.Src)

	zeroBlack := bot.obs.frameInfo.Spec.GetColor(signal.ZeroBlack)

	for y := enemyWindow.Min.Y; y < enemyWindow.Max.Y; y++ {
		ct := 0
		for x := enemyWindow.Min.X; x < enemyWindow.Max.X; x++ {
			if findEnemyImage.At(x, y) != zeroBlack {
				ct++
			}

			if ct > 4 {
				bot.enemy.Min.X = x
				bot.enemy.Max.X = x + 2
				bot.enemy.Min.Y = y
				bot.enemy.Max.Y = y + 2
				return y
			}
		}
	}

	return -1
}

func (bot *spaceJockeyBot) findPlayer() int {
	for y := playerLimitRect.Min.Y; y <= playerLimitRect.Max.Y; y++ {
		var r image.Rectangle
		r.Min.X = playerLimitRect.Min.X
		r.Max.X = playerLimitRect.Max.X
		r.Min.Y = y
		r.Max.Y = y + playerHeight
		if cmpSubImage(bot.image, r, playerHash) {
			bot.player = r
			return y
		}
	}
	return -1
}

// the amount of fuzz in player move accuracy
var playerFuzz = 5

func (bot *spaceJockeyBot) movePlayerUpToY(y int) {
	py := bot.findPlayer()
	if py == -1 {
		return
	}

	bot.input.PushEvent(ports.InputEvent{Port: plugging.PortLeft, Ev: ports.Up, D: ports.DataStickTrue})

	fy := rand.IntN(playerFuzz)
	for y < py && y < py+fy && py > playerLimitRect.Min.Y {
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

	fy := rand.IntN(playerFuzz)
	for y > py && y > py+fy && py < playerLimitRect.Max.Y {
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

	feedback bots.Feedback

	// quit as soon as possible when a value appears on the channel
	quit     chan bool
	quitting bool

	// the most recent image from the observer
	image *image.RGBA

	// current position of detected sprites
	player image.Rectangle
	enemy  image.Rectangle
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
	for range n {
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

	// highligh player and detected enemy
	col := color.RGBA{50, 200, 50, 100}
	draw.Draw(&img, bot.player, &image.Uniform{col}, image.Point{}, draw.Over)
	draw.Draw(&img, bot.enemy, &image.Uniform{col}, image.Point{}, draw.Over)

	select {
	case bot.feedback.Images <- &img:
	default:
	}
}

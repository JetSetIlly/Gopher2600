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

// Package bots is the root package for the bot subsystem. Sub-packages
// reference this package.
package bots

import (
	"image"

	"github.com/jetsetilly/gopher2600/hardware/riot/ports"
	"github.com/jetsetilly/gopher2600/hardware/television"
)

// TV defines the television functions required by a bot.
type TV interface {
	AddPixelRenderer(television.PixelRenderer)
	AddAudioMixer(television.AudioMixer)
}

// VCS defines the VCS functions required by a bot.
type VCS interface {
	QueueEvent(ports.InputEvent) error
}

// Diagnostic instances are sent over the Feedback Diagnostic channel.
type Diagnostic struct {
	Group      string
	Diagnostic string
}

// Feedback defines the channels that can be used to retrieve information from
// a running bot.
type Feedback struct {
	// consumers of the Images channel should probably only show one frame at a
	// time so a buffer size of 1 is probably sufficient
	Images chan *image.RGBA

	// buffer length of the Log channel should be sufficient long for the bot
	Diagnostic chan Diagnostic
}

// Bot defines the functions the all bots must implement.
type Bot interface {
	BotID() string
	Quit()
	Feedback() *Feedback
}

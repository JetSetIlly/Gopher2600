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

package fonts

import _ "embed"

//go:embed "fa-solid-900.ttf"
var FontAwesome []byte

// Unicode points in FontAwesome for icons used in the application.
const (
	Run                    = '\uf04b'
	Halt                   = '\uf04c'
	BackArrow              = '\uf104'
	BackArrowDouble        = '\uf100'
	UpArrow                = '\uf106'
	UpArrowDouble          = '\uf102'
	StepOver               = '\uf2f9'
	Disk                   = '\uf0c7'
	Mouse                  = '\uf8cc'
	GoingForward           = '\uf01e'
	Persist                = '\uf021'
	Breakpoint             = '\uf06a'
	AudioMute              = '\uf1f6'
	AudioUnmute            = '\uf0f3'
	TermPrompt             = '\uf105'
	ColorSwatch            = '\uf111'
	TapeRewind             = '\uf049'
	TapePlay               = '\uf04b'
	TapeStop               = '\uf04d'
	TapeFastForward        = '\uf04e'
	EmulationPause         = '\uf04c'
	EmulationRun           = '\uf04b'
	EmulationRewindBack    = '\uf04a'
	EmulationRewindForward = '\uf04e'
	EmulationPausedAtStart = '\uf049'
	EmulationPausedAtEnd   = '\uf050'
	MusicNote              = '\uf001'
	VolumeRising           = '\uf062'
	VolumeFalling          = '\uf063'
	Camera                 = '\uf030'
	Chip                   = '\uf2db'
	Unlocked               = '\uf13e'
	CPUKilled              = '\uf714'
	CoProcBug              = '\uf188'
	ExecutionNotes         = '\uf02b'
	CPUBug                 = '\uf188'
	Paw                    = '\uf1b0'
	NonCartExecution       = '\uf54c'
	CoProcExecution        = '\uf135'
	DisasmGotoCurrent      = '\uf530'
	Filter                 = '\uf0b0'
	PageFault              = '\uf0fe'
	Bot                    = '\uf544'
	Warning                = '\uf071'
	CoProcCycles           = '\uf021'
	CoProcLastStart        = '\uf26c'
	CoProcKernel           = '\uf5fd' // layers
	MagnifyingGlass        = '\uf002'
	PaintBrush             = '\uf1fc'
	CaretRight             = '\uf0da'
	TreeOpen               = '\uf0d7'
	TreeClosed             = '\uf0da'
	ByteChange             = '\uf30b'
	Trash                  = '\uf1f8'
	Pointer                = '\uf30b'
	PaintRoller            = '\uf5aa'
	Pencil                 = '\uf303'
	Bug                    = '\uf188'
	Cancel                 = '\uf057'
	TV                     = '\uf108'
	Geometry               = '\uf568'
	Inlined                = '\uf03c'
	SpeechBubble           = '\uf075'
	TimelineOffScreen      = '\uf0a5'
	TimelineJitter         = '\uf0de'
	TimelineComparison     = '\uf02e'
	TimelineComparisonLock = '\uf023'
	Developer              = '\uf0c3'
	Paperclip              = '\uf0c6'
	Directory              = '\uf07c'
	TVBrightness           = '\uf185'
	TVContrast             = '\uf042'
	TVSaturation           = '\uf0e9'
	TVHue                  = '\uf043'
	Elipsis                = '\uf141'
	VBLANKtop              = '\uf35b'
	VBLANKbottom           = '\uf358'
	VBLANKatari            = '\uf14a'
	MeterSegment           = '\uf04d'
	RenderTime             = '\uf06a'
	SelectedTick           = '\uf058'
)

// The first and last unicode points used in the application. We use this to
// make sure we're using as small a font texture as possible.
const (
	FontAwesomeMin = '\ue005'
	FontAwesomeMax = '\uf8ff'
)

//go:embed "Gopher2600-Icons.ttf"
var Gopher2600Icons []byte

// Unicode points in AtariIcons for icons used in the application.
const (
	Stick    = '\ue000'
	Paddle   = '\ue001'
	Keypad   = '\ue002'
	Tape     = '\ue003'
	Wifi     = '\ue004'
	Savekey  = '\ue005'
	Gamepad  = '\ue006'
	AtariVox = '\ue007'
)

// The first and last unicode points used in the application. We use this to
// make sure we're using as small a font texture as possible.
const (
	Gopher2600IconMin = '\ue000'
	Gopher2600IconMax = '\ue007'
)

//go:embed "Hack-Regular.ttf"
var Hack []byte

// The first and last unicode points used in the application. We use this to
// make sure we're using as small a font texture as possible.
const (
	HackMin = '\u0003'
	HackMax = '\u1ef9'
)

//go:embed "JetBrainsMono-Regular.ttf"
var JetBrainsMono []byte

// The first and last unicode points used in the application. We use this to
// make sure we're using as small a font texture as possible.
const (
	JetBrainsMonoMin = '\u0003'
	JetBrainsMonoMax = '\u00ff'
)

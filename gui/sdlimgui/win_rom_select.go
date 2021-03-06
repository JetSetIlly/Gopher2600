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

package sdlimgui

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/inkyblackness/imgui-go/v4"
	"github.com/jetsetilly/gopher2600/cartridgeloader"
	"github.com/jetsetilly/gopher2600/logger"
)

const winSelectROMID = "Select ROM"

type winSelectROM struct {
	img  *SdlImgui
	open bool

	currPath string
	entries  []os.FileInfo
	err      error

	selectedFile string
	showAllFiles bool
	showHidden   bool

	scrollToTop  bool
	centreOnFile bool

	// height of options line at bottom of window. valid after first frame
	controlHeight float32
}

func newFileSelector(img *SdlImgui) (window, error) {
	win := &winSelectROM{
		img:          img,
		showAllFiles: false,
		showHidden:   false,
		scrollToTop:  true,
		centreOnFile: true,
	}

	path, err := os.Getwd()
	win.err = err

	err = win.setPath(path)
	if err != nil {
		return nil, err
	}

	return win, nil
}

func (win *winSelectROM) init() {
}

func (win winSelectROM) id() string {
	return winSelectROMID
}

func (win *winSelectROM) isOpen() bool {
	return win.open
}

func (win *winSelectROM) setOpen(open bool) {
	if open {
		win.open = true

		// goto current cartridge location
		f, err := filepath.Abs(win.img.lz.Cart.Filename)
		if err != nil {
			f = win.img.lz.Cart.Filename
		}

		d := filepath.Dir(f)
		err = win.setPath(d)
		if err != nil {
			logger.Logf("sdlimgui", "error setting path (%s)", d)
		}
		win.selectedFile = win.img.lz.Cart.Filename

		return
	}

	win.open = false
}

func (win *winSelectROM) draw() {
	if !win.open {
		// set centreOnFile to true, ready for next time window is open
		win.centreOnFile = true
		return
	}

	// reset centreOnFile at end of draw
	defer func() { win.centreOnFile = false }()

	imgui.SetNextWindowPosV(imgui.Vec2{70, 58}, imgui.ConditionFirstUseEver, imgui.Vec2{0, 0})
	imgui.SetNextWindowSizeV(imgui.Vec2{375, 397}, imgui.ConditionFirstUseEver)
	imgui.BeginV(win.id(), &win.open, 0)

	if imgui.Button("Parent") {
		d := filepath.Dir(win.currPath)
		err := win.setPath(d)
		if err != nil {
			logger.Logf("sdlimgui", "error setting path (%s)", d)
		}
		win.scrollToTop = true
	}

	imgui.SameLine()
	imgui.Text(win.currPath)

	height := imgui.WindowHeight() - imgui.CursorPosY() - win.controlHeight - imgui.CurrentStyle().FramePadding().Y*2 - imgui.CurrentStyle().ItemInnerSpacing().Y
	imgui.BeginChildV("##selector", imgui.Vec2{X: 0, Y: height}, true, 0)

	if win.scrollToTop {
		imgui.SetScrollY(0)
		win.scrollToTop = false
	}

	// list directories
	imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.ROMSelectDir)
	for _, f := range win.entries {
		// ignore dot files
		if !win.showHidden && f.Name()[0] == '.' {
			continue
		}

		fi, err := os.Stat(filepath.Join(win.currPath, f.Name()))
		if err != nil {
			continue
		}

		if fi.Mode().IsDir() {
			s := strings.Builder{}
			s.WriteString(f.Name())
			s.WriteString(" [dir]")

			if imgui.Selectable(s.String()) {
				d := filepath.Join(win.currPath, f.Name())
				err = win.setPath(d)
				if err != nil {
					logger.Logf("sdlimgui", "error setting path (%s)", d)
				}
				win.scrollToTop = true
			}
		}
	}
	imgui.PopStyleColor()

	// list files
	imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.ROMSelectFile)
	for _, f := range win.entries {
		// ignore dot files
		if !win.showHidden && f.Name()[0] == '.' {
			continue
		}

		fi, err := os.Stat(filepath.Join(win.currPath, f.Name()))
		if err != nil {
			continue
		}

		// ignore invalid file extensions unless showAllFiles flags is set
		ext := strings.ToUpper(filepath.Ext(fi.Name()))
		if !win.showAllFiles {
			hasExt := false
			for _, e := range cartridgeloader.FileExtensions {
				if e == ext {
					hasExt = true
					break
				}
			}
			if !hasExt {
				continue // to next file
			}
		}

		if fi.Mode().IsRegular() {
			selected := f.Name() == filepath.Base(win.selectedFile)

			if selected && win.centreOnFile {
				imgui.SetScrollHereY(0.0)
			}

			if imgui.SelectableV(f.Name(), selected, 0, imgui.Vec2{0, 0}) {
				win.selectedFile = filepath.Join(win.currPath, f.Name())
			}
		}
	}
	imgui.PopStyleColor()

	imgui.EndChild()

	// control buttons. start controlHeight measurement
	win.controlHeight = imguiMeasureHeight(func() {
		imgui.Checkbox("Show all files", &win.showAllFiles)
		imgui.SameLine()
		imgui.Checkbox("Show hidden entries", &win.showHidden)

		imgui.Spacing()

		if imgui.Button("Cancel") {
			win.setOpen(false)
		}

		if win.selectedFile != "" {
			imgui.SameLine()

			var s string

			// load or reload button
			if win.selectedFile == win.img.lz.Cart.Filename {
				s = fmt.Sprintf("Reload %s", filepath.Base(win.selectedFile))
			} else {
				s = fmt.Sprintf("Load %s", filepath.Base(win.selectedFile))
			}

			if imgui.Button(s) {
				// build terminal command and run
				cmd := strings.Builder{}
				cmd.WriteString("INSERT \"")
				cmd.WriteString(win.selectedFile)
				cmd.WriteString("\"")
				win.img.term.pushCommand(cmd.String())
				win.setOpen(false)
			}
		}
	})

	imgui.End()
}

func (win *winSelectROM) setPath(path string) error {
	var err error

	win.currPath = filepath.Clean(path)
	win.entries, err = ioutil.ReadDir(win.currPath)
	win.selectedFile = ""

	return err
}

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

	"github.com/inkyblackness/imgui-go/v2"
	"github.com/jetsetilly/gopher2600/cartridgeloader"
	"github.com/jetsetilly/gopher2600/logger"
)

const winSelectROMTitle = "Select ROM"

type winSelectROM struct {
	windowManagement
	img *SdlImgui

	currPath string
	entries  []os.FileInfo
	err      error

	selectedFile string
	showAllFiles bool
	showHidden   bool

	// height of options line at bottom of window. valid after first frame
	controlHeight float32
}

func newFileSelector(img *SdlImgui) (managedWindow, error) {
	win := &winSelectROM{
		img:          img,
		showAllFiles: false,
		showHidden:   false,
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
	return winSelectROMTitle
}

func (win *winSelectROM) destroy() {
}

func (win *winSelectROM) draw() {
	if !win.open {
		return
	}

	imgui.SetNextWindowPosV(imgui.Vec2{70, 58}, imgui.ConditionFirstUseEver, imgui.Vec2{0, 0})
	imgui.SetNextWindowSizeV(imgui.Vec2{375, 397}, imgui.ConditionFirstUseEver)

	imgui.BeginV(winSelectROMTitle, &win.open, 0)

	if imgui.Button("Parent") {
		d := filepath.Dir(win.currPath)
		err := win.setPath(d)
		if err != nil {
			logger.Log("sdlimgui", fmt.Sprintf("error setting path (%s)", d))
		}
	}

	imgui.SameLine()
	imgui.Text(win.currPath)

	height := imgui.WindowHeight() - imgui.CursorPosY() - win.controlHeight - imgui.CurrentStyle().FramePadding().Y*2 - imgui.CurrentStyle().ItemInnerSpacing().Y
	imgui.BeginChildV("##selector", imgui.Vec2{X: 0, Y: height}, true, 0)

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
					logger.Log("sdlimgui", fmt.Sprintf("error setting path (%s)", d))
				}
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

			if imgui.SelectableV(f.Name(), selected, 0, imgui.Vec2{0, 0}) {
				win.selectedFile = filepath.Join(win.currPath, f.Name())
			}
		}
	}
	imgui.PopStyleColor()

	imgui.EndChild()

	// control buttons. start controlHeight measurement
	controlHeight := imgui.CursorPosY()

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

	// commit controlHeight measurement
	win.controlHeight = imgui.CursorPosY() - controlHeight

	imgui.End()
}

func (win *winSelectROM) setPath(path string) error {
	var err error

	win.currPath = filepath.Clean(path)
	win.entries, err = ioutil.ReadDir(win.currPath)
	win.selectedFile = ""

	return err
}

// overriding managedWindow implementation.
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
			logger.Log("sdlimgui", fmt.Sprintf("error setting path (%s)", d))
		}
		win.selectedFile = win.img.lz.Cart.Filename

		return
	}

	win.open = false
}

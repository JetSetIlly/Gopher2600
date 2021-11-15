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
	"image"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-gl/gl/v3.2-core/gl"
	"github.com/inkyblackness/imgui-go/v4"
	"github.com/jetsetilly/gopher2600/cartridgeloader"
	"github.com/jetsetilly/gopher2600/curated"
	"github.com/jetsetilly/gopher2600/hardware/television/specification"
	"github.com/jetsetilly/gopher2600/logger"
	"github.com/jetsetilly/gopher2600/thumbnailer"
)

const winSelectROMID = "Select ROM"

type winSelectROM struct {
	img  *SdlImgui
	open bool

	currPath string
	entries  []os.DirEntry
	err      error

	selectedFile string
	showAllFiles bool
	showHidden   bool

	scrollToTop  bool
	centreOnFile bool

	// height of options line at bottom of window. valid after first frame
	controlHeight float32

	thmb        *thumbnailer.Thumbnailer
	thmbReceive chan *image.RGBA
	thmbTexture uint32
}

func newFileSelector(img *SdlImgui) (window, error) {
	win := &winSelectROM{
		img:          img,
		showAllFiles: false,
		showHidden:   false,
		scrollToTop:  true,
		centreOnFile: true,
	}

	var err error

	// create a new thumbnailer instance
	win.thmb, err = thumbnailer.NewThumbnailer()
	if err != nil {
		return nil, curated.Errorf("debugger: %v", err)
	}

	path, err := os.Getwd()
	win.err = err
	path = filepath.Join(path, "roms")
	err = win.setPath(path)
	if err != nil {
		return nil, err
	}

	gl.GenTextures(1, &win.thmbTexture)
	gl.BindTexture(gl.TEXTURE_2D, win.thmbTexture)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.NEAREST)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.NEAREST)

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

		path, err := os.Getwd()
		path = filepath.Join(path, "roms")
		err = win.setPath(path)
		if err != nil {
			logger.Logf("sdlimgui", "error setting path (%s)", path)
		}

		// goto current cartridge location. directly accessing filename
		// from VCS - there's very little risk of a race condition here but you
		// never know so we should bear it in mind
		f, err := filepath.Abs(win.img.vcs.Mem.Cart.Filename)
		if err != nil {
			f = win.img.lz.Cart.Filename
		}

		d := filepath.Dir(f)
		err = win.setPath(d)
		if err != nil {
			logger.Logf("sdlimgui", "error setting path (%s)", d)
		}

		win.setSelectedFile(f)

		return
	} else {
		win.thmb.EndCreation()
	}

	win.open = false
}

func (win *winSelectROM) draw() {
	select {
	case img := <-win.thmbReceive:
		if img != nil {
			gl.PixelStorei(gl.UNPACK_ROW_LENGTH, int32(img.Stride)/4)
			defer gl.PixelStorei(gl.UNPACK_ROW_LENGTH, 0)

			gl.BindTexture(gl.TEXTURE_2D, win.thmbTexture)
			gl.TexImage2D(gl.TEXTURE_2D, 0,
				gl.RGBA, int32(img.Bounds().Size().X), int32(img.Bounds().Size().Y), 0,
				gl.RGBA, gl.UNSIGNED_BYTE,
				gl.Ptr(img.Pix))
		}
	default:
	}

	if !win.open {
		// set centreOnFile to true, ready for next time window is open
		win.centreOnFile = true
		return
	}

	// reset centreOnFile at end of draw
	defer func() { win.centreOnFile = false }()

	// window size/positioning dependent on emulation mode. also we don't want
	// to save window settings for playmode
	var flgs imgui.Condition
	var winFlgs imgui.WindowFlags
	if win.img.isPlaymode() {
		flgs = imgui.ConditionAppearing
		winFlgs = imgui.WindowFlagsNoSavedSettings
	} else {
		flgs = imgui.ConditionFirstUseEver
		winFlgs = imgui.WindowFlagsNone
	}
	winFlgs |= imgui.WindowFlagsAlwaysAutoResize

	imgui.SetNextWindowPosV(imgui.Vec2{20, 20}, flgs, imgui.Vec2{0, 0})

	if imgui.BeginV(win.id(), &win.open, winFlgs) {
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

		if imgui.BeginTable("romSelector", 2) {
			imgui.TableNextRow()
			imgui.TableNextColumn()

			height := imgui.WindowHeight() - imgui.CursorPosY() - win.controlHeight - imgui.CurrentStyle().FramePadding().Y*2 - imgui.CurrentStyle().ItemInnerSpacing().Y
			imgui.BeginChildV("##selector", imgui.Vec2{X: 300, Y: height}, true, 0)

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
						win.setSelectedFile(filepath.Join(win.currPath, f.Name()))
					}
				}
			}
			imgui.PopStyleColor()

			imgui.EndChild()

			imgui.TableNextColumn()
			imgui.Image(imgui.TextureID(win.thmbTexture), imgui.Vec2{specification.ClksVisible * 3, specification.AbsoluteMaxScanlines})

			imgui.EndTable()
		}

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
					win.img.dbg.PushRawEvent(func() {
						err := win.img.dbg.InsertCartridge(win.selectedFile)
						if err != nil {
							logger.Logf("sdlimgui", err.Error())
						}
					})
					win.setOpen(false)
				}
			}
		})
	}

	imgui.End()
}

func (win *winSelectROM) setPath(path string) error {
	var err error
	win.currPath = filepath.Clean(path)
	win.entries, err = os.ReadDir(win.currPath)
	if err != nil {
		return err
	}
	win.setSelectedFile("")
	return nil
}

func (win *winSelectROM) setSelectedFile(filename string) {
	win.selectedFile = filename
	if filename == "" {
		return
	}

	cartload, err := cartridgeloader.NewLoader(filename, "AUTO")
	if err != nil {
		logger.Logf("ROM Select", err.Error())
		return
	}

	win.thmbReceive = win.thmb.Create(cartload, thumbnailer.UndefinedNumFrames)
}

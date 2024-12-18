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
	"bytes"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/image/draw"

	"github.com/inkyblackness/imgui-go/v4"
	"github.com/jetsetilly/gopher2600/archivefs"
	"github.com/jetsetilly/gopher2600/cartridgeloader"
	"github.com/jetsetilly/gopher2600/gui/fonts"
	"github.com/jetsetilly/gopher2600/hardware/peripherals"
	"github.com/jetsetilly/gopher2600/hardware/television/specification"
	"github.com/jetsetilly/gopher2600/logger"
	"github.com/jetsetilly/gopher2600/properties"
	"github.com/jetsetilly/gopher2600/resources"
	"github.com/jetsetilly/gopher2600/thumbnailer"
	"github.com/sahilm/fuzzy"
)

const winSelectROMID = "Select ROM"

type winSelectROM struct {
	playmodeWin
	debuggerWin

	img  *SdlImgui
	path archivefs.AsyncPath

	// selectedName is the name of the ROM in a normalised form
	selectedName string

	// properties of selected
	selectedProperties properties.Entry

	showAll    bool
	showHidden bool

	scrollToTop  bool
	centreOnFile bool

	informationOpen bool

	// height of options line at bottom of window. valid after first frame
	controlHeight float32

	// dimensions of the listview selector
	listviewDim imgui.Vec2

	// animated thumbnail of emulation
	thmb          *thumbnailer.Anim
	thmbTexture   texture
	thmbImage     *image.RGBA
	thmbDim       imgui.Vec2
	thmbPosOffset imgui.Vec2

	// the emulation thumbnail is placed inside a child of the following
	// dimensions and which is sized according the the value in both
	// listviewDim and thmbDim
	thmbChildDim imgui.Vec2

	// map of normalised ROM titles to box art images
	boxart        []string
	boxartTexture texture
	boxartDim     imgui.Vec2
	boxartUse     bool
}

// boxart from libretro project
// github.com/libretro-thumbnails/Atari_-_2600/tree/4ea759821724d6c7bcf2f46020d79fc4270ed2f6/Named_Boxarts
const namedBoxarts = "Named_Boxarts"

func newSelectROM(img *SdlImgui) (window, error) {
	win := &winSelectROM{
		img:          img,
		showAll:      false,
		showHidden:   false,
		scrollToTop:  true,
		centreOnFile: true,
	}
	win.debuggerGeom.noFocusTracking = true
	win.path = archivefs.NewAsyncPath(win)

	var err error

	// it is assumed in the polling routines that if the file rom selector is
	// open then the thumbnailer is open. if we ever decide that the thumbnailer
	// should be optional we should change this - we don't want the polling to
	// be high if there is no reason
	win.thmb, err = thumbnailer.NewAnim(win.img.dbg.VCS().Env.Prefs, win.img.dbg.VCS().TV.GetCreationSpecID())
	if err != nil {
		return nil, err
	}

	win.thmbTexture = img.rnd.addTexture(shaderTVColour, true, true)
	win.thmbImage = image.NewRGBA(image.Rect(0, 0, 0, 0))
	win.thmbDim = imgui.Vec2{X: specification.WidthTV, Y: specification.HeightTV}.Times(2.0)
	win.listviewDim = imgui.Vec2{X: 300, Y: win.thmbDim.Y * 1.4}
	win.thmbPosOffset = imgui.Vec2{X: 0, Y: (win.listviewDim.Y - win.thmbDim.Y) / 2}
	win.thmbChildDim = imgui.Vec2{X: win.thmbDim.X, Y: win.listviewDim.Y}

	// load and normalise box art names
	boxartPath, err := resources.JoinPath(namedBoxarts)
	if err != nil {
		logger.Log(logger.Allow, "sdlimgui", err)
	} else {
		boxartFiles, err := os.ReadDir(boxartPath)
		if err != nil {
			logger.Log(logger.Allow, "sdlimgui", err)
		} else {
			for _, n := range boxartFiles {
				win.boxart = append(win.boxart, n.Name())
			}
		}
	}

	// prepare boxart texture
	win.boxartTexture = img.rnd.addTexture(shaderColor, false, false)

	return win, nil
}

func (win *winSelectROM) init() {
}

func (win *winSelectROM) destroy() {
	win.path.Destroy <- true
}

func (win winSelectROM) id() string {
	return winSelectROMID
}

func (win *winSelectROM) setOpen(open bool) {
	if !open {
		win.path.Close <- true
		return
	}

	// open at the most recently selected ROM
	win.path.Set <- win.img.prefs.recentROM.String()
}

func (win *winSelectROM) playmodeSetOpen(open bool) {
	win.playmodeWin.playmodeSetOpen(open)
	win.centreOnFile = true
	win.setOpen(open)

	// set centreOnFile to true, ready for next time window is open
	if !open {
		win.centreOnFile = true
	}
}

func (win *winSelectROM) playmodeDraw() bool {
	if !win.playmodeOpen {
		win.thmb.EndCreation()
		return false
	}

	win.render()

	posFlgs := imgui.ConditionAppearing
	winFlgs := imgui.WindowFlagsNoSavedSettings | imgui.WindowFlagsAlwaysAutoResize

	imgui.SetNextWindowPosV(imgui.Vec2{X: 75, Y: 75}, posFlgs, imgui.Vec2{X: 0, Y: 0})

	if imgui.BeginV(win.playmodeID(win.id()), &win.playmodeOpen, winFlgs) {
		win.draw()
	}

	win.playmodeWin.playmodeGeom.update()
	imgui.End()

	return true
}

func (win *winSelectROM) debuggerSetOpen(open bool) {
	win.debuggerWin.debuggerSetOpen(open)
	win.centreOnFile = true
	win.setOpen(open)

	// set centreOnFile to true, ready for next time window is open
	if !open {
		win.centreOnFile = true
	}
}

func (win *winSelectROM) debuggerDraw() bool {
	if !win.debuggerOpen {
		win.thmb.EndCreation()
		return false
	}

	win.render()

	posFlgs := imgui.ConditionFirstUseEver
	winFlgs := imgui.WindowFlagsAlwaysAutoResize

	imgui.SetNextWindowPosV(imgui.Vec2{X: 75, Y: 75}, posFlgs, imgui.Vec2{X: 0, Y: 0})

	if imgui.BeginV(win.debuggerID(win.id()), &win.debuggerOpen, winFlgs) {
		win.draw()
	}

	win.debuggerWin.debuggerGeom.update()
	imgui.End()

	return true
}

func (win *winSelectROM) render() {
	// receive new thumbnail data and copy to texture
	select {
	case newImage := <-win.thmb.Render:
		if newImage != nil {
			sz := newImage.Bounds().Size()
			if sz != win.thmbImage.Bounds().Size() {
				win.thmbImage = image.NewRGBA(image.Rect(0, 0, sz.X, sz.Y))
				win.thmbTexture.markForCreation()
			}

			// copy new image so that it is centred in the thumbnail image
			draw.Copy(win.thmbImage, image.Point{X: 0, Y: 0},
				newImage, newImage.Bounds(), draw.Over, nil)

			// render image
			win.thmbTexture.render(win.thmbImage)
		}
	default:
	}
}

func (win *winSelectROM) draw() {
	err := win.path.Process()
	if err != nil {
		logger.Log(logger.Allow, "sdlimgui", err)
	}

	imgui.BeginGroup()

	if imgui.Button("Parent") {
		win.path.Set <- filepath.Dir(win.path.Results.Dir)
		win.scrollToTop = true
	}

	imgui.SameLine()
	const maxDisplayPath = 68
	displayPath := archivefs.RemoveArchiveExt(win.path.Results.Dir)
	displayPath = displayPath[max(0, len(displayPath)-maxDisplayPath):]
	if len(displayPath) == maxDisplayPath {
		displayPath = fmt.Sprintf(" %c%s", fonts.BackArrowDouble, displayPath)
	}
	imgui.Text(displayPath)

	// ROM selector listview
	imgui.BeginChildV("##selector", win.listviewDim, true, 0)

	if win.scrollToTop {
		imgui.SetScrollY(0)
		win.scrollToTop = false
	}

	// list directories
	imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.ROMSelectDir)
	for _, e := range win.path.Results.Entries {
		// ignore dot files
		if !win.showHidden && e.Name[0] == '.' {
			continue
		}

		if e.IsDir {
			s := strings.Builder{}
			if e.IsArchive {
				s.WriteString(string(fonts.Paperclip))
				s.WriteString(" ")
				s.WriteString(archivefs.TrimArchiveExt(e.Name))
			} else {
				s.WriteString(string(fonts.Directory))
				s.WriteString(" ")
				s.WriteString(e.Name)
			}

			if imgui.Selectable(s.String()) {
				win.path.Set <- filepath.Join(win.path.Results.Dir, e.Name)
				win.scrollToTop = true
			}
		}
	}
	imgui.PopStyleColor()

	// list files
	imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.ROMSelectFile)
	for _, e := range win.path.Results.Entries {
		// ignore dot files
		if !win.showHidden && e.Name[0] == '.' {
			continue
		}

		// ignore invalid file extensions unless showAll flags is set
		ext := strings.ToUpper(filepath.Ext(e.Name))
		if !win.showAll {
			hasExt := false
			for _, e := range cartridgeloader.FileExtensions {
				if e == ext {
					hasExt = true
					break
				}
			}
			if !hasExt {
				for _, e := range archivefs.ArchiveExtensions {
					if e == ext {
						hasExt = true
						break
					}
				}
			}
			if !hasExt {
				continue // to next file
			}
		}

		if !e.IsDir {
			selected := e.Name == win.path.Results.Base

			if selected && win.centreOnFile {
				imgui.SetScrollHereY(0.0)
				win.centreOnFile = false
			}

			if imgui.SelectableV(e.Name, selected, 0, imgui.Vec2{X: 0, Y: 0}) {
				win.path.Set <- filepath.Join(win.path.Results.Dir, e.Name)
			}
			if imgui.IsItemHovered() && imgui.IsMouseDoubleClicked(0) {
				win.insertCartridge()
			}
		}
	}
	imgui.PopStyleColor()
	imgui.EndChild()

	// emulation thumbnail is on the same line as the listview
	imgui.SameLineV(0, 10)
	imgui.BeginChildV("##emulation", win.thmbChildDim, true, 0)
	imgui.SetCursorScreenPos(imgui.CursorScreenPos().Plus(win.thmbPosOffset))
	imgui.Image(imgui.TextureID(win.thmbTexture.getID()), win.thmbDim)
	imgui.EndChild()

	// control buttons. start controlHeight measurement
	win.controlHeight = imguiMeasureHeight(func() {
		// results of preview emulation from the thumbnailer
		previewResults := win.thmb.UpdateResults()

		imgui.SetNextItemOpen(win.informationOpen, imgui.ConditionAlways)
		if !imgui.CollapsingHeaderV(win.selectedName, imgui.TreeNodeFlagsNone) {
			win.informationOpen = false
		} else {
			win.informationOpen = true
			if imgui.BeginTable("#properties", 3) {
				imgui.TableSetupColumnV("#information", imgui.TableColumnFlagsWidthStretch, -1, 0)
				imgui.TableSetupColumnV("#spacingA", imgui.TableColumnFlagsWidthFixed, -1, 1)
				imgui.TableSetupColumnV("#boxart", imgui.TableColumnFlagsWidthFixed, -1, 2)

				// property table. we measure the height of this table to
				// help centering the box art image in the next column
				imgui.TableNextRow()
				imgui.TableNextColumn()
				propertyTableTop := imgui.CursorPosY()
				if imgui.BeginTable("#properties", 2) {
					imgui.TableSetupColumnV("#category", imgui.TableColumnFlagsWidthFixed, -1, 0)
					imgui.TableSetupColumnV("#detail", imgui.TableColumnFlagsWidthFixed, -1, 1)

					// wrap text
					imgui.PushTextWrapPosV(imgui.CursorPosX() + imgui.ContentRegionAvail().X)
					defer imgui.PopTextWrapPos()

					imgui.TableNextRow()
					imgui.TableNextColumn()
					imgui.Text("Name")
					imgui.TableNextColumn()
					if win.selectedProperties.IsValid() {
						imgui.Text(win.selectedProperties.Name)
					} else {
						imgui.Text(win.selectedName)
					}

					imgui.TableNextRow()
					imgui.TableNextColumn()
					imgui.AlignTextToFramePadding()
					imgui.Text("Mapper")
					imgui.TableNextColumn()
					if previewResults != nil {
						imgui.Text(previewResults.VCS.Mem.Cart.ID())
					} else {
						imgui.Text("-")
					}

					imgui.TableNextRow()
					imgui.TableNextColumn()
					imgui.PushItemFlag(imgui.ItemFlagsDisabled, true)
					imgui.PushStyleVarFloat(imgui.StyleVarAlpha, disabledAlpha)
					imgui.AlignTextToFramePadding()
					imgui.Text("Television")
					imgui.TableNextColumn()
					if previewResults != nil {
						imgui.SetNextItemWidth(80)
						if imgui.BeginCombo("##tvspec", previewResults.SpecID) {
							for _, s := range specification.SpecList {
								if imgui.Selectable(s) {
								}
							}
							imgui.EndCombo()
						}
					} else {
						imgui.Text("-")
					}
					imgui.PopStyleVar()
					imgui.PopItemFlag()

					imgui.TableNextRow()
					imgui.TableNextColumn()
					imgui.PushItemFlag(imgui.ItemFlagsDisabled, true)
					imgui.PushStyleVarFloat(imgui.StyleVarAlpha, disabledAlpha)
					imgui.AlignTextToFramePadding()
					imgui.Text("Players")
					imgui.TableNextColumn()
					if previewResults != nil {
						imgui.SetNextItemWidth(100)
						if imgui.BeginCombo("##leftplayer", string(previewResults.VCS.RIOT.Ports.LeftPlayer.ID())) {
							for _, s := range peripherals.AvailableLeftPlayer {
								if imgui.Selectable(s) {
								}
							}
							imgui.EndCombo()
						}
						imgui.SameLineV(0, 15)
						imgui.Text("&")
						imgui.SameLineV(0, 15)
						imgui.SetNextItemWidth(100)
						if imgui.BeginCombo("##rightplayer", string(previewResults.VCS.RIOT.Ports.RightPlayer.ID())) {
							for _, s := range peripherals.AvailableRightPlayer {
								if imgui.Selectable(s) {
								}
							}
							imgui.EndCombo()
						}
					} else {
						imgui.Text("-")
					}
					imgui.PopStyleVar()
					imgui.PopItemFlag()

					if win.selectedProperties.Manufacturer != "" {
						imgui.TableNextRow()
						imgui.TableNextColumn()
						imgui.AlignTextToFramePadding()
						imgui.Text("Manufacturer")
						imgui.TableNextColumn()
						imgui.Text(win.selectedProperties.Manufacturer)
					}
					if win.selectedProperties.Rarity != "" {
						imgui.TableNextRow()
						imgui.TableNextColumn()
						imgui.AlignTextToFramePadding()
						imgui.Text("Rarity")
						imgui.TableNextColumn()
						imgui.Text(win.selectedProperties.Rarity)
					}
					if win.selectedProperties.Model != "" {
						imgui.TableNextRow()
						imgui.TableNextColumn()
						imgui.AlignTextToFramePadding()
						imgui.Text("Model")
						imgui.TableNextColumn()
						imgui.Text(win.selectedProperties.Model)
					}

					if win.selectedProperties.Note != "" {
						imgui.TableNextRow()
						imgui.TableNextColumn()
						imgui.AlignTextToFramePadding()
						imgui.Text("Note")
						imgui.TableNextColumn()
						imgui.Text(win.selectedProperties.Note)
					}

					imgui.EndTable()
				}
				propertyTableBottom := imgui.CursorPosY()
				propertyTableHeight := propertyTableBottom - propertyTableTop

				// spacing
				imgui.TableNextColumn()

				if win.boxartUse {
					imgui.TableNextColumn()

					// if thumbnail height is less than height of
					// property table then we position the image so that
					// it's centered in relation to the property table
					p := imgui.CursorPos()
					if win.boxartDim.Y < propertyTableHeight {
						p.Y += (propertyTableHeight - win.boxartDim.Y) / 2
						imgui.SetCursorPos(p)
					} else {
						// if height of thumbnail is greater than or
						// equal to height of property table then we add
						// a imgui.Spacing(). this may expand the height
						// of the property table but that's okay
						imgui.Spacing()
					}

					imgui.Image(imgui.TextureID(win.boxartTexture.getID()), win.boxartDim)
				}

				imgui.EndTable()
			}
		}

		imguiSeparator()

		if imgui.Button("Cancel") {
			// close rom selected in both the debugger and playmode
			win.debuggerSetOpen(false)
			win.playmodeSetOpen(false)
		}

		if win.selectedName != "" {
			var s string

			// load or reload button
			if win.path.Results.Selected == win.img.cache.VCS.Mem.Cart.Filename {
				s = fmt.Sprintf("Reload %s", win.selectedName)
			} else {
				s = fmt.Sprintf("Load %s", win.selectedName)
			}

			// only show load cartridge button if the file is being
			// emulated by the thumbnailer. if it's not then that's a good
			// sign that the file isn't supported
			if win.thmb.IsEmulating() {
				imgui.SameLine()
				if imgui.Button(s) {
					win.insertCartridge()
				}
			}
		}
	})

	imgui.EndGroup()

	const romSelectPopupID = "romSelectPopupID"
	if imgui.IsItemHovered() && imgui.IsMouseDown(1) {
		imgui.OpenPopup(romSelectPopupID)
	}

	if imgui.BeginPopup(romSelectPopupID) {
		imgui.Text("Show Options")
		imguiSeparator()
		imgui.Checkbox("All Files", &win.showAll)
		imgui.Checkbox("Hidden", &win.showHidden)
		imgui.EndPopup()
	}
}

func (win *winSelectROM) insertCartridge() {
	// do not try to load cartridge if the file is not being emulated by the
	// thumbnailer. if it's not then that's a good sign that the file isn't
	// supported
	if !win.thmb.IsEmulating() {
		return
	}

	done := make(chan bool)
	win.img.dbg.InsertCartridge(win.path.Results.Selected, done)
	go func() {
		if <-done {
			win.img.prefs.recentROM.Set(win.path.Results.Selected)
		}
	}()

	// close rom selected in both the debugger and playmode
	win.debuggerSetOpen(false)
	win.playmodeSetOpen(false)
}

// imnplements the archivefs.FilenameSetter interface
func (win *winSelectROM) SetSelectedFilename(filename string) {
	// return immediately if the filename is empty
	if filename == "" {
		return
	}

	// create cartridge loader and start thumbnail emulation
	cartload, err := cartridgeloader.NewLoaderFromFilename(filename, "AUTO", "AUTO", win.img.dbg.Properties)
	if err != nil {
		logger.Log(logger.Allow, "ROM Select", err)
		return
	}

	win.selectedProperties = cartload.Property
	win.selectedName = win.selectedProperties.Name
	if win.selectedName == "" {
		win.selectedName = win.path.Results.Base
		win.selectedName = cartridgeloader.NameFromFilename(win.selectedName)
	}

	// normalise ROM name for presentation
	win.selectedName, _, _ = strings.Cut(win.selectedName, "(")
	win.selectedName = strings.TrimSpace(win.selectedName)

	// find box art as best we can
	err = win.findBoxart()
	if err != nil {
		logger.Log(logger.Allow, "sdlimgui", err)
	}

	// create thumbnail animation
	win.thmb.Create(cartload, win.img.dbg.VCS().TV.GetCreationSpecID(), thumbnailer.UndefinedNumFrames)

	// defer boxart lookup to when we receive the property
}

func (win *winSelectROM) findBoxart() error {
	// reset boxartUse flag until we are certain we've loaded a suitable image
	win.boxartUse = false

	// fuzzy find a candidate image
	n, _, _ := strings.Cut(win.selectedProperties.Name, "(")
	n = strings.TrimSpace(n)
	m := fuzzy.Find(n, win.boxart)
	if len(m) == 0 {
		return nil
	}

	// load image
	p, err := resources.JoinPath(namedBoxarts, m[0].Str)
	if err != nil {
		return fmt.Errorf("boxart: %w", err)
	}

	d, err := os.ReadFile(p)
	if err != nil {
		return fmt.Errorf("boxart: %w", err)
	}

	// conversion function
	render := func(src image.Image) {
		if _, ok := src.(*image.RGBA); ok {
			return
		}
		b := src.Bounds()
		dst := image.NewRGBA(image.Rect(0, 0, b.Dx()/4, b.Dy()/4))
		draw.BiLinear.Scale(dst, dst.Bounds(), src, b, draw.Src, nil)
		sz := dst.Bounds().Max
		win.boxartDim = imgui.Vec2{X: float32(sz.X), Y: float32(sz.Y)}

		// rendering the image without first marking the texture for
		// (re)creation causes problems when transitioning from some images to
		// another image
		//
		// unclear what the cause is but a good example of a problem image is
		// the image of "Extra Terrestrials". Image file and MD5 hash:
		//
		//    99761614c5cfac9e72809dfaf87d886  Extra Terrestrials (USA).png
		win.boxartTexture.markForCreation()
		win.boxartTexture.render(dst)

		win.boxartUse = true
	}

	// convert image and render into texture
	ext := filepath.Ext(p)
	switch ext {
	case ".png":
		img, err := png.Decode(bytes.NewReader(d))
		if err != nil {
			return fmt.Errorf("boxart: %w", err)
		}
		render(img)
	default:
		return fmt.Errorf("boxart: unsupported file extension: *%s", ext)
	}

	return nil
}

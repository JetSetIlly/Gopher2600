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
//
// *** NOTE: all historical versions of this file, as found in any
// git repository, are also covered by the licence, even when this
// notice is not present ***

package sdlimgui

type windows struct {
	screen *tvScreen
	cpu    *cpu
	ram    *ram
	tia    *tia
	delays *delays
	riot   *riot
	tv     *tv
}

func newWindows(img *SdlImgui) (*windows, error) {
	win := &windows{}

	var err error

	win.screen, err = newTvScreen(img)
	if err != nil {
		return nil, err
	}
	win.cpu, err = newCPU(img)
	if err != nil {
		return nil, err
	}
	win.ram, err = newRAM(img)
	if err != nil {
		return nil, err
	}
	win.tia, err = newTIA(img)
	if err != nil {
		return nil, err
	}
	win.delays, err = newDelays(img)
	if err != nil {
		return nil, err
	}
	win.riot, err = newRIOT(img)
	if err != nil {
		return nil, err
	}
	win.tv, err = newTV(img)
	if err != nil {
		return nil, err
	}

	return win, nil
}

func (win *windows) destroy() {
	win.screen.destroy()
}

func (win *windows) draw() {
	win.screen.draw()
	win.cpu.draw()
	win.ram.draw()
	win.tia.draw()
	win.delays.draw()
	win.riot.draw()
	win.tv.draw()
}

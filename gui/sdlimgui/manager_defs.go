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

// windowDef specifies a window creator, the menu it appears in whether it
// should open on start.
type windowDef struct {
	// the window creation function
	create func(*SdlImgui) (window, error)

	// menu entry the window will be associated with. the label and windowID
	// fields can be left blank. if it is the window.id() value will be used.
	//
	// the restrictBus and restrictMapper fields are optional.
	menu menuEntry

	// whether the window should be opened in an defaultOpen or closed state. this is
	// the default state and will be overridden if loadManagerState() is called
	// and a previously saved state can be found
	defaultOpen bool
}

// list of windows to add to the window manager.
var windowDefs = [...]windowDef{
	// windows called from "debugger" menu
	{create: newSelectROM, menu: menuEntry{group: menuDebugger}},
	{create: newWinPrefs, menu: menuEntry{group: menuDebugger}},
	{create: newWinTerm, menu: menuEntry{group: menuDebugger}},
	{create: newWinLog, menu: menuEntry{group: menuDebugger}},

	// windows that appear in the "vcs" menu
	{create: newWinCollisions, menu: menuEntry{group: menuVCS}},
	{create: newWinControl, menu: menuEntry{group: menuVCS}, defaultOpen: true},
	{create: newWinCPU, menu: menuEntry{group: menuVCS}, defaultOpen: true},
	{create: newWinDisasm, menu: menuEntry{group: menuVCS}, defaultOpen: true},
	{create: newWinDbgScr, menu: menuEntry{group: menuVCS}, defaultOpen: true},
	{create: newWinRAM, menu: menuEntry{group: menuVCS}, defaultOpen: true},
	{create: newWinPeripherals, menu: menuEntry{group: menuVCS}},
	{create: newWinPorts, menu: menuEntry{group: menuVCS}},
	{create: newWinTIA, menu: menuEntry{group: menuVCS, label: winTIAMenu}, defaultOpen: true},
	{create: newWinTIAAudio, menu: menuEntry{group: menuVCS, label: winTIAAudioMenu}},
	{create: newWinTimer, menu: menuEntry{group: menuVCS}, defaultOpen: true},

	// windows that appear in the "tools" menu
	{create: newWinOscilloscope, menu: menuEntry{group: menuTools}, defaultOpen: true},
	{create: newWinTracker, menu: menuEntry{group: menuTools}, defaultOpen: false},
	{create: newWinTimeline, menu: menuEntry{group: menuTools}, defaultOpen: true},
	{create: newWin6507Pinout, menu: menuEntry{group: menuTools}, defaultOpen: false},
	// {create: newWinPaint, menu: menuEntry{group: menuTools}, defaultOpen: false},

	// windows that appear in cartridge specific menu
	{create: newWinDPCregisters, menu: menuEntry{group: menuCart, restrictBus: menuRestrictRegister, restrictMapper: []string{"DPC"}}},
	{create: newWinDPCplusRegisters, menu: menuEntry{group: menuCart, restrictBus: menuRestrictRegister, restrictMapper: []string{"DPC+"}}},
	{create: newWinCDFRegisters, menu: menuEntry{group: menuCart, restrictBus: menuRestrictRegister, restrictMapper: []string{"CDF", "CDFJ", "CDF0", "CDF1", "CDFJ+"}}},
	{create: newWinCDFStreams, menu: menuEntry{group: menuCart, restrictBus: menuRestrictRegister, restrictMapper: []string{"CDF", "CDFJ", "CDF0", "CDF1", "CDFJ+"}}},
	{create: newWinSuperchargerRegisters, menu: menuEntry{group: menuCart, restrictBus: menuRestrictRegister, restrictMapper: []string{"AR"}}},
	{create: newWinCartTape, menu: menuEntry{group: menuCart, restrictBus: menuRestrictTape}},
	{create: newWinCartRAM, menu: menuEntry{group: menuCart, restrictBus: menuRestrictRAM}},
	{create: newWinCartStatic, menu: menuEntry{group: menuCart, restrictBus: menuRestrictStatic}},

	// coprocessor windows
	{create: newWinCoProcDisasm, menu: menuEntry{group: menuCoProc, restrictBus: menuRestrictCoProc, label: winCoProcDisasmMenu}},
	{create: newWinCoProcFaults, menu: menuEntry{group: menuCoProc, restrictBus: menuRestrictCoProc, label: winCoProcFaultsMenu}},
	{create: newWinCoProcRegisters, menu: menuEntry{group: menuCoProc, restrictBus: menuRestrictCoProc, label: winCoProcRegistersMenu}},
	{create: newWinCoProcProfiling, menu: menuEntry{group: menuCoProc, restrictBus: menuRestrictCoProc, label: winCoProcProfilingMenu}},
	{create: newWinCoProcFunctions, menu: menuEntry{group: menuCoProc, restrictBus: menuRestrictCoProc, label: winCoProcFunctionsMenu}},
	{create: newWinCoProcGlobals, menu: menuEntry{group: menuCoProc, restrictBus: menuRestrictCoProc, label: winCoProcGlobalsMenu}},
	{create: newWinCoProcLocals, menu: menuEntry{group: menuCoProc, restrictBus: menuRestrictCoProc, label: winCoProcLocalsMenu}},
	{create: newWinCoProcSource, menu: menuEntry{group: menuCoProc, restrictBus: menuRestrictCoProc, label: winCoProcSourceMenu}},

	// plusrom windows
	{create: newWinPlusROMNetwork, menu: menuEntry{group: menuPlusROM, label: winPlusROMNetworkMenu}},
	{create: newWinPlusROMNick, menu: menuEntry{group: menuPlusROM, label: winPlusROMNickMenu}},

	// savekey windows
	{create: newWinSaveKeyActivity, menu: menuEntry{group: menuSaveKey, label: winSaveKeyActivityMenu}},
	{create: newWinSaveKeyEEPROM, menu: menuEntry{group: menuSaveKey, label: winSaveKeyEEPROMMenu}},

	// atarivox windows
	{create: newWinAtarivox, menu: menuEntry{group: menuAtariVox, label: winAtariVoxActivityMenu}},

	// windows that do not have a menu entry (windows for playmode only)
	{create: newWinComparison, menu: menuEntry{group: menuNone}, defaultOpen: false},
	{create: newWinBot, menu: menuEntry{group: menuNone}, defaultOpen: false},
}

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

package prefs

// list of preference values that are no longer used.
var defunct = []string{
	"debugger.randpins",
	"debugger.randstate",
	"hardware.instantARM",
	"sdlimgui.playmode.terminalOnError",
	"crt.inputGamma",
	"crt.outputGamma",
	"crt.maskScanlineScaling",
	"crt.phosphorSpeed",
	"crt.blur",
	"crt.blurLevel",
	"crt.vignette",
	"crt.maskBright",
	"crt.maskBrightness",
	"crt.scanlinesBright",
	"crt.scanlinesBrightness",
	"hardware.arm7.allowMAMfromThumb",
	"hardware.arm7.flashAccessTime",
	"hardware.arm7.flashAccessTime1",
	"hardware.arm7.sramAccessTime",
	"hardware.arm7.defaultMAM",
	"tia.revision.hmove.ripplestart",
	"tia.revision.hmove.rippleend",
	"sdlimgui.playmode.windowsize",
	"sdlimgui.playmode.windowpos",
	"sdlimgui.playmode.windowSize",
	"sdlimgui.playmode.windowPos",
	"sdlimgui.playmode.audioEnabled",
	"sdlimgui.debugger.windowSize",
	"sdlimgui.debugger.windowPos",
	"sdlimgui.debugger.audioEnabled",
	"crt.unsyncTolerance",
	"crt.syncSpeedScanlines",
	"hiscore.authtoken",
	"hiscore.server",
	"hardware.arm7.flashLatency",
	"sdlimgui.display.fastSync",
	"sdlimgui.glswapinterval",
	"hardware.arm7.abortOnIllegalMem",
	"hardware.arm7.abortOnStackCollision",
	"hardware.arm7.extendedMemoryErrorLogging",
	"plusrom.id", // replaced with plusrom.id_v2.1.1
	"crt.noise",
	"crt.noiseLevel",
	"sdlimgui.fonts.gui",
	"sdlimgui.fonts.terminal",
	"sdlimgui.fonts.code",
	"crt.syncPowerOn",
	"crt.syncSpeed",
	"crt.syncSensitivity",
	"sdlimgui.playmode.coprocDevNotification",
	"sdlimgui.playmode.fpsOverlay",
	"crt.brightness",
	"crt.contrast",
	"crt.hue",
	"crt.saturation",
	"sdlimgui.display.frameQueue",
	"sdlimgui.display.frameQueueAuto",
	"crt.vsync.recovery",
	"crt.vsync.sensitivity",
	"crt.integerScaling",
	"television.halt.desynchronised",
	"television.vsync.immediatedesync",
	"crt.maskFine",
	"crt.scanlinesFine",
	"emulation.recentrom",      // replaced with sdlimgui.emulation.recentrom
	"display.color.brightness", // replaced with television.color.brightness
	"display.color.contrast",   // replaced with television.color.contrast
	"display.color.hue",        // replaced with television.color.hue
	"display.color.saturation", // replaced with television.color.saturation
	"crt.bevel",
	"crt.bevelSize",
	"crt.flicker",
	"crt.flickerLevel",
	"crt.ghosting",
	"crt.ghostingAmount",
	"crt.fringing",
	"crt.fringingAmount", // replaced with crt.chromaticAberration
	"crt.enabled",        // replaced with crt.pixelPerfect (inverted setting)
	"crt.whiteLevel",
	"crt.interference",      // replaced with crt.rfInterference
	"crt.interferenceLevel", // replaces with crt.rfNoiseLevel and crt.rfGhostingLevel
	"television.vsync.recovery",
	"television.color.legacy",               // replaced with television.color.legacy.enabled
	"hardware.arm7.MisalignedAccessIsFault", // replaced with hardware.arm7.abortOnMisalignedAccess
	"hardware.arm7.unwrapACE",               // replaced with hardware.unwrapACE
}

// returns true if string is in list of defunct values.
func isDefunct(s string) bool {
	for _, m := range defunct {
		if s == m {
			return true
		}
	}
	return false
}

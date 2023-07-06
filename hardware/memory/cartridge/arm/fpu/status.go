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

package fpu

type FPSCR struct {
	// "A2.5.3 Floating-point Status and Control Register, FPSCR" of "ARMv7-M"
	// Page A2-37
	value uint32
}

func (fpscr *FPSCR) AHP() bool {
	// bit 26
	return fpscr.value&0x04000000 == 0x04000000
}

func (fpscr *FPSCR) SetAHP(set bool) {
	// bit 26
	fpscr.value &= 0xfbffffff
	if set {
		fpscr.value |= 0x04000000
	}
}

func (fpscr *FPSCR) DN() bool {
	// bit 25
	return fpscr.value&0x02000000 == 0x02000000
}

func (fpscr *FPSCR) SetDN(set bool) {
	// bit 25
	fpscr.value &= 0xfdffffff
	if set {
		fpscr.value |= 0x02000000
	}
}

func (fpscr *FPSCR) FZ() bool {
	// bit 24
	return fpscr.value&0x01000000 == 0x01000000
}

func (fpscr *FPSCR) SetFZ(set bool) {
	// bit 24
	fpscr.value &= 0xfeffffff
	if set {
		fpscr.value |= 0x01000000
	}
}

func (fpscr *FPSCR) UFC() bool {
	// bit 3
	return fpscr.value&0x00000008 == 0x00000008
}

func (fpscr *FPSCR) SetUFC(set bool) {
	// bit 3
	fpscr.value &= 0xfffffff7
	if set {
		fpscr.value |= 0x00000008
	}
}

type FPRounding byte

// List of valid rounding methods for FPU
const (
	FPRoundNearest FPRounding = 0b00
	FPRoundPlusInf FPRounding = 0b01
	FPRoundNegInf  FPRounding = 0b10
	FPRoundZero    FPRounding = 0b11
)

func (fpscr *FPSCR) RMode() FPRounding {
	// bits 22-23
	return FPRounding((fpscr.value & 0x00c00000) >> 22)
}

func (fpscr *FPSCR) SetRMode(mode FPRounding) {
	// bits 22-23
	fpscr.value &= 0xff3fffff
	fpscr.value |= uint32(mode) << 22
}

func (fpu *FPU) StandardFPSCRValue() FPSCR {
	// page A2-53 of "ARMv7-M"
	var fpscr FPSCR
	fpscr.SetDN(true)
	fpscr.SetFZ(true)
	fpscr.SetAHP(fpu.Status.AHP())
	return fpscr
}

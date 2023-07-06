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

// see thumb2.go for documentation information

package arm

import "fmt"

func (arm *ARM) decodeThumb2FPU(opcode uint16) *DisasmEntry {
	// "Chapter A6 The Floating-point Instruction Set Encoding" of "ARMv7-M"
	switch arm.state.function32bitOpcodeHi & 0x0e00 {
	case 0x0e00:
		switch opcode & 0x0010 {
		case 0x0000:
			// "A6.4 Floating-point data-processing instructions" of "ARMv7-M"
			return arm.decodeThumb2FPUDataProcessing(opcode)
		case 0x0010:
			// "A6.6 32-bit transfer between Arm core and extension registers" of "ARMv7-M"
			return arm.decodeThumb2FPU32bitTransfer(opcode)
		}
	case 0x0c00:
		switch opcode & 0x00d0 {
		case 0x00d0:
			// "A6.7 64-bit transfers between Arm core and extension registers" of "ARMv7-M"
			return arm.decodeThumb2FPUTransfers(opcode)
		default:
			// "A6.5 Extension register load or store instructions" of "ARMv7-M"
			return arm.decodeThumb2FPURegisterLoadStore(opcode)
		}
	}

	panic("undecoded FPU instruction")
}

func (arm *ARM) decodeThumb2FPUDataProcessing(opcode uint16) *DisasmEntry {
	// "A6.4 Floating-point data-processing instructions" of "ARMv7-M"

	T := arm.state.function32bitOpcodeHi&0x1000 == 0x1000
	opc1 := (arm.state.function32bitOpcodeHi & 0x00f0) >> 4
	opc2 := arm.state.function32bitOpcodeHi & 0x000f
	sz := opcode&0x0100 == 0x0100
	opc3 := (opcode & 0x00c0) >> 6
	opc4 := opcode & 0x000f

	_ = opc4

	if !T {
		switch opc1 & 0b1011 {
		case 0b0011:
			D := (arm.state.function32bitOpcodeHi & 0x40) >> 6
			Vn := arm.state.function32bitOpcodeHi & 0x000f
			Vd := (opcode & 0xf000) >> 12
			sz := (opcode & 0x0100) == 0x0100
			N := (opcode & 0x0080) >> 7
			M := (opcode & 0x0020) >> 5
			Vm := opcode & 0x000f

			var d uint16
			var n uint16
			var m uint16

			if sz {
				d = (D << 4) | Vd
				n = (N << 4) | Vn
				m = (M << 4) | Vm
				panic("double precision VADD")
			} else {
				d = (Vd << 1) | D
				n = (Vn << 1) | N
				m = (Vm << 1) | M
			}

			switch opc3 & 0b01 {
			case 0b00:
				// "A7.7.225 VADD" of "ARMv7-M"
				if arm.decodeOnly {
					return &DisasmEntry{
						Is32bit:  true,
						Operator: "VADD",
					}
				}

				arm.state.fpu.Registers[d] = uint32(arm.state.fpu.FPAdd(
					uint64(arm.state.fpu.Registers[n]), uint64(arm.state.fpu.Registers[m]),
					32, true))
				return nil

			case 0b01:
				// "A7.7.260 VSUB" of "ARMv7-M"
				if arm.decodeOnly {
					return &DisasmEntry{
						Is32bit:  true,
						Operator: "VSUB",
					}
				}

				arm.state.fpu.Registers[d] = uint32(arm.state.fpu.FPSub(
					uint64(arm.state.fpu.Registers[n]), uint64(arm.state.fpu.Registers[m]),
					32, true))
				return nil
			}

		case 0b1000:
			// "A7.7.232 VDIV" of "ARMv7-M"
			if arm.decodeOnly {
				return &DisasmEntry{
					Is32bit:  true,
					Operator: "VDIV",
				}
			}

			D := (arm.state.function32bitOpcodeHi & 0x40) >> 6
			Vn := arm.state.function32bitOpcodeHi & 0x000f
			Vd := (opcode & 0xf000) >> 12
			sz := (opcode & 0x0100) == 0x0100
			N := (opcode & 0x0080) >> 7
			M := (opcode & 0x0020) >> 5
			Vm := opcode & 0x000f

			if sz {
				// d := (D << 4) | Vd
				// n := (N << 4) | Vn
				// m := (M << 4) | Vm
				panic("double precision VDIV")
			} else {
				d := (Vd << 1) | D
				n := (Vn << 1) | N
				m := (Vm << 1) | M
				arm.state.fpu.Registers[d] = uint32(arm.state.fpu.FPDiv(
					uint64(arm.state.fpu.Registers[n]), uint64(arm.state.fpu.Registers[m]),
					32, true))
				return nil
			}

		case 0b1011:
			if opc2&0b1000 == 0b1000 {
				if opc3&0b01 == 0b01 {
					// "A7.7.228 VCVT, VCVTR (between floating-point and integer)" of "ARMv7-M"
					if arm.decodeOnly {
						return &DisasmEntry{
							Is32bit:  true,
							Operator: "VCVT",
						}
					}

					D := (arm.state.function32bitOpcodeHi & 0x40) >> 6
					Vd := (opcode & 0xf000) >> 12
					op := opcode&0x0080 == 0x0080
					M := (opcode & 0x0020) >> 5
					Vm := opcode & 0x000f

					toInteger := opc2&0b0001 == 0b0001
					if toInteger {
						panic("unimplemented VCVT (to integer)")
					} else {
						if sz {
							panic("double precision VCVT")
						} else {
							d := Vd<<1 | D
							m := Vm<<1 | M
							arm.state.fpu.Registers[d] = uint32(arm.state.fpu.FixedToFP(uint64(arm.state.fpu.Registers[m]),
								0, op, false, 32, true))
							return nil
						}
					}
				}
			}
		}
	}

	panic(fmt.Sprintf("undecoded FPU instrucion (A6.4): %04x %04x", arm.state.function32bitOpcodeHi, opcode))
}

func (arm *ARM) decodeThumb2FPU32bitTransfer(opcode uint16) *DisasmEntry {
	// "A6.6 32-bit transfer between ARM core and extension registers" of "ARMv7-M"

	T := arm.state.function32bitOpcodeHi&0x1000 == 0x1000
	L := arm.state.function32bitOpcodeHi&0x0010 == 0x0100
	C := opcode&0x0100 == 0x0100

	if T {
		panic("undefined 32bit FPU transfer")
	}

	A := (arm.state.function32bitOpcodeHi & 0x00e0) >> 5

	if L {
		if A == 0b111 {
			panic("unimplemented VMRS")
		} else {
			if C {
				panic("VMOV (L && C)")
			} else {
				panic("VMOV (L && !C)")
			}
		}
	} else {
		if A == 0b111 {
			panic("unimplemented VMSR")
		} else {
			if C {
				panic("VMOV (!L && V)")
			} else {
				// "A7.7.243 VMOV (between Arm core register and single-precision register)" of "ARMv7-M"
				toArmRegister := arm.state.function32bitOpcodeHi&0x0010 == 0x0010
				Vn := arm.state.function32bitOpcodeHi & 0x000f
				Rarm := (opcode & 0xf000) >> 12
				n := (opcode & 0x0080) >> 7
				Rfpu := (Vn << 1) | n

				if toArmRegister {
					arm.state.registers[Rarm] = arm.state.fpu.Registers[Rfpu]
				} else {
					arm.state.fpu.Registers[Rfpu] = arm.state.registers[Rarm]
				}
			}
		}
	}

	return nil
}

func (arm *ARM) decodeThumb2FPURegisterLoadStore(opcode uint16) *DisasmEntry {
	// "A6.5 Extension register load or store instructions" of "ARMv7-M"

	op := (arm.state.function32bitOpcodeHi & 0x01f0) >> 4
	Rn := arm.state.function32bitOpcodeHi & 0x000f

	switch op & 0b11011 {
	case 0b10000:
		fallthrough
	case 0b11000:
		// "A7.7.259 VSTR" of "ARMv7-M"
		if arm.decodeOnly {
			return &DisasmEntry{
				Is32bit:  true,
				Operator: "VSTR",
			}
		}

		add := arm.state.function32bitOpcodeHi&0x0080 == 0x0080
		D := (arm.state.function32bitOpcodeHi & 0x0040) >> 6
		Rn := arm.state.function32bitOpcodeHi & 0x000f
		Vd := (opcode & 0xf000) >> 12
		imm8 := opcode & 0x00ff
		imm32 := uint32(imm8 << 2)

		addr := arm.state.registers[Rn]
		if add {
			addr += imm32
		} else {
			addr -= imm32
		}

		if opcode&0x0100 == 0x0100 {
			// 64bit floats (T1 encoding)
			d := (D << 4) | Vd
			arm.write32bit(addr, arm.state.fpu.Registers[d], true)
			addr += 4
			arm.write32bit(addr, arm.state.fpu.Registers[d+1], true)
		} else {
			// 32bit floats (T1 encoding)
			d := (Vd << 1) | D
			arm.write32bit(addr, arm.state.fpu.Registers[d], true)
		}

	case 0b10010:
		switch Rn {
		case 0b1101:
			// "A7.7.252 VPUSH" of "ARMv7-M"
			if arm.decodeOnly {
				return &DisasmEntry{
					Is32bit:  true,
					Operator: "VPUSH",
				}
			}

			D := (arm.state.function32bitOpcodeHi & 0x0040) >> 6
			Vd := (opcode & 0xf000) >> 12
			imm8 := opcode & 0x00ff
			imm32 := uint32(imm8 << 2)

			// extent of stack
			addr := arm.state.registers[rSP] - imm32

			if opcode&0x0100 == 0x0100 {
				// 64bit floats (T1 encoding)
				d := (D << 4) | Vd
				for i := uint16(0); i < imm8; i += 2 {
					arm.write32bit(addr, arm.state.fpu.Registers[d+i], true)
					addr += 4
					arm.write32bit(addr, arm.state.fpu.Registers[d+i+1], true)
					addr += 4
				}
			} else {
				// 32bit floats (T2 encoding)
				d := (Vd << 1) | D
				for i := uint16(0); i < imm8; i++ {
					arm.write32bit(addr, arm.state.fpu.Registers[d+i], true)
					addr += 4
				}
			}

			arm.state.registers[rSP] -= uint32(imm32)

		default:
			panic("unimplemented FPU register load/save instruction (VSTM)")
		}

	case 0b10001:
		fallthrough
	case 0b11001:
		// "A7.7.236 VLDR" of "ARMv7-M"
		if arm.decodeOnly {
			return &DisasmEntry{
				Is32bit:  true,
				Operator: "VLDR",
			}
		}

		add := arm.state.function32bitOpcodeHi&0x0080 == 0x0080
		imm8 := opcode & 0x00ff
		imm32 := uint32(imm8 << 2)
		Vd := (opcode & 0xf000) >> 12
		D := (arm.state.function32bitOpcodeHi & 0x0040) >> 6
		Rn := arm.state.function32bitOpcodeHi & 0x000f

		addr := arm.state.registers[Rn]
		if Rn == rPC {
			addr = align(arm.state.registers[rPC], 4)
		}
		if add {
			addr += imm32
		} else {
			addr -= imm32
		}

		if opcode&0x0100 == 0x0100 {
			// 64bit floats (T1 encoding)
			d := (D << 4) | Vd
			arm.state.fpu.Registers[d] = arm.read32bit(addr, true)
			addr += 4
			arm.state.fpu.Registers[d+1] = arm.read32bit(addr, true)
		} else {
			// 32bit floats (T2 encoding)
			d := (Vd << 1) | D
			arm.state.fpu.Registers[d] = arm.read32bit(addr, true)
		}

	case 0b01001:
		fallthrough
	case 0b01011:
		if Rn == 0b1101 && op&0b11011 == 0b01011 {
			panic("unimplemented FPU register load/save instruction (VPOP)")
		}

		// "A7.7.235 VLDM" of "ARMv7-M"
		if arm.decodeOnly {
			return &DisasmEntry{
				Is32bit:  true,
				Operator: "VLDM",
			}
		}

		add := (arm.state.function32bitOpcodeHi & 0x0080) == 0x0080
		D := (arm.state.function32bitOpcodeHi & 0x0040) >> 6
		wback := (arm.state.function32bitOpcodeHi & 0x0020) == 0x0020

		Vd := (opcode & 0xf000) >> 12
		imm8 := opcode & 0x00ff
		imm32 := imm8 << 2

		addr := arm.state.registers[Rn]
		if add {
			addr += uint32(imm32)
		} else {
			addr -= uint32(imm32)
		}

		if wback {
			if add {
				arm.state.registers[Rn] += uint32(imm32)
			} else {
				arm.state.registers[Rn] -= uint32(imm32)
			}
		}

		if opcode&0x0100 == 0x0100 {
			// 64bit floats (T1 encoding)
			d := (D << 4) | Vd
			for i := uint16(0); i < imm8; i += 2 {
				arm.state.fpu.Registers[d+i] = arm.read32bit(addr, true)
				addr += 4
				arm.state.fpu.Registers[d+i+1] = arm.read32bit(addr, true)
				addr += 4
			}
		} else {
			// 32bit floats (T2 encoding)
			d := (Vd << 1) | D
			for i := uint16(0); i < imm8; i++ {
				arm.state.fpu.Registers[d+i] = arm.read32bit(addr, true)
				addr += 4
			}
		}
	case 0b01010:
		// "A7.7.258 VSTM" of "ARMv7-M"
		if arm.decodeOnly {
			return &DisasmEntry{
				Is32bit:  true,
				Operator: "VSTM",
			}
		}

		add := arm.state.function32bitOpcodeHi&0x0080 == 0x0080
		imm8 := opcode & 0x00ff
		imm32 := uint32(imm8 << 2)
		Vd := (opcode & 0xf000) >> 12
		D := (arm.state.function32bitOpcodeHi & 0x0040) >> 6
		Rn := arm.state.function32bitOpcodeHi & 0x000f

		addr := arm.state.registers[Rn]
		if add {
			addr += imm32
		} else {
			addr -= imm32
		}

		if opcode&0x0100 == 0x0100 {
			// 64bit floats (T1 encoding)
			d := (D << 4) | Vd
			arm.write32bit(addr, arm.state.fpu.Registers[d], true)
			addr += 4
			arm.write32bit(addr, arm.state.fpu.Registers[d+1], true)
		} else {
			// 32bit floats (T2 encoding)
			d := (Vd << 1) | D
			arm.write32bit(addr, arm.state.fpu.Registers[d], true)
		}
	default:
		panic("unimplemented FPU register load/save instruction")
	}

	return nil
}

func (arm *ARM) decodeThumb2FPUTransfers(opcode uint16) *DisasmEntry {
	// "A6.7 64-bit transfers between Arm core and extension registers" of "ARMv7-M"
	panic("undecoded FPU instrucion (A6.7)")
}

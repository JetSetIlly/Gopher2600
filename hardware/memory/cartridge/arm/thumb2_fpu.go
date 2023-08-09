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

import (
	"fmt"

	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/arm/fpu"
)

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
		switch arm.state.function32bitOpcodeHi & 0x01e0 {
		case 0x0040:
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
		case 0b0010:
			if opc3&0b01 == 0b001 {
				op := (opcode & 0x0040) == 0x0040

				var typ fpu.VFPNegMul

				// "A7.7.250 VNMLA, VNMLS, VNMUL" of "ARMv7-M"
				switch arm.state.function32bitOpcodeHi & 0x0030 {
				case 0x0010:
					if op {
						if arm.decodeOnly {
							return &DisasmEntry{
								Is32bit:  true,
								Operator: "VNMLA",
							}
						}
						typ = fpu.VFPNegMul_VNMLA
					} else {
						if arm.decodeOnly {
							return &DisasmEntry{
								Is32bit:  true,
								Operator: "VNMLS",
							}
						}
						typ = fpu.VFPNegMul_VNMLS
					}
				case 0x0020:
					if op {
						if arm.decodeOnly {
							return &DisasmEntry{
								Is32bit:  true,
								Operator: "VNMUL",
							}
						}
						typ = fpu.VFPNegMul_VNMNUL
					} else {
						panic("illegal instruction in VNMLA, VNMLS, VNMUL group")
					}
				default:
					panic("illegal instruction in VNMLA, VNMLS, VNMUL group")
				}

				D := (arm.state.function32bitOpcodeHi & 0x40) >> 6
				Vn := arm.state.function32bitOpcodeHi & 0x000f
				Vd := (opcode & 0xf000) >> 12
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
					panic("double precision VNMLA, VNMLS, VNMUL")
				} else {
					d = (Vd << 1) | D
					n = (Vn << 1) | N
					m = (Vm << 1) | M

					prod := arm.state.fpu.FPMul(uint64(arm.state.fpu.Registers[n]), uint64(arm.state.fpu.Registers[m]), 32, true)
					switch typ {
					case fpu.VFPNegMul_VNMLA:
						negProd := arm.state.fpu.FPNeg(prod, 32)
						negReg := arm.state.fpu.FPNeg(uint64(arm.state.fpu.Registers[d]), 32)
						arm.state.fpu.Registers[d] = uint32(arm.state.fpu.FPAdd(negReg, negProd, 32, true))
					case fpu.VFPNegMul_VNMLS:
						negReg := arm.state.fpu.FPNeg(uint64(arm.state.fpu.Registers[d]), 32)
						arm.state.fpu.Registers[d] = uint32(arm.state.fpu.FPAdd(negReg, prod, 32, true))
					case fpu.VFPNegMul_VNMNUL:
						negProd := arm.state.fpu.FPNeg(prod, 32)
						arm.state.fpu.Registers[d] = uint32(negProd)
					}
				}
				return nil

			} else {
				// "A7.7.248 VMUL" of "ARMv7-M"

				D := (arm.state.function32bitOpcodeHi & 0x40) >> 6
				Vn := arm.state.function32bitOpcodeHi & 0x000f
				Vd := (opcode & 0xf000) >> 12
				N := (opcode & 0x0080) >> 7
				M := (opcode & 0x0020) >> 5
				Vm := opcode & 0x000f

				var d uint16
				var n uint16
				var m uint16
				var bits int
				var regPrefix rune

				if sz {
					d = (D << 4) | Vd
					n = (N << 4) | Vn
					m = (M << 4) | Vm
					panic("double precision VMUL")
				} else {
					d = (Vd << 1) | D
					n = (Vn << 1) | N
					m = (Vm << 1) | M
					bits = 32
					regPrefix = 'S'
				}

				if arm.decodeOnly {
					return &DisasmEntry{
						Is32bit:  true,
						Operator: "VMUL",
						Operand:  fmt.Sprintf("%c%d, %c%d, %c%d", regPrefix, d, regPrefix, n, regPrefix, m),
					}
				}

				arm.state.fpu.Registers[d] = uint32(arm.state.fpu.FPMul(
					uint64(arm.state.fpu.Registers[n]), uint64(arm.state.fpu.Registers[m]),
					bits, true))
				return nil
			}
		case 0b0011:
			D := (arm.state.function32bitOpcodeHi & 0x40) >> 6
			Vn := arm.state.function32bitOpcodeHi & 0x000f
			Vd := (opcode & 0xf000) >> 12
			N := (opcode & 0x0080) >> 7
			M := (opcode & 0x0020) >> 5
			Vm := opcode & 0x000f

			var d uint16
			var n uint16
			var m uint16
			var bits int
			var regPrefix rune

			if sz {
				d = (D << 4) | Vd
				n = (N << 4) | Vn
				m = (M << 4) | Vm
				bits = 64
				panic("double precision VADD/VSUB")
			} else {
				d = (Vd << 1) | D
				n = (Vn << 1) | N
				m = (Vm << 1) | M
				bits = 32
				regPrefix = 'S'
			}

			switch opc3 & 0b01 {
			case 0b00:
				// "A7.7.225 VADD" of "ARMv7-M"
				if arm.decodeOnly {
					return &DisasmEntry{
						Is32bit:  true,
						Operator: "VADD",
						Operand:  fmt.Sprintf("%c%d, %c%d, %c%d", regPrefix, d, regPrefix, n, regPrefix, m),
					}
				}
				arm.state.fpu.Registers[d] = uint32(arm.state.fpu.FPAdd(
					uint64(arm.state.fpu.Registers[n]), uint64(arm.state.fpu.Registers[m]), bits, true))
				return nil

			case 0b01:
				// "A7.7.260 VSUB" of "ARMv7-M"
				if arm.decodeOnly {
					return &DisasmEntry{
						Is32bit:  true,
						Operator: "VSUB",
						Operand:  fmt.Sprintf("%c%d, %c%d, %c%d", regPrefix, d, regPrefix, n, regPrefix, m),
					}
				}
				arm.state.fpu.Registers[d] = uint32(arm.state.fpu.FPSub(
					uint64(arm.state.fpu.Registers[n]), uint64(arm.state.fpu.Registers[m]), bits, true))
				return nil
			}

		case 0b1000:
			// "A7.7.232 VDIV" of "ARMv7-M"
			D := (arm.state.function32bitOpcodeHi & 0x40) >> 6
			Vn := arm.state.function32bitOpcodeHi & 0x000f
			Vd := (opcode & 0xf000) >> 12
			N := (opcode & 0x0080) >> 7
			M := (opcode & 0x0020) >> 5
			Vm := opcode & 0x000f

			var d uint16
			var n uint16
			var m uint16
			var bits int
			var regPrefix rune

			if sz {
				d = (D << 4) | Vd
				n = (N << 4) | Vn
				m = (M << 4) | Vm
				panic("double precision VDIV")
			} else {
				d = (Vd << 1) | D
				n = (Vn << 1) | N
				m = (Vm << 1) | M
				regPrefix = 'S'
				bits = 32
			}

			if arm.decodeOnly {
				return &DisasmEntry{
					Is32bit:  true,
					Operator: "VDIV",
					Operand:  fmt.Sprintf("%c%d, %c%d, %c%d", regPrefix, d, regPrefix, n, regPrefix, m),
				}
			}

			arm.state.fpu.Registers[d] = uint32(arm.state.fpu.FPDiv(
				uint64(arm.state.fpu.Registers[n]), uint64(arm.state.fpu.Registers[m]),
				bits, true))
			return nil

		case 0b1011:
			if opc2&0b1000 == 0b1000 {
				if opc3&0b01 == 0b01 {
					// "A7.7.228 VCVT, VCVTR (between floating-point and integer)" of "ARMv7-M"
					toInteger := opc2&0b100 == 0b100

					D := (arm.state.function32bitOpcodeHi & 0x40) >> 6
					Vd := (opcode & 0xf000) >> 12
					op := opcode&0x0080 == 0x0080
					M := (opcode & 0x0020) >> 5
					Vm := opcode & 0x000f

					if toInteger {
						unsigned := opc2&0x001 != 0x001

						roundZero := op
						d := Vd<<1 | D
						if sz {
							// m := (M << 4) | Vm
							panic("double precision VCVT (to integer)")
						} else {
							m := Vm<<1 | M

							if arm.decodeOnly {
								e := &DisasmEntry{
									Is32bit:  true,
									Operator: "VCVT",
									Operand:  fmt.Sprintf("S%d, S%d", d, m),
								}
								if unsigned {
									e.Operator = fmt.Sprintf("%s.u32.f32", e.Operator)
								} else {
									e.Operator = fmt.Sprintf("%s.s32.f32", e.Operator)
								}
								return e
							}

							arm.state.fpu.Registers[d] = uint32(arm.state.fpu.FPToFixed(uint64(arm.state.fpu.Registers[m]),
								32, 0, unsigned, roundZero, true))
							return nil
						}
					} else {
						unsigned := !op
						m := Vm<<1 | M

						if sz {
							// d := (D << 4) | Vd
							panic("double precision VCVT")
						} else {
							d := Vd<<1 | D

							if arm.decodeOnly {
								e := &DisasmEntry{
									Is32bit:  true,
									Operator: "VCVT",
									Operand:  fmt.Sprintf("S%d, S%d", d, m),
								}
								if unsigned {
									e.Operator = fmt.Sprintf("%s.f32.u32", e.Operator)
								} else {
									e.Operator = fmt.Sprintf("%s.f32.s32", e.Operator)
								}
								return e
							}

							arm.state.fpu.Registers[d] = uint32(arm.state.fpu.FixedToFP(uint64(arm.state.fpu.Registers[m]),
								32, 0, unsigned, false, true))
							return nil
						}
					}
				}
			} else if opc2&0b0100 == 0b0100 {
				// "A7.7.226 VCMP, VCMPE" of "ARMv7-M"
				if arm.decodeOnly {
					return &DisasmEntry{
						Is32bit:  true,
						Operator: "VCMP",
					}
				}

				D := (arm.state.function32bitOpcodeHi & 0x40) >> 6
				Vd := (opcode & 0xf000) >> 12
				E := opcode&0x0080 == 0x0080
				M := (opcode & 0x0020) >> 5
				Vm := opcode & 0x000f

				var d uint16
				var m uint16
				var N int

				if sz {
					// d := (D << 4) | Vd
					// m := (M << 4) | Vm
					panic("defined double precision VCMP, VCMPE")
				} else {
					d = (Vd << 1) | D
					m = (Vm << 1) | M
					N = 32
				}

				if M == 0b01 {
					// Encoding T1 (with m register)
					arm.state.fpu.FPCompare(uint64(arm.state.fpu.Registers[d]), uint64(arm.state.fpu.Registers[m]), N, E, true)
				} else {
					// Encoding T2 (with zero)
					arm.state.fpu.FPCompare(uint64(arm.state.fpu.Registers[d]), arm.state.fpu.FPZero(false, N), N, E, true)
				}

				return nil
			} else if opc2&0b0100 == 0b0000 {
				if opc3&0b11 == 0b01 {
					// "A7.7.240 VMOV (register)" of "ARMv7-M"
					if arm.decodeOnly {
						return &DisasmEntry{
							Is32bit:  true,
							Operator: "VMOV (register)",
						}
					}

					D := (arm.state.function32bitOpcodeHi & 0x40) >> 6
					Vd := (opcode & 0xf000) >> 12
					M := (opcode & 0x0020) >> 5
					Vm := opcode & 0x000f

					if sz {
						// d := (D << 4) | Vd
						// m := (M << 4) | Vm
						panic("double precision VMOV (register)")
					} else {
						d := (Vd << 1) | D
						m := (Vm << 1) | M
						arm.state.fpu.Registers[d] = arm.state.fpu.Registers[m]
					}

					return nil
				} else {
					// "A7.7.224 VABS" of "ARMv7-M"
					if arm.decodeOnly {
						return &DisasmEntry{
							Is32bit:  true,
							Operator: "VABS",
						}
					}

					D := (arm.state.function32bitOpcodeHi & 0x40) >> 6
					Vd := (opcode & 0xf000) >> 12
					M := (opcode & 0x0020) >> 5
					Vm := opcode & 0x000f

					if sz {
						// d := (D << 4) | Vd
						// m := (M << 4) | Vm
						panic("double precision VABS")
					} else {
						d := (Vd << 1) | D
						m := (Vm << 1) | M
						v := arm.state.fpu.Registers[m]
						arm.state.fpu.Registers[d] = uint32(arm.state.fpu.FPAbs(uint64(v), 32))
					}

					return nil
				}
			}

		case 0b1001:
			// "A7.7.234 VFNMA, VFNMS" of "ARMv7-M"
			op := opcode&0x0080 == 0x0080

			if op {
				if arm.decodeOnly {
					return &DisasmEntry{
						Is32bit:  true,
						Operator: "VFNMS",
					}
				}
			} else {
				if arm.decodeOnly {
					return &DisasmEntry{
						Is32bit:  true,
						Operator: "VFNMA",
					}
				}
			}

			D := (arm.state.function32bitOpcodeHi & 0x40) >> 6
			Vn := arm.state.function32bitOpcodeHi & 0x000f
			Vd := (opcode & 0xf000) >> 12
			N := (opcode & 0x0080) >> 7
			M := (opcode & 0x0020) >> 5
			Vm := opcode & 0x000f

			var d uint16
			var n uint16
			var m uint16
			var bits int

			if sz {
				d = (D << 4) | Vd
				n = (N << 4) | Vn
				m = (M << 4) | Vm
				panic("double precision VFMA/VFMS")
			} else {
				d = (Vd << 1) | D
				n = (Vn << 1) | N
				m = (Vm << 1) | M
				bits = 32
			}

			v := uint64(arm.state.fpu.Registers[n])
			if op {
				v = arm.state.fpu.FPNeg(v, bits)
			}
			arm.state.fpu.Registers[d] = uint32(arm.state.fpu.FPMulAdd(
				arm.state.fpu.FPNeg(uint64(arm.state.fpu.Registers[d]), bits),
				v, uint64(arm.state.fpu.Registers[m]), bits, true))

			return nil

		case 0b1010:
			// "A7.7.233 VFMA, VFMS" of "ARMv7-M"
			op := opcode&0x0080 == 0x0080

			if op {
				if arm.decodeOnly {
					return &DisasmEntry{
						Is32bit:  true,
						Operator: "VFMS",
					}
				}
			} else {
				if arm.decodeOnly {
					return &DisasmEntry{
						Is32bit:  true,
						Operator: "VFMA",
					}
				}
			}

			D := (arm.state.function32bitOpcodeHi & 0x40) >> 6
			Vn := arm.state.function32bitOpcodeHi & 0x000f
			Vd := (opcode & 0xf000) >> 12
			N := (opcode & 0x0080) >> 7
			M := (opcode & 0x0020) >> 5
			Vm := opcode & 0x000f

			var d uint16
			var n uint16
			var m uint16
			var bits int

			if sz {
				d = (D << 4) | Vd
				n = (N << 4) | Vn
				m = (M << 4) | Vm
				panic("double precision VFMA/VFMS")
			} else {
				d = (Vd << 1) | D
				n = (Vn << 1) | N
				m = (Vm << 1) | M
				bits = 32
			}

			v := uint64(arm.state.fpu.Registers[n])
			if op {
				v = arm.state.fpu.FPNeg(v, bits)
			}
			arm.state.fpu.Registers[d] = uint32(arm.state.fpu.FPMulAdd(
				uint64(arm.state.fpu.Registers[d]),
				v, uint64(arm.state.fpu.Registers[m]), bits, true))

			return nil
		}
	}

	panic(fmt.Sprintf("undecoded FPU instrucion (A6.4): %04x %04x", arm.state.function32bitOpcodeHi, opcode))
}

func (arm *ARM) decodeThumb2FPU32bitTransfer(opcode uint16) *DisasmEntry {
	// "A6.6 32-bit transfer between ARM core and extension registers" of "ARMv7-M"

	T := arm.state.function32bitOpcodeHi&0x1000 == 0x1000
	L := arm.state.function32bitOpcodeHi&0x0010 == 0x0010
	C := opcode&0x0100 == 0x0100

	if T {
		panic("undefined 32bit FPU transfer")
	}

	A := (arm.state.function32bitOpcodeHi & 0x00e0) >> 5
	B := (opcode & 0x0060) >> 5

	if L {
		if C {
			if A&0b110 == 0b000 && B == 0b00 {
				// "A7.7.242 VMOV (scalar to Arm core register)" of "ARMv7-M
				panic("VMOV (scalar to Arm core register)")
			} else {
				panic("undefined 32bit FPU transfer")
			}
		} else {
			if A == 0b111 {
				// "A7.7.246 VMRS" of "ARMv7-M"
				if arm.decodeOnly {
					return &DisasmEntry{
						Is32bit:  true,
						Operator: "VMRS",
					}
				}
				Rt := (opcode & 0xf000) >> 12
				if Rt == 15 {
					arm.state.status.negative = arm.state.fpu.Status.N()
					arm.state.status.zero = arm.state.fpu.Status.Z()
					arm.state.status.carry = arm.state.fpu.Status.C()
					arm.state.status.overflow = arm.state.fpu.Status.V()
				} else {
					arm.state.registers[Rt] = arm.state.fpu.Status.Value()
				}
			} else if A == 0b000 {
				// "A7.7.243 VMOV (between Arm core register and single-precision register)" of "ARMv7-M"
				if arm.decodeOnly {
					return &DisasmEntry{
						Is32bit:  true,
						Operator: "VMOV",
						Operand:  "between ARM core and SP register",
					}
				}
				op := arm.state.function32bitOpcodeHi&0x0010 == 0x0010
				Vn := arm.state.function32bitOpcodeHi & 0x000f
				Rt := (opcode & 0xf000) >> 12
				N := (opcode & 0x0080) >> 7
				Rn := (Vn << 1) | N

				if op {
					arm.state.registers[Rt] = arm.state.fpu.Registers[Rn]
				} else {
					arm.state.fpu.Registers[Rn] = arm.state.registers[Rt]
				}
			} else {
				panic("undefined 32bit FPU transfer")
			}
		}
	} else {
		if C {
			// "A7.7.243 VMOV (Arm core register to scalar)" of "ARMv7-M
			panic("VMOV (Arm core register to scalar)")
		} else {
			if A == 0b111 {
				// "A7.7.247 VMSR" of "ARMv7-M"
				panic("unimplemented VMSR")
			} else {
				// "A7.7.243 VMOV (between Arm core register and single-precision register)" of "ARMv7-M"
				if arm.decodeOnly {
					return &DisasmEntry{
						Is32bit:  true,
						Operator: "VMOV",
						Operand:  "between ARM core and SP register",
					}
				}
				op := arm.state.function32bitOpcodeHi&0x0010 == 0x0010
				Vn := arm.state.function32bitOpcodeHi & 0x000f
				Rt := (opcode & 0xf000) >> 12
				N := (opcode & 0x0080) >> 7
				Rn := (Vn << 1) | N

				if op {
					arm.state.registers[Rt] = arm.state.fpu.Registers[Rn]
				} else {
					arm.state.fpu.Registers[Rn] = arm.state.registers[Rt]
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

	maskedOp := op & 0b11011

	if maskedOp == 0b01000 || maskedOp == 0b01010 || (maskedOp == 0b10010 && Rn != 0b1101) {
		// "A7.7.258 VSTM" of "ARMv7-M"
		if arm.decodeOnly {
			return &DisasmEntry{
				Is32bit:  true,
				Operator: "VSTM",
			}
		}

		// P := arm.state.function32bitOpcodeHi&0x0100 == 0x0100
		U := arm.state.function32bitOpcodeHi&0x0080 == 0x0080
		D := (arm.state.function32bitOpcodeHi & 0x0040) >> 6
		W := arm.state.function32bitOpcodeHi&0x0020 == 0x0020
		Vd := (opcode & 0xf000) >> 12

		imm8 := opcode & 0x00ff
		imm32 := uint32(imm8 << 2)

		addr := arm.state.registers[Rn]
		if !U {
			addr -= imm32
		}

		if W {
			if U {
				arm.state.registers[Rn] += imm32
			} else {
				arm.state.registers[Rn] -= imm32
			}
		}

		if opcode&0x0100 == 0x0100 {
			// 64bit floats (T1 encoding)
			// d := (D << 4) | Vd
			panic("double VSTM")
		} else {
			// 32bit floats (T2 encoding)
			d := (Vd << 1) | D

			if imm8 == 0 || imm8+d > 32 {
				panic("too many registers for VSTM")
			}

			for i := uint16(0); i < imm8; i++ {
				arm.write32bit(addr, arm.state.fpu.Registers[d+i], true)
				addr += 4
			}
		}

		return nil
	}

	if maskedOp == 0b10000 || maskedOp == 0b11000 {
		// "A7.7.259 VSTR" of "ARMv7-M"
		if arm.decodeOnly {
			return &DisasmEntry{
				Is32bit:  true,
				Operator: "VSTR",
			}
		}

		U := arm.state.function32bitOpcodeHi&0x0080 == 0x0080
		D := (arm.state.function32bitOpcodeHi & 0x0040) >> 6
		Vd := (opcode & 0xf000) >> 12
		imm8 := opcode & 0x00ff
		imm32 := uint32(imm8 << 2)

		addr := arm.state.registers[Rn]
		if U {
			addr += imm32
		} else {
			addr -= imm32
		}

		if opcode&0x0100 == 0x0100 {
			// 64bit floats (T1 encoding)
			// d := (D << 4) | Vd
			panic("double precision VSTR")
		} else {
			// 32bit floats (T1 encoding)
			d := (Vd << 1) | D
			arm.write32bit(addr, arm.state.fpu.Registers[d], true)
		}

		return nil
	}

	if maskedOp == 0b10010 && Rn == 0b1101 {
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
		arm.state.registers[rSP] -= uint32(imm32)

		if opcode&0x0100 == 0x0100 {
			// 64bit floats (T1 encoding)
			d := (D << 4) | Vd
			for i := uint16(0); i < imm8; i += 2 {
				// TODO: better endian-ness
				arm.write32bit(addr, arm.state.fpu.Registers[d+i], true)
				addr += 4
				arm.write32bit(addr, arm.state.fpu.Registers[d+i+1], true)
				addr += 4
			}
		} else {
			// 32bit floats (T2 encoding)
			d := (Vd << 1) | D

			if imm8 == 0 || imm8+d > 32 {
				panic("too many registers for VPUSH")
			}

			for i := uint16(0); i < imm8; i++ {
				arm.write32bit(addr, arm.state.fpu.Registers[d+i], true)
				addr += 4
			}
		}

		return nil
	}

	if maskedOp == 0b10001 || maskedOp == 0b11001 {
		// "A7.7.236 VLDR" of "ARMv7-M"
		if arm.decodeOnly {
			return &DisasmEntry{
				Is32bit:  true,
				Operator: "VLDR",
			}
		}

		U := arm.state.function32bitOpcodeHi&0x0080 == 0x0080
		D := (arm.state.function32bitOpcodeHi & 0x0040) >> 6
		Vd := (opcode & 0xf000) >> 12
		imm8 := opcode & 0x00ff
		imm32 := uint32(imm8 << 2)

		addr := arm.state.registers[Rn]
		if Rn == rPC {
			addr = alignTo32bits(arm.state.registers[rPC])
		}
		if U {
			addr += imm32
		} else {
			addr -= imm32
		}

		if opcode&0x0100 == 0x0100 {
			panic("double precision VLDR")
		} else {
			// 32bit floats (T2 encoding)
			d := (Vd << 1) | D
			arm.state.fpu.Registers[d] = arm.read32bit(addr, true)
		}

		return nil
	}

	if maskedOp == 0b10011 || (maskedOp == 0b01011 && Rn != 0b1101) {
		// "A7.7.235 VLDM" of "ARMv7-M"
		if arm.decodeOnly {
			return &DisasmEntry{
				Is32bit:  true,
				Operator: "VLDM",
			}
		}

		// P := arm.state.function32bitOpcodeHi&0x0100 == 0x0100
		U := arm.state.function32bitOpcodeHi&0x0080 == 0x0080
		D := (arm.state.function32bitOpcodeHi & 0x0040) >> 6
		W := arm.state.function32bitOpcodeHi&0x0020 == 0x0020
		Vd := (opcode & 0xf000) >> 12

		imm8 := opcode & 0x00ff
		imm32 := uint32(imm8 << 2)

		addr := arm.state.registers[Rn]
		if !U {
			addr -= imm32
		}

		if W {
			if U {
				arm.state.registers[Rn] += imm32
			} else {
				arm.state.registers[Rn] -= imm32
			}
		}

		if opcode&0x0100 == 0x0100 {
			// 64bit floats (T1 encoding)
			// d := (D << 4) | Vd
			panic("double VLDM")
		} else {
			// 32bit floats (T2 encoding)
			d := (Vd << 1) | D

			if imm8 == 0 || imm8+d > 32 {
				panic("too many registers for VLDM")
			}

			for i := uint16(0); i < imm8; i++ {
				arm.state.fpu.Registers[d+i] = arm.read32bit(addr, true)
				addr += 4
			}
		}

		return nil
	}

	panic("unimplemented FPU register load/save instruction")
}

func (arm *ARM) decodeThumb2FPUTransfers(opcode uint16) *DisasmEntry {
	// "A6.7 64-bit transfers between Arm core and extension registers" of "ARMv7-M"
	panic("undecoded FPU instrucion (A6.7)")
}

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

package timer

import (
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/arm/architecture"
)

// the operation of the TIMx units in STM32 ARM packages can be found in the
// STM32 reference manual (referred to as STM32 in comments below this one):
//
// https://www.st.com/resource/en/reference_manual/dm00031020-stm32f405-415-stm32f407-417-stm32f427-437-and-stm32f429-439-advanced-arm-based-32-bit-mcus-stmicroelectronics.pdf

// TIM2 implements the TIM2 timer found in STM32 processors.
type TIM2 struct {
	mmap   architecture.Map
	cycles cycles

	// current register values
	control    uint32
	prescaler  uint32
	autoreload uint32
	counter    uint32
	status     uint32

	// extracted control register flags
	enable              bool   // CEN
	downcounting        bool   // DIR
	updateEventDisabled bool   // UDIS
	updateRequestSource bool   // URS - not a flag but only two options for the "source"
	autoReloadBuffered  bool   // ARPE
	clockDivision       uint32 // CKD

	// the autoreload shadow register is updated from the autoreload register
	// when:
	// 1) the autoreload register is written to AND autoReloadBuffered is false
	// 2) at an update event
	autoreloadShadow uint32

	// prescalarShadow is the prescaler register value that is being used
	// currently. the prescaler register can change but the prescalerCounter
	// will still be ticking towards the prescalarShadow value
	prescalarShadow  uint32
	prescalerCounter uint32
}

func NewTIM2(mmap architecture.Map) *TIM2 {
	return &TIM2{
		mmap: mmap,
		cycles: cycles{
			clkDiv: mmap.ClkDiv,
		},
	}
}

// Reset implementes the Timer interface
func (t *TIM2) Reset() {
	t.setControlRegister(0x00000000)
	t.prescaler = 0x0
	t.autoreload = 0xffffffff
	t.counter = 0.0
	t.cycles.resolve()
}

func (t *TIM2) setControlRegister(val uint32) {
	// control value
	t.control = val

	// "Note that the actual counter enable signal CNT_EN is set 1 clock cycle
	// after CEN." page 591 of STM32
	t.enable = val&0x0001 == 0x0001

	t.updateEventDisabled = val&0x0002 == 0x0002
	t.updateRequestSource = val&0x0004 == 0x0004
	t.downcounting = val&0x0010 == 0x0010
	t.autoReloadBuffered = val&0x0040 == 0x0040

	switch (val & 0x300) >> 8 {
	case 0b00:
		t.clockDivision = 1
	case 0b01:
		t.clockDivision = 2
	case 0b10:
		t.clockDivision = 4
	case 0b11:
		panic("ARM TIM2_CR1: CLK bits of 11 (reserved bit pattern)")
	}

	if val&0x0060 != 0x0000 {
		panic("ARM TIM2_CR1: only CMS bits of 00 (edge-aligned mode) supported")
	}
	if val&0x0008 != 0x0000 {
		panic("ARM TIM2_CR1: only OMP bit of 0 supported")
	}
	if val&0xfc00 != 0x0000 {
		panic("ARM TIM2_CR1: reserved bits are not zero")
	}
}

// Step implementes the Timer interface
func (t *TIM2) Step(cycles float32) {
	if t.cycles.step(cycles) {
		t.Resolve()
	}
}

// Resolve implementes the Timer interface
func (t *TIM2) Resolve() {
	// nothing to do if TIM2 is not enabled
	if !t.enable {
		// just reset the cycles counter
		t.cycles.reset()
		return
	}

	// resolve cycles
	cycles := t.cycles.resolve()

	// adjust for clock division value
	cycles /= t.clockDivision

	// number of counter ticks required
	t.prescalerCounter += cycles

	// adjust prescaler and find number of ticks to accumulate counter by
	var counterTicks uint32
	if t.prescalarShadow > 0 {
		for t.prescalerCounter >= t.prescalarShadow {
			counterTicks++
			t.prescalerCounter -= t.prescalarShadow
		}
	} else {
		counterTicks += t.prescalerCounter
		t.prescalerCounter = 0
	}

	if counterTicks == 0 {
		return
	}

	if t.downcounting {
		c := t.counter - counterTicks

		if c == 0 || c > t.counter {
			// counter underflow
			t.updateEvent()
		} else {
			t.counter = c
		}
	} else {
		c := t.counter + counterTicks

		if c >= t.autoreloadShadow || c < t.counter {
			// counter overflow
			t.updateEvent()
		} else {
			t.counter = c
		}
	}
}

func (t *TIM2) updateEvent() {
	if !t.updateEventDisabled {
		t.prescalarShadow = t.prescaler
		t.autoreloadShadow = t.autoreload

		// set update interupt flag of status register
		t.status |= 0x0001
	}

	// reset of the counters occurs even when updateEventDisable is true. this
	// seems to be the case because at the bottom of page 592 of the "STM32
	// reference" we read:
	//
	// "... no update event occurs until the UDIS bit has been written to 0. However,
	// the counter restarts from 0 ..."
	//
	// it is unclear if this applies to all update events or only to update
	// events generated as a result of timer expiry. until we see contradictory
	// information we wll treat all update events the same
	if t.downcounting {
		t.counter = t.autoreloadShadow
	} else {
		t.counter = 0
	}
	t.prescalerCounter = 0
}

// Read implementes the Timer interface
func (t *TIM2) Read(addr uint32) (uint32, bool) {
	var val uint32

	switch addr {
	case t.mmap.TIM2CR1:
		// TIMx Control register
		val = t.control
	case t.mmap.TIM2CNT:
		// TIMx Counter
		t.Resolve()
		val = t.counter
	default:
		return 0, false
	}

	return val, true
}

// Write implementes the Timer interface
func (t *TIM2) Write(addr uint32, val uint32) bool {
	// "18.4.21 TIMx register map" of "RM0090 reference"
	switch addr {
	case t.mmap.TIM2CR1:
		// TIMx Control
		t.setControlRegister(val)
	case t.mmap.TIM2EGR:
		// TIMx Event Generation
		v := val

		// Bit 0 UG Update Generation
		if v&0x0001 == 0x0001 {
			if !t.updateRequestSource {
				t.updateEvent()
			}
		}
		if v&0x005e != 0x0000 {
			panic("ARM TIM2_EGR: only setting UG bit of this register is supported")
		}
		if val&0xffa0 != 0x0000 {
			panic("ARM TIM2_EGR: reserved bits are not zero")
		}
	case t.mmap.TIM2CNT:
		// TIMx Counter
		// should the cycles field be reset() or should we honour any remaining
		// accumulated cycles? I'm not sure
		t.counter = val
	case t.mmap.TIM2PSC:
		// TIMx Prescalar
		t.prescaler = val & 0x0000ffff
	case t.mmap.TIM2ARR:
		// TIMx Autoload
		t.autoreload = val

		// copy autoreload value to shadow immediately if autoReloadBuffered is false
		if !t.autoReloadBuffered {
			t.autoreloadShadow = t.autoreload
		}
	default:
		return false
	}

	return true
}

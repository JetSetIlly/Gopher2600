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

package ports

import (
	"fmt"
	"strings"

	"github.com/jetsetilly/gopher2600/hardware/instance"
	"github.com/jetsetilly/gopher2600/hardware/memory/chipbus"
	"github.com/jetsetilly/gopher2600/hardware/memory/cpubus"
	"github.com/jetsetilly/gopher2600/hardware/riot/ports/plugging"
)

// Input implements the input/output part of the RIOT (the IO in RIOT).
type Ports struct {
	instance *instance.Instance

	riot chipbus.Memory
	tia  chipbus.Memory

	Panel       Peripheral
	LeftPlayer  Peripheral
	RightPlayer Peripheral

	monitor plugging.PlugMonitor

	// local copies of key chip memory registers

	// the latch bit represents the value of bit 6 of the VBLANK register. used
	// to affect how INPTx registers are written. see WriteINPTx() function
	latch bool

	// the swcha_w field is a copy of the SWCHA register as it was written
	// by the CPU. it is not necessarily the value of SWCHA as written by the
	// RIOT.
	//
	// we need this so that changing the SWACNT (by the CPU) will cause the
	// correct value to be written to be written to the SWCHA register.
	//
	// we can think of these as the input lines that are used in conjunction
	// with the SWACNT bits to create the SWCHA register
	swcha_w uint8

	// swcha_mux is the value that has most recently been written to the SWCHA
	// register by the RIOT
	//
	// the value has *not* been masked by the swacnt value
	//
	// we use it to mux the Player0 and Player 1 nibbles into the single register
	swcha_mux uint8

	// port B equivalents of the above. there is no swchbMux field because only
	// one peripheral uses port B at a time.
	//
	// there is a swchb_raw however. this is the value as written by the
	// peripheral (the panel) before SWBCNT has been applied to it
	swchb_w   uint8
	swchb_raw uint8

	// state of peripheral audio output. applies to peripherals that implement
	// ports.mutePeripheral interface
	peripheralsMuted bool
}

// NewPorts is the preferred method of initialisation of the Ports type.
func NewPorts(instance *instance.Instance, riotMem chipbus.Memory, tiaMem chipbus.Memory) *Ports {
	p := &Ports{
		instance: instance,
		riot:     riotMem,
		tia:      tiaMem,
		latch:    false,
	}
	return p
}

func (p *Ports) End() {
	if p.LeftPlayer != nil {
		p.LeftPlayer.Unplug()
	}
	if p.RightPlayer != nil {
		p.RightPlayer.Unplug()
	}
	if p.Panel != nil {
		p.Panel.Unplug()
	}
}

// Snapshot returns a copy of the RIOT Ports sub-system in its current state.
func (p *Ports) Snapshot() *Ports {
	n := *p
	n.Panel = p.Panel.Snapshot()
	n.LeftPlayer = p.LeftPlayer.Snapshot()
	n.RightPlayer = p.RightPlayer.Snapshot()
	return &n
}

// Plumb new ChipBusses into the Ports sub-system. Depending on context it
// might be advisable for ResetPeripherals() to be called after plumbing has
// succeeded.
func (p *Ports) Plumb(riotMem chipbus.Memory, tiaMem chipbus.Memory) {
	p.riot = riotMem
	p.tia = tiaMem
	if p.Panel != nil {
		p.Panel.Plumb(p)
	}
	if p.LeftPlayer != nil {
		p.LeftPlayer.Plumb(p)
	}
	if p.RightPlayer != nil {
		p.RightPlayer.Plumb(p)
	}
}

// Plug connects a peripheral to a player port.
func (p *Ports) Plug(port plugging.PortID, c NewPeripheral) error {
	periph := c(p.instance, port, p)
	if periph == nil {
		return fmt.Errorf("can't attach peripheral to port (%v)", port)
	}

	// notify monitor of plug event
	if p.monitor != nil {
		p.monitor.Plugged(port, periph.ID())
	}

	// attach any existing monitors to the new player peripheral
	if a, ok := periph.(plugging.Monitorable); ok {
		a.AttachPlugMonitor(p.monitor)
	}

	switch port {
	case plugging.PortPanel:
		if p.Panel != nil {
			p.Panel.Unplug()
		}
		p.Panel = periph
	case plugging.PortLeftPlayer:
		if p.LeftPlayer != nil {
			p.LeftPlayer.Unplug()
		}
		p.LeftPlayer = periph
	case plugging.PortRightPlayer:
		if p.RightPlayer != nil {
			p.RightPlayer.Unplug()
		}
		p.RightPlayer = periph
	default:
		return fmt.Errorf("can't attach peripheral to port (%v)", port)
	}

	periph.Reset()

	// make sure any new audio producing peripherals are aware of the mute state
	p.MutePeripherals(p.peripheralsMuted)

	return nil
}

func (p *Ports) String() string {
	s := strings.Builder{}
	s.WriteString(fmt.Sprintf("SWCHA(W): %#02x ", p.swcha_w))
	s.WriteString(fmt.Sprintf("SWACNT: %#02x ", p.riot.ChipRefer(chipbus.SWACNT)))
	swcha := p.riot.ChipRefer(chipbus.SWCHA)
	s.WriteString(fmt.Sprintf("SWCHA: %#02x ", swcha))
	if swcha != p.deriveSWCHA() {
		s.WriteString("[SWCHA has been poked] ")
	}

	s.WriteString(fmt.Sprintf("SWCHB(W): %#02x ", p.swchb_w))
	s.WriteString(fmt.Sprintf("SWBCNT: %#02x ", p.riot.ChipRefer(chipbus.SWBCNT)))
	swchb := p.riot.ChipRefer(chipbus.SWCHB)
	s.WriteString(fmt.Sprintf("SWCHB: %#02x ", swchb))
	if swchb != p.deriveSWCHB() {
		s.WriteString("[SWCHB has been poked] ")
	}
	return s.String()
}

// mutePeripheral is implemented by peripherals that produce audio independent
// of the emulators sound output. This is useful for implementations that call
// on third-party applications/processes to produce output.
//
// used exclusively by the MutePeripherals() function
type mutePeripheral interface {
	Mute(bool)
}

// MutePeripherals sets the mute state of peripherals that implement the mutePeripheral interface.
func (p *Ports) MutePeripherals(muted bool) {
	if r, ok := p.LeftPlayer.(mutePeripheral); ok {
		r.Mute(muted)
	}
	if r, ok := p.RightPlayer.(mutePeripheral); ok {
		r.Mute(muted)
	}
	p.peripheralsMuted = muted
}

// RestartPeripherals calls restart on any attached peripherals that implement
// that the RestartPeripheral interface.
func (p *Ports) RestartPeripherals() {
	if r, ok := p.LeftPlayer.(RestartPeripheral); ok {
		r.Restart()
	}
	if r, ok := p.RightPlayer.(RestartPeripheral); ok {
		r.Restart()
	}
	if r, ok := p.Panel.(RestartPeripheral); ok {
		r.Restart()
	}
}

// DisabledPeripherals calls restart on any attached peripherals that implement
// that DisablePeripheral interface.
func (p *Ports) DisablePeripherals(disabled bool) {
	if r, ok := p.LeftPlayer.(DisablePeripheral); ok {
		r.Disable(disabled)
	}
	if r, ok := p.RightPlayer.(DisablePeripheral); ok {
		r.Disable(disabled)
	}
	if r, ok := p.Panel.(DisablePeripheral); ok {
		r.Disable(disabled)
	}
}

// ResetPeripherals to an initial state.
func (p *Ports) ResetPeripherals() {
	if p.LeftPlayer != nil {
		p.LeftPlayer.Reset()
	}
	if p.RightPlayer != nil {
		p.RightPlayer.Reset()
	}
	if p.Panel != nil {
		p.Panel.Reset()
	}
}

// Update checks to see if ChipData applies to the Input type and updates the
// internal controller/panel states accordingly. Returns true if ChipData
// requires more attention.
func (p *Ports) Update(data chipbus.ChangedRegister) bool {
	switch data.Register {
	case cpubus.VBLANK:
		p.latch = data.Value&0x40 == 0x40

		// peripheral update
		_ = p.LeftPlayer.Update(data)
		_ = p.RightPlayer.Update(data)

	case cpubus.SWCHA:
		p.swcha_w = data.Value

		// mask value and set SWCHA register. some peripherals may call
		// WriteSWCHx() as part of the Update() function which will write over
		// this value.
		//
		// we should think of this write as the default event in case the
		// peripheral chooses to do nothing with the new value
		swcha := ^(p.riot.ChipRefer(chipbus.SWACNT)) | p.swcha_w
		p.riot.ChipWrite(chipbus.SWCHA, swcha)

		// mask value with SWACNT bits before passing to peripheral
		data.Value &= p.riot.ChipRefer(chipbus.SWACNT)
		_ = p.LeftPlayer.Update(data)
		_ = p.RightPlayer.Update(data)

	case cpubus.SWACNT:
		p.riot.ChipWrite(chipbus.SWACNT, data.Value)

		// peripheral update for SWACNT
		_ = p.LeftPlayer.Update(data)
		_ = p.RightPlayer.Update(data)

		// i/o bits have changed so change the data in the SWCHA register
		swcha := ^(p.riot.ChipRefer(chipbus.SWACNT)) | p.swcha_w
		p.riot.ChipWrite(chipbus.SWCHA, swcha)

		// adjusting SWACNT also affects the SWCHA lines to the peripheral.
		// adjust SWCHA lines and update peripheral with new SWCHA data
		data = chipbus.ChangedRegister{
			Register: cpubus.SWCHA,
			Value:    p.riot.ChipRefer(chipbus.SWCHA),
		}
		_ = p.LeftPlayer.Update(data)
		_ = p.RightPlayer.Update(data)

	case cpubus.SWCHB:
		p.swchb_w = data.Value
		p.riot.ChipWrite(chipbus.SWCHB, p.deriveSWCHB())

	case cpubus.SWBCNT:
		p.riot.ChipWrite(chipbus.SWBCNT, data.Value)
		p.riot.ChipWrite(chipbus.SWCHB, p.deriveSWCHB())
	}

	return false
}

// Step input state forward one cycle.
func (p *Ports) Step() {
	// not much to do here because most input operations happen on demand.
	// recharging of the paddle capacitors however happens (a little bit) every
	// step. also savekey needs to be processed every cycle
	if p.LeftPlayer != nil {
		p.LeftPlayer.Step()
	}
	if p.RightPlayer != nil {
		p.RightPlayer.Step()
	}
	p.Panel.Step()
}

// AttchPlugMonitor implements the plugging.Monitorable interface.
func (p *Ports) AttachPlugMonitor(m plugging.PlugMonitor) {
	p.monitor = m

	// make sure any already attached peripherals know about the new monitor
	if a, ok := p.LeftPlayer.(plugging.Monitorable); ok {
		a.AttachPlugMonitor(m)
	}
	if a, ok := p.RightPlayer.(plugging.Monitorable); ok {
		a.AttachPlugMonitor(m)
	}
	if a, ok := p.Panel.(plugging.Monitorable); ok {
		a.AttachPlugMonitor(m)
	}

	// notify monitor of currently plugged peripherals
	if p.monitor != nil {
		p.monitor.Plugged(plugging.PortLeftPlayer, p.LeftPlayer.ID())
		p.monitor.Plugged(plugging.PortRightPlayer, p.RightPlayer.ID())
	}
}

// PeripheralID returns the ID of the peripheral in the identified port.
func (p *Ports) PeripheralID(id plugging.PortID) plugging.PeripheralID {
	switch id {
	case plugging.PortPanel:
		return p.Panel.ID()
	case plugging.PortLeftPlayer:
		return p.LeftPlayer.ID()
	case plugging.PortRightPlayer:
		return p.RightPlayer.ID()
	}

	return plugging.PeriphNone
}

// WriteSWCHx implements the peripheral.PeripheralBus interface.
func (p *Ports) WriteSWCHx(id plugging.PortID, data uint8) {
	switch id {
	case plugging.PortLeftPlayer:
		data &= 0xf0               // keep only the bits for player 0
		data |= p.swcha_mux & 0x0f // combine with the existing player 1 bits
		p.swcha_mux = data
		p.riot.ChipWrite(chipbus.SWCHA, p.deriveSWCHA())
	case plugging.PortRightPlayer:
		data = (data & 0xf0) >> 4  // move bits into the player 1 nibble
		data |= p.swcha_mux & 0xf0 // combine with the existing player 0 bits
		p.swcha_mux = data
		p.riot.ChipWrite(chipbus.SWCHA, p.deriveSWCHA())
	case plugging.PortPanel:
		p.swchb_raw = data
		p.riot.ChipWrite(chipbus.SWCHB, p.deriveSWCHB())
	default:
		return
	}
}

// WriteINPTx implements the peripheral.PeripheralBus interface.
func (p *Ports) WriteINPTx(inptx chipbus.Register, data uint8) {
	// the VBLANK latch bit only applies to INPT4 and INPT5
	latch := false
	if inptx == chipbus.INPT4 || inptx == chipbus.INPT5 {
		latch = p.latch
	}

	// write memory if button is pressed or it is not and the button latch
	// is false
	if data != 0x80 || !latch {
		p.tia.ChipWrite(inptx, data)
	}
}

// HandleInputEvent forwards the InputEvent to the perupheral in the correct
// port. Returns true if the event was handled and false if not.
func (p *Ports) HandleInputEvent(inp InputEvent) (bool, error) {
	var handled bool
	var err error

	switch inp.Port {
	case plugging.PortPanel:
		handled, err = p.Panel.HandleEvent(inp.Ev, inp.D)
	case plugging.PortLeftPlayer:
		handled, err = p.LeftPlayer.HandleEvent(inp.Ev, inp.D)
	case plugging.PortRightPlayer:
		handled, err = p.RightPlayer.HandleEvent(inp.Ev, inp.D)
	}

	// if error was because of an unhandled event then return without error
	if err != nil {
		return handled, fmt.Errorf("ports: %w", err)
	}

	return handled, nil
}

// PeekField returns the value of the named field.
//
// This is the same as a Peek() on the equivalent memory location in most
// cases, but there are a couple of fields that are not directly associated
// with a memory location.
//
// swacnt, swcha, swacnt and swbcnt are directly as they would be if read by
// Peek()
//
// swcha_w and swchb_w are the swcha and swchb values as most recently written
// by the 6507 program (or with the PokeField() function)
//
// swcha_derived is the value SWCHA should be if the RIOT ports logic hasn't
// been interfered with. swcha_derived and swcha may be unequal because of a
// Poke() or PokeField("swcha").
//
// swchb_derived is the same as swcha_derived except for SWCHB register.
func (p *Ports) PeekField(fld string) interface{} {
	switch fld {
	case "swcha_w":
		return p.swcha_w
	case "swacnt":
		return p.riot.ChipRefer(chipbus.SWACNT)
	case "swcha":
		return p.riot.ChipRefer(chipbus.SWCHA)
	case "swcha_derived":
		return p.deriveSWCHA()

	case "swchb_w":
		return p.swchb_w
	case "swbcnt":
		return p.riot.ChipRefer(chipbus.SWBCNT)
	case "swchb":
		return p.riot.ChipRefer(chipbus.SWCHB)
	case "swchb_derived":
		return p.deriveSWCHB()
	}

	panic(fmt.Sprintf("Ports.PeekField: unknown field: %s", fld))
}

// PokeField sets the named field with a new value.
//
// Fieldnames the same as described for PeekField() except that you cannot
// update the swchb_derived field.
func (p *Ports) PokeField(fld string, v interface{}) {
	switch fld {
	case "swcha_w":
		p.swcha_w = v.(uint8)
		p.riot.ChipWrite(chipbus.SWCHA, p.deriveSWCHA())
	case "swacnt":
		p.riot.ChipWrite(chipbus.SWACNT, v.(uint8))
		p.riot.ChipWrite(chipbus.SWCHA, p.deriveSWCHA())
	case "swcha":
		p.riot.ChipWrite(chipbus.SWCHA, v.(uint8))

	case "swchb_w":
		p.swchb_w = v.(uint8)
		p.riot.ChipWrite(chipbus.SWCHB, p.deriveSWCHB())
	case "swbcnt":
		p.riot.ChipWrite(chipbus.SWBCNT, v.(uint8))
		p.riot.ChipWrite(chipbus.SWCHB, p.deriveSWCHB())
	case "swchb":
		p.riot.ChipWrite(chipbus.SWCHB, v.(uint8))

	default:
		panic(fmt.Sprintf("Ports.PokeField: unknown field: %s", fld))
	}
}

// the derived value of SWCHA. the value it should be if the RIOT logic has
// proceeded normally (ie. no poking)
//
//	SWCHA_W   SWACNT   <input>      SWCHA
//	   0        0         1           1            ^SWCHA_W & ^SWACNT & <input>
//	   0        0         0           0
//	   0        1         1           0
//	   0        1         0           0
//	   1        0         1           1            SWCHA_W & ^SWACNT & <input>
//	   1        0         0           0
//	   1        1         1           1            SWCHA_W & SWACNT & <input>
//	   1        1         0           0
//
//	a := p.swcha_w
//	b := swacnt
//	c := p.swcha_mux
//
//	(^a & ^b & c) | (a & ^b & c) | (a & b & c)
//	(a & c & (^b|b)) | (^a & ^b & c)
//	(a & c) | (^a & ^b & c)
func (p *Ports) deriveSWCHA() uint8 {
	swacnt := p.riot.ChipRefer(chipbus.SWACNT)
	return (p.swcha_w & p.swcha_mux) | (^p.swcha_w & ^swacnt & p.swcha_mux)
}

// the derived value of SWCHB. the value it should be if the RIOT logic has
// proceeded normally (ie. no poking).
//
//	SWCHB_W   SWBCNT   <input>      SWCHB
//	   0        0         1           1            ^SWCHB_W & ^SWBCNT & <input>
//	   0        0         0           0
//	   0        1         1           0
//	   0        1         0           0
//	   1        0         1           1            SWCHB_W & ^SWBCNT & <input>
//	   1        0         0           0
//	   1        1         1           1            SWCHB_W & SWBCNT & <input>
//	   1        1         0           1            SWCHB_W & SWBCNT & ^<input>
//
//	(The last entry of the truth table is different to the truth table for SWCHA)
//
//	a := p.swchb_w
//	b := swbcnt
//	c := p.swchb_raw
//
//	(^a & ^b & c) | (a & ^b & c) | (a & b & c) | (a & b & ^c)
//	(^a & ^b & c) | (a & ^b & c) | (a & b)
//	(^b & c) | (a & b)
func (p *Ports) deriveSWCHB() uint8 {
	swbcnt := p.riot.ChipRefer(chipbus.SWBCNT)
	return (^swbcnt & p.swchb_raw) | (p.swchb_w & swbcnt)
}

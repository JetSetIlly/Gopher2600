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

package harmony

import (
	"fmt"

	"github.com/jetsetilly/gopher2600/curated"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/mapper"
)

// DPCplusStatic implements the bus.CartStatic interface.
type DPCplusStatic struct {
	Arm  []byte
	Data []byte
	Freq []byte
}

// GetStatic implements the bus.CartDebugBus interface.
func (cart dpcPlus) GetStatic() []mapper.CartStatic {
	s := make([]mapper.CartStatic, 3)

	s[0].Label = "ARM"
	s[1].Label = "Data"
	s[2].Label = "Freq"

	s[0].Data = make([]byte, len(cart.static.Arm))
	s[1].Data = make([]byte, len(cart.static.Data))
	s[2].Data = make([]byte, len(cart.static.Freq))

	copy(s[0].Data, cart.static.Arm)
	copy(s[1].Data, cart.static.Data)
	copy(s[2].Data, cart.static.Freq)

	return s
}

// StaticWrite implements the bus.CartDebugBus interface.
func (cart *dpcPlus) PutStatic(label string, addr uint16, data uint8) error {
	switch label {
	case "ARM":
		if int(addr) >= len(cart.static.Arm) {
			return curated.Errorf("dpc+: %v", fmt.Errorf("address too high (%#04x) for %s area", addr, label))
		}
		cart.static.Arm[addr] = data

	case "Data":
		if int(addr) >= len(cart.static.Data) {
			return curated.Errorf("dpc+: %v", fmt.Errorf("address too high (%#04x) for %s area", addr, label))
		}
		cart.static.Data[addr] = data

	case "Freq":
		if int(addr) >= len(cart.static.Freq) {
			return curated.Errorf("dpc+: %v", fmt.Errorf("address too high (%#04x) for %s area", addr, label))
		}
		cart.static.Freq[addr] = data

	default:
		return curated.Errorf("dpc+: %v", fmt.Errorf("unknown static area (%s)", label))
	}

	return nil
}

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

package cartridge

import (
	"fmt"

	"github.com/jetsetilly/gopher2600/errors"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/harmony"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/supercharger"
)

func (cart Cartridge) fingerprint8k(data []byte) func([]byte) (cartMapper, error) {
	if fingerprintTigervision(data) {
		return newTigervision
	}

	if fingerprintParkerBros(data) {
		return newParkerBros
	}

	return newAtari8k
}

func (cart Cartridge) fingerprint16k(data []byte) func([]byte) (cartMapper, error) {
	if fingerprintTigervision(data) {
		return newTigervision
	}

	if fingerprintMnetwork(data) {
		return newMnetwork
	}

	return newAtari16k
}

func (cart Cartridge) fingerprint32k(data []byte) func([]byte) (cartMapper, error) {
	if fingerprintTigervision(data) {
		return newTigervision
	}

	return newAtari32k
}

func (cart *Cartridge) fingerprint(data []byte) error {
	var err error

	// harmony cartridges have a recognisable byte sequence. we can use this
	// regardless of length. any further differentiation can be done in the
	// harmony emulation itself.
	if data[0x20] == 0x1e && data[0x21] == 0xab && data[0x22] == 0xad && data[0x23] == 0x10 {
		// !!TODO: this might be a CFDJ cartridge. check for that.
		cart.mapper, err = harmony.NewDPCplus(data)
		return err
	}

	if supercharger.FingerprintSupercharger(data) {
		cart.mapper, err = supercharger.NewSupercharger(data)
		return err
	}

	if fingerprint3ePlus(data) {
		cart.mapper, err = new3ePlus(data)
		return err
	}

	switch len(data) {
	case 2048:
		cart.mapper, err = newAtari2k(data)
		if err != nil {
			return err
		}

	case 4096:
		cart.mapper, err = newAtari4k(data)
		if err != nil {
			return err
		}

	case 8192:
		cart.mapper, err = cart.fingerprint8k(data)(data)
		if err != nil {
			return err
		}

	case 10240:
		fallthrough

	case 10495:
		cart.mapper, err = newDPC(data)
		if err != nil {
			return err
		}

	case 12288:
		cart.mapper, err = newCBS(data)
		if err != nil {
			return err
		}

	case 16384:
		cart.mapper, err = cart.fingerprint16k(data)(data)
		if err != nil {
			return err
		}

	case 32768:
		cart.mapper, err = cart.fingerprint32k(data)(data)
		if err != nil {
			return err
		}

	case 65536:
		return errors.New(errors.CartridgeError, "65536 bytes not yet supported")

	default:
		return errors.New(errors.CartridgeError, fmt.Sprintf("unrecognised cartridge size (%d bytes)", len(data)))
	}

	// if cartridge mapper implements the optionalSuperChip interface then try
	// to add the additional RAM
	if superchip, ok := cart.mapper.(optionalSuperchip); ok {
		superchip.addSuperchip()
	}

	return nil
}

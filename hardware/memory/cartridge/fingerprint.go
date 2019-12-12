package cartridge

import (
	"fmt"
	"gopher2600/errors"
)

func (cart Cartridge) fingerprint8k(data []byte) func([]byte) (cartMapper, error) {
	if fingerprintTigervision(data) {
		return newTigervision
	}

	if fingerprintParkerBros(data) {
		return newparkerBros
	}

	return newAtari8k
}

func (cart Cartridge) fingerprint16k(data []byte) func([]byte) (cartMapper, error) {
	if fingerprintMnetwork(data) {
		return newMnetwork
	}

	return newAtari16k
}

func (cart *Cartridge) fingerprint(data []byte) error {
	var err error

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
		cart.mapper, err = newAtari32k(data)
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

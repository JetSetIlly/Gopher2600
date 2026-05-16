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

package cartridge

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/jetsetilly/gopher2600/environment"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/mapper"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/mapper/banking"
	"github.com/jetsetilly/gopher2600/logger"
)

const FlatID = "flat"

type Flat struct {
	env *environment.Environment
	mem []uint8
}

func flatMixPattern(env *environment.Environment) ([][4]uint16, error) {
	ext := filepath.Ext(env.Loader.Filename)
	mix := fmt.Sprintf("%s.mix", strings.TrimSuffix(env.Loader.Filename, ext))
	d, err := os.ReadFile(mix)
	if err != nil {
		return nil, err
	}

	lns := strings.Split(strings.ReplaceAll(string(d), "\r", ""), "\n")

	var origins [][4]uint16

	re := regexp.MustCompile(`/add=([0-9A-Fa-f]+):([0-9A-Fa-f]+):([0-9A-Fa-f]+)$`)

	for n, ln := range lns {
		m := re.FindStringSubmatch(ln)
		if m != nil {
			m1, err := strconv.ParseUint(m[1], 16, 16)
			if err != nil {
				return nil, fmt.Errorf("mix: %w", err)
			}
			m2, err := strconv.ParseUint(m[2], 16, 16)
			if err != nil {
				return nil, fmt.Errorf("mix: %w", err)
			}
			m3, err := strconv.ParseUint(m[3], 16, 16)
			if err != nil {
				return nil, fmt.Errorf("mix: %w", err)
			}
			origins = append(origins, [4]uint16{uint16(n), uint16(m1), uint16(m2), uint16(m3)})
		}
	}

	return origins, nil
}

func newFlat(env *environment.Environment) (mapper.CartMapper, error) {
	cart := &Flat{
		env: env,
		mem: make([]uint8, 0x10000),
	}

	data, err := io.ReadAll(env.Loader)
	if err != nil {
		return nil, fmt.Errorf("flat: %w", err)
	}

	// open mix file if possible
	mix, err := flatMixPattern(env)
	if err != nil {
		if len(data) != 16384 {
			return nil, fmt.Errorf("flat: %w", err)
		}

		// if binary is file is exactly 16k then use the default mix pattern
		logger.Logf(env, "flat", "using default mix pattern")
		mix = [][4]uint16{
			{0, 0x5000, 0x5fff, 0x0000},
			{0, 0x7000, 0x7fff, 0x1000},
			{0, 0xd000, 0xdfff, 0x2000},
			{0, 0xf000, 0xffff, 0x3000}}
	}

	// copy data to cartridge memory according to mix pattern
	for _, m := range mix {
		origin := m[1]
		size := m[2] - m[1] + 1
		source := m[3]

		if int(source) >= len(data) {
			return nil, fmt.Errorf("flat: mix: line %d: start of file slice is past the end of the ROM file", m[0])
		}

		if int(source)+int(size) > len(data) {
			return nil, fmt.Errorf("flat: mix: line %d: size of file slice is too large", m[0])
		}

		copy(cart.mem[origin:], data[source:source+size])
	}

	return cart, nil
}

func (cart *Flat) MappedBanks() string {
	return "-"
}
func (cart *Flat) ID() string {
	return FlatID
}

func (cart *Flat) Snapshot() mapper.CartMapper {
	n := *cart
	return &n

}
func (cart *Flat) Plumb(env *environment.Environment) {
	cart.env = env
}

func (cart *Flat) Reset() error {
	return nil
}

func (cart *Flat) Access(addr uint16, peek bool) (data uint8, mask uint8, err error) {
	return cart.mem[addr], mapper.CartDrivenPins, nil
}

func (cart *Flat) AccessVolatile(addr uint16, data uint8, poke bool) error {
	cart.mem[addr] = data
	return nil
}

func (cart *Flat) NumBanks() int {
	return 1
}
func (cart *Flat) GetBank(addr uint16) banking.Information {
	return banking.Information{Number: 0}
}

func (cart *Flat) AccessPassive(addr uint16, data uint8) error {
	return nil
}

func (cart *Flat) Step(clock float32) {
}

func (cart *Flat) CopyBanks() []banking.Content {
	c := make([]banking.Content, 1)
	c[0].Number = 0
	c[0].Data = cart.mem[:]
	c[0].Origins = []uint16{0x0000}
	return c
}

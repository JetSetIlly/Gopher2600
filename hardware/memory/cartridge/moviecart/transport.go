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

package moviecart

// transport is the process of communication between the kernel and the
// microcontroller. for convenience these types have been added to facilitate
// testing and comparison between transportDirection value and transportButtons
// values.

type transportDirection uint8

func (b transportDirection) isUp() bool {
	return b&transportUp == transportUp
}

func (b transportDirection) isDown() bool {
	return b&transportDown == transportDown
}

func (b transportDirection) isLeft() bool {
	return b&transportLeft == transportLeft
}

func (b transportDirection) isRight() bool {
	return b&transportRight == transportRight
}

type transportButtons uint8

func (b transportButtons) isBW() bool {
	return b&transportBW == transportBW
}

func (b transportButtons) isReset() bool {
	return b&transportReset == transportReset
}

func (b transportButtons) isButton() bool {
	return b&transportButton == transportButton
}

// masks for transport input.
const (
	transportRight  transportDirection = 0x10
	transportLeft   transportDirection = 0x08
	transportDown   transportDirection = 0x04
	transportUp     transportDirection = 0x02
	transportBW     transportButtons   = 0x10
	transportSelect transportButtons   = 0x04
	transportReset  transportButtons   = 0x02
	transportButton transportButtons   = 0x01
)

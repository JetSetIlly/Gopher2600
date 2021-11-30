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

package bot

import (
	"fmt"
	"time"

	"github.com/jetsetilly/gopher2600/bots/uci"
	"github.com/jetsetilly/gopher2600/curated"
	"github.com/jetsetilly/gopher2600/hardware"
	"github.com/jetsetilly/gopher2600/hardware/riot/ports"
	"github.com/jetsetilly/gopher2600/hardware/riot/ports/plugging"
)

func VideoChessBot(vcs *hardware.VCS) error {
	uci, err := uci.NewUCI("/usr/local/bin/stockfish")
	if err != nil {
		return curated.Errorf("bot: %v", err)
	}

	uci.Start()

	go func() {
		prevPosition := make([]uint8, 64)
		position := make([]uint8, 64)

		startingPos := [...]uint8{5, 4, 3, 2, 1, 3, 4, 5, 70, 70, 70, 70, 70, 70, 70, 70, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 142, 142, 142, 142, 142, 142, 142, 142, 13, 12, 11, 10, 9, 11, 12, 13}

		started := false
		for !started {
			<-time.After(100 * time.Millisecond)
			copy(position, vcs.Mem.RAM.RAM)

			started = true
			for i := range startingPos {
				if position[i] != startingPos[i] {
					started = false
					break // for loop
				}
			}
			if started {
				fmt.Println("* video chess recognised")
			}
		}

		copy(prevPosition, position)
		uci.SubmitMove <- ""

		cursorCol := 3
		cursorRow := 4

		for {
			// wait fro move from UCI
			move := <-uci.GetMove
			fmt.Printf("* playing move %s\n", move)

			// convert move into row/col numbers
			fromCol := int(move[0]) - 97
			fromRow := int(move[1]) - 48
			toCol := int(move[2]) - 97
			toRow := int(move[3]) - 48

			// move to piece being moved
			moveCol := cursorCol - fromCol
			moveRow := cursorRow - fromRow
			moveCursor(vcs, moveCol, moveRow, true)

			// move piece to new position
			moveCol = fromCol - toCol
			moveRow = fromRow - toRow
			moveCursor(vcs, moveCol, moveRow, false)

			fmt.Println("* vcs is thinking")
			copy(prevPosition, vcs.Mem.RAM.RAM)

			waitForUpdate(vcs)

			foundMove := false
			for !foundMove {
				fromFound := false
				toFound := false

				waiting := true
				for waiting {
					<-time.After(10 * time.Millisecond)
					sl := vcs.TV.GetCoords().Scanline
					if sl > 2 && sl < 200 {
						copy(position, vcs.Mem.RAM.RAM)
						waiting = false
					}
				}

				// check for changed position for black pieces
				for i := range position {
					// 1 king
					// 2 queen
					// 3 bishop
					// 4 knight
					// 5 rook
					// 6 pawn
					piece := position[i] & 0x0f
					prevPiece := prevPosition[i] & 0x0f

					if piece != prevPiece {
						if piece >= 1 && piece <= 6 {
							toCol = i % 8
							toRow = 8 - (i / 8)
							toFound = true
						} else if piece == 8 {
							fromCol = i % 8
							fromRow = 8 - (i / 8)
							fromFound = true
						}
					}
				}

				foundMove = toFound && fromFound
			}

			move = fmt.Sprintf("%c%d%c%d", rune(fromCol+97), fromRow, rune(toCol+97), toRow)
			fmt.Printf("* vcs played %s\n", move)

			uci.SubmitMove <- move

			copy(prevPosition, position)

			cursorCol = fromCol
			cursorRow = fromRow
		}
	}()

	return nil
}

func moveCursor(vcs *hardware.VCS, moveCol int, moveRow int, shortcut bool) {
	waitForUpdate(vcs)

	if moveCol == moveRow || -moveCol == moveRow {
		fmt.Printf("* moving cursor (diagonally) %d %d\n", moveCol, moveRow)

		move := moveCol
		if move < 0 {
			move = -move
		}

		var direction ports.Event

		if moveCol > 0 && moveRow > 0 {
			direction = ports.LeftDown
		} else if moveCol > 0 && moveRow < 0 {
			direction = ports.LeftUp
		} else if moveCol < 0 && moveRow > 0 {
			direction = ports.RightDown
		} else if moveCol < 0 && moveRow < 0 {
			direction = ports.RightUp
		}

		for i := 0; i < move; i++ {
			vcs.ForwardEventToRIOT(plugging.PortLeftPlayer, direction, ports.DataStickTrue)
			waitForFrames(vcs)
			vcs.ForwardEventToRIOT(plugging.PortLeftPlayer, direction, ports.DataStickFalse)
			waitForUpdate(vcs)
		}
	} else {
		fmt.Printf("* moving cursor %d %d\n", moveCol, moveRow)

		if shortcut {
			if moveCol > 4 {
				moveCol -= 8

				// correct moveRow caused by board wrap
				moveRow--
			} else if moveCol < -4 {
				moveCol += 8

				// correct moveRow caused by board wrap
				moveRow++
			}
			if moveRow > 4 {
				moveRow -= 8
			} else if moveRow < -4 {
				moveRow += 8
			}
		}

		if moveCol > 0 {
			for i := 0; i < moveCol; i++ {
				vcs.ForwardEventToRIOT(plugging.PortLeftPlayer, ports.Left, ports.DataStickTrue)
				waitForFrames(vcs)
				vcs.ForwardEventToRIOT(plugging.PortLeftPlayer, ports.Left, ports.DataStickFalse)
				waitForUpdate(vcs)
			}
		} else if moveCol < 0 {
			for i := 0; i > moveCol; i-- {
				vcs.ForwardEventToRIOT(plugging.PortLeftPlayer, ports.Right, ports.DataStickTrue)
				waitForFrames(vcs)
				vcs.ForwardEventToRIOT(plugging.PortLeftPlayer, ports.Right, ports.DataStickFalse)
				waitForUpdate(vcs)
			}
		}

		if moveRow > 0 {
			for i := 0; i < moveRow; i++ {
				vcs.ForwardEventToRIOT(plugging.PortLeftPlayer, ports.Down, ports.DataStickTrue)
				waitForFrames(vcs)
				vcs.ForwardEventToRIOT(plugging.PortLeftPlayer, ports.Down, ports.DataStickFalse)
				waitForUpdate(vcs)
			}
		} else if moveRow < 0 {
			for i := 0; i > moveRow; i-- {
				vcs.ForwardEventToRIOT(plugging.PortLeftPlayer, ports.Up, ports.DataStickTrue)
				waitForFrames(vcs)
				vcs.ForwardEventToRIOT(plugging.PortLeftPlayer, ports.Up, ports.DataStickFalse)
				waitForUpdate(vcs)
			}
		}
	}

	waitForUpdate(vcs)
	vcs.ForwardEventToRIOT(plugging.PortLeftPlayer, ports.Fire, true)
	waitForFrames(vcs)
	vcs.ForwardEventToRIOT(plugging.PortLeftPlayer, ports.Fire, false)
	waitForUpdate(vcs)
}

func waitForFrames(vcs *hardware.VCS) {
	targetFrame := vcs.TV.GetCoords().Frame + 20
	waiting := true
	for waiting {
		<-time.After(10 * time.Millisecond)
		waiting = vcs.TV.GetCoords().Frame < targetFrame
	}
}

func waitForUpdate(vcs *hardware.VCS) {
	waiting := true
	for waiting {
		<-time.After(10 * time.Millisecond)
		v, err := vcs.Mem.RAM.Peek(0xf4)
		if err != nil {
			panic(err)
		}
		waiting = v != 0x00
	}
}

func printBoard(position []uint8) {
	for i := range position {
		if i%8 == 0 {
			fmt.Println("")
		}
		fmt.Printf("%02x ", position[i])
	}
	fmt.Println("")
}

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

package elf

import (
	"fmt"
	"io"

	"github.com/jetsetilly/gopher2600/debugger/terminal/commandline"
)

func newCommands() (*commandline.Commands, error) {
	var template = []string{
		"STREAM (DRAIN|NEXT)",
	}

	commands, err := commandline.ParseCommandTemplate(template)
	if err != nil {
		return nil, err
	}
	return commands, nil
}

// Commands implements the mapper.TerminalCommand interface
func (elf *Elf) Commands() *commandline.Commands {
	return elf.commands
}

// ParseCommand implements the mapper.TerminalCommand interface
func (elf *Elf) ParseCommand(w io.Writer, command string) error {
	tokens := commandline.TokeniseInput(command)
	err := elf.commands.ValidateTokens(tokens)
	if err != nil {
		return err
	}

	tokens.Reset()
	arg, _ := tokens.Get()

	switch arg {
	case "STREAM":
		if !elf.mem.stream.active {
			return fmt.Errorf("ELF streaming is not active")
		}

		arg, ok := tokens.Get()
		if ok {
			switch arg {
			case "DRAIN":
				if elf.mem.stream.drain {
					w.Write([]byte("ELF stream is already draining"))
				} else {
					w.Write([]byte("ELF stream draining started"))
					elf.mem.stream.drain = true
				}
			case "NEXT":
				if elf.mem.stream.drain {
					w.Write([]byte(fmt.Sprintf("%s", elf.mem.stream.stream[elf.mem.stream.drainPtr])))
				} else {
					w.Write([]byte("ELF stream is not currently draining"))
				}
			}
		} else {
			if elf.mem.stream.drain {
				w.Write([]byte(fmt.Sprintf("ELF stream draining: %d remaining",
					elf.mem.stream.drainTop-elf.mem.stream.drainPtr)))
			} else {
				w.Write([]byte(fmt.Sprintf("ELF stream length: %d",
					elf.mem.stream.ptr)))
			}
		}
	}

	return nil
}

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

// Package uci handles a running UCI engine.
package uci

import (
	"fmt"
	"io"
	"os/exec"
	"strings"
	"time"

	"github.com/jetsetilly/gopher2600/bots"
	"github.com/jetsetilly/gopher2600/logger"
)

// UCI handles a running UCI engine, accepts move and offers the best move to
// play next.
type UCI struct {
	stdin  io.WriteCloser
	stdout io.ReadCloser

	// history of moves
	moves []string

	// submitting the empty string as a move is effectively saying to analyse
	// the current position
	SubmitMove chan string

	// once a move has been submitted the best move will be returned on this
	// channel
	GetMove chan string

	// end the UCI program. the UCI should not be restarted after it has quit
	Quit chan bool

	diagnostic chan bots.Diagnostic
}

func (uci *UCI) close() {
	_ = uci.stdin.Close()
	_ = uci.stdout.Close()
}

// NewUCI prepares and launches a new UCI engine.
func NewUCI(pathToEngine string, diagnostic chan bots.Diagnostic) (*UCI, error) {
	uci := &UCI{
		SubmitMove: make(chan string),
		GetMove:    make(chan string),
		Quit:       make(chan bool),
		moves:      make([]string, 0, 100),
		diagnostic: diagnostic,
	}

	cmd := exec.Command(pathToEngine)

	var err error

	uci.stdin, err = cmd.StdinPipe()
	if err != nil {
		return nil, err
	}

	uci.stdout, err = cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}

	end := make(chan error)
	go func() {
		err := cmd.Run()
		if err != nil {
			end <- err
			return
		}
		err = cmd.Wait()
		if err != nil {
			end <- err
			return
		}
	}()

	return uci, nil
}

// Start communication with the UCI engine.
func (uci *UCI) Start() error {
	buf := make([]byte, 4096)

	// get banner
	n, err := uci.stdout.Read(buf)
	if err != nil {
		return err
	}

	select {
	case uci.diagnostic <- bots.Diagnostic{
		Group:      "UCI",
		Diagnostic: string(buf[:n]),
	}:
	default:
	}

	// submit isready
	_, err = uci.stdin.Write([]byte("uci\n"))
	if err != nil {
		return err
	}

	// get uci response
	_, err = uci.stdout.Read(buf)
	if err != nil {
		return err
	}

	// submit isready
	_, err = uci.stdin.Write([]byte("isready\n"))
	if err != nil {
		return err
	}

	// get readyok reply
	_, err = uci.stdout.Read(buf)
	if err != nil {
		return err
	}

	// starting position
	_, err = uci.stdin.Write([]byte("ucinewgame\n"))
	if err != nil {
		return err
	}

	// no reply expected

	go func() {
		for {
			select {
			case move := <-uci.SubmitMove:
				if len(move) > 0 {
					uci.moves = append(uci.moves, move)

					move = "position startpos moves"
					for _, m := range uci.moves {
						move = fmt.Sprintf("%s %s", move, m)
					}
					move = fmt.Sprintf("%s\n", move)
					uci.stdin.Write([]byte(move))
				}

				_, err = uci.stdin.Write([]byte("go depth 10\n"))
				if err != nil {
					logger.Logf("uci", err.Error())
					return
				}

				done := false
				for !done {
					<-time.After(10 * time.Millisecond)

					n, err = uci.stdout.Read(buf)
					if err != nil {
						logger.Logf("uci", err.Error())
						return
					}

					select {
					case uci.diagnostic <- bots.Diagnostic{
						Group:      "UCI",
						Diagnostic: string(buf[:n]),
					}:
					default:
					}

					if n > 0 {
						s := strings.Index(string(buf[:n]), "bestmove ")
						if s > -1 {
							move = string(buf[s+9 : s+13])
							uci.GetMove <- move
							uci.moves = append(uci.moves, move)
							done = true
						}
					}
				}

			case <-uci.Quit:
				uci.close()
			}
		}
	}()

	return nil
}

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

package plusrom

import (
	"bytes"
	"fmt"
	"io"
	"net/http"

	"github.com/jetsetilly/gopher2600/logger"
)

type AddrInfo struct {
	Host string
	Path string
}

func (ai AddrInfo) String() string {
	return fmt.Sprintf("http://%s/%s", ai.Host, ai.Path)
}

type network struct {
	addr       AddrInfo
	sendBuffer bytes.Buffer
	recvBuffer bytes.Buffer
}

func (n *network) send(data uint8, send bool) {
	n.sendBuffer.WriteByte(data)
	if send {
		go func() {
			logger.Log("plusrom [http]", "sending")
			resp, err := http.Post(n.addr.String(), "Content-Type: application/octet-stream", &n.sendBuffer)
			if err != nil {
				fmt.Println(err.Error())
				return
			}
			n.recvBuffer.ReadFrom(resp.Body)
			l := n.recv()
			if int(l) != n.recvRemaining() {
				fmt.Println("unexpected length")
			}
		}()
	}
}

func (n *network) recvRemaining() int {
	return n.recvBuffer.Len()
}

func (n *network) recv() uint8 {
	b, err := n.recvBuffer.ReadByte()
	if err == io.EOF {
		return 0
	}
	return b
}

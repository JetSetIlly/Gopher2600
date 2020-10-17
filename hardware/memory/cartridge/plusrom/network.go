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
	"net/http"
	"net/http/httputil"
	"sync"

	"github.com/jetsetilly/gopher2600/logger"
)

const (
	sendBufferCap = 256
	recvBufferCap = 256
)

type network struct {
	prefs *Preferences
	ai    AddrInfo

	// these buffers should only be accessed directly by the emulator goroutine
	// (ie. not from the send() goroutine or by any UI goroutine come to that)
	sendBuffer bytes.Buffer
	recvBuffer bytes.Buffer

	// shared between emulator goroutine and send() goroutine
	respChan chan bytes.Buffer

	// to keep things simple we're insisting that one send() goroutine
	// concludes before starting another one
	//
	// alternatively, we could start the goroutine in newNetwork() and to
	// signal the start a new network event through a channel but I feel that
	// because plusrom network activity is so infrequent (probably) then this
	// method is okay. Whatever happens, I think we should only ever have one
	// network request active at once.
	sendLock sync.Mutex
}

func newNetwork(prefs *Preferences) *network {
	return &network{
		prefs:    prefs,
		respChan: make(chan bytes.Buffer, 5),
	}
}

// set to true to log HTTP repsonses/requests. this will do have to do until we
// implement logging levels.
const httpLogging = false

// add a single byte to the send buffer, capping the length of the buffer at
// the sendBufferCap value. if the "send" flag is true then the buffer is sent
// over the network. the function will not wait for the network activity.
func (n *network) send(data uint8, send bool) {
	if n.sendBuffer.Len() >= sendBufferCap {
		logger.Log("plusrom", "send buffer is full")
		return
	}
	n.sendBuffer.WriteByte(data)

	if send {
		go func(send bytes.Buffer, addr AddrInfo) {
			n.sendLock.Lock()
			defer n.sendLock.Unlock()

			logger.Log("plusrom [net]", fmt.Sprintf("sending to %s", addr.String()))

			req, err := http.NewRequest("POST", addr.String(), &send)
			if err != nil {
				logger.Log("plusrom [net]", err.Error())
				return
			}

			// content length HTTP header is the length of the send buffer
			req.Header.Set("Content-Length", fmt.Sprintf("%d", send.Len()))

			// from http://pluscart.firmaplus.de/pico/?PlusROM
			//
			// "The bytes are send to the back end as content of an HTTP 1.0
			// POST request with "Content-Type: application/octet-stream"."
			req.Header.Set("Content-Type", "application/octet-stream")

			// from http://pluscart.firmaplus.de/pico/?PlusROM
			//
			// "Emulators should generate a "PlusStore-ID" http-header with
			// their request, that consists of a nickname given by the user and
			// a generated uuid (starting with "WE") separated by a space
			// character."
			id := fmt.Sprintf("%s WE%s", n.prefs.Nick.String(), n.prefs.ID.String())
			req.Header.Set("PlusStore-ID", id)

			logger.Log("plusrom [net]", fmt.Sprintf("PlusStore-ID: %s", id))

			// log of complete request
			if httpLogging {
				s, _ := httputil.DumpRequest(req, true)
				logger.Log("plusrom [net]", fmt.Sprintf("request: %q", s))
			}

			// send response over network
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				logger.Log("plusrom [net]", err.Error())
				return
			}
			defer resp.Body.Close()

			// log of complete response
			if httpLogging {
				s, _ := httputil.DumpResponse(resp, true)
				logger.Log("plusrom [net]", fmt.Sprintf("response: %q", s))
			}

			// pass response to main goroutine
			var r bytes.Buffer
			_, err = r.ReadFrom(resp.Body)
			if err != nil {
				logger.Log("plusrom [net]", fmt.Sprintf("response: %v", err))
			}
			n.respChan <- r
		}(n.sendBuffer, n.ai)

		// a copy of the sendBuffer has been passed to the new goroutine so we
		// can now clear the references buffer
		n.sendBuffer = *bytes.NewBuffer([]byte{})
	}
}

// getResponse is called whenever recv() and recvRemaining() is called. it
// checks for a new HTTP responses and adds the response to the receive buffer.
func (n *network) getResponse() {
	select {
	case r := <-n.respChan:
		logger.Log("plusrom [net]", fmt.Sprintf("received %d bytes", r.Len()))

		l, err := r.ReadByte()
		if err != nil {
			logger.Log("plusrom", err.Error())
			return
		}

		if int(l) != r.Len() {
			logger.Log("plusrom [net]", "unexpected length received")
		}

		// from http://pluscart.firmaplus.de/pico/?PlusROM
		//
		// The response of the back end should also be a "Content-Type:
		// application/octet-stream" and the response-body should contain the payload
		// and the first byte of the response should be the length of the payload, so
		// "Content-Length" is payload + 1 byte. This is a workaround, because we don't
		// have enough time in the emulator routine to analyse the "Content-Length"
		// header of the response.
		_, err = n.recvBuffer.ReadFrom(&r)
		if err != nil {
			logger.Log("plusrom", err.Error())
			return
		}

		if n.recvBuffer.Len() > recvBufferCap {
			logger.Log("plusrom", "receive buffer is full")
			n.recvBuffer.Truncate(recvBufferCap)
		}

	default:
	}
}

// the number of bytes in the receive buffer. checks for network response.
func (n *network) recvRemaining() int {
	n.getResponse()
	return n.recvBuffer.Len()
}

// return the next byte in the receive buffer. returns 0 if buffer is empty.
func (n *network) recv() uint8 {
	n.getResponse()

	if n.recvBuffer.Len() == 0 {
		return 0
	}

	b, err := n.recvBuffer.ReadByte()
	if err != nil {
		logger.Log("plusrom", err.Error())
	}
	return b
}

// CopyRecvBuffer makes a copy of the bytes in the receive buffer.
func (cart *PlusROM) CopyRecvBuffer() []uint8 {
	return cart.net.recvBuffer.Bytes()
}

// CopySendBuffer makes a copy of the bytes in the send buffer.
func (cart *PlusROM) CopySendBuffer() []uint8 {
	return cart.net.sendBuffer.Bytes()
}

// SetRecvBuffer sets the entry that is idx places from the front with the
// specified value.
func (cart *PlusROM) SetRecvBuffer(idx int, data uint8) {
	c := cart.CopyRecvBuffer()
	c[idx] = data
	cart.net.recvBuffer = *bytes.NewBuffer(c)
}

// SetSendBuffer sets the entry that is idx places from the front with the
// specified value.
func (cart *PlusROM) SetSendBuffer(idx int, data uint8) {
	c := cart.CopySendBuffer()
	c[idx] = data
	cart.net.sendBuffer = *bytes.NewBuffer(c)
}

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

	"github.com/jetsetilly/gopher2600/environment"
	"github.com/jetsetilly/gopher2600/logger"
)

const (
	sendBufferCap = 256
	recvBufferCap = 256
)

type SendState struct {
	// the data that will be sent to the PlusROM network
	Buffer [sendBufferCap]uint8

	// the send buffer entry which will (over-)written next
	WritePtr uint8

	// the number of cpu clocks before the send buffer is transmitted over the
	// network. the value assigned to this field is not accurate with regards
	// to the baud rate of the PlusCart or the VCS system clock
	//
	// if the timeout value is zero then there is no pending transmission
	Cycles int

	// the amount of data to be transmitted when Cycles reaches zero. has no
	// meaning if Cycles equals zero. not the same as WritePtr
	SendLen uint8
}

type network struct {
	env *environment.Environment
	ai  AddrInfo

	send       SendState
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

func newNetwork(env *environment.Environment) *network {
	return &network{
		env:      env,
		respChan: make(chan bytes.Buffer, 5),
	}
}

// set to true to log HTTP repsonses/requests. this will do have to do until we
// implement logging levels.
const httpLogging = false

// add a single byte to the send buffer, capping the length of the buffer at
// the sendBufferCap value. if the "send" flag is true then the buffer is sent
// over the network. the function will not wait for the network activity.
func (n *network) buffer(data uint8) {
	n.send.Buffer[n.send.WritePtr] = data
	n.send.WritePtr++
}

func (n *network) commit() {
	n.send.SendLen = n.send.WritePtr
	n.send.WritePtr--

	// the figure of 1024 is not accurate but it is sufficient to emulate the
	// observed behaviour in the hardware. a realistic figure will be based on
	// the system clock of the VCS and the baudrate of the PlusCart (which is
	// 115200)
	n.send.Cycles = int(n.send.WritePtr) * 1024
}

func (n *network) transmitWait() {
	if n.send.Cycles > 0 {
		n.send.Cycles--
		if n.send.Cycles == 0 {
			n.transmit()
		}
	}
}

func (n *network) transmit() {
	var sendBuffer bytes.Buffer
	sendBuffer.Write(n.send.Buffer[:n.send.SendLen])

	go func(send bytes.Buffer, addr AddrInfo) {
		n.sendLock.Lock()
		defer n.sendLock.Unlock()

		logger.Logf("plusrom [net]", "sending to %s", addr.String())

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
		// "PlusCarts with firmware versions newer than v2.1.1 are sending a "PlusROM-Info" http header.
		//
		// The new "PlusROM-Info" header is discussed/explained in this AtariAge thread:
		// https://atariage.com/forums/topic/324456-redesign-plusrom-request-http-header"
		//
		id := fmt.Sprintf("agent=Gopher2600; ver=0.17.0; id=%s; nick=%s",
			// whether or not ID and Nick are valid has been handled in the preferences system
			n.env.Prefs.PlusROM.ID.String(),
			n.env.Prefs.PlusROM.Nick.String(),
		)
		req.Header.Set("PlusROM-Info", id)
		logger.Logf("plusrom [net]", "PlusROM-Info: %s", id)

		// -----------------------------------------------
		// PlusCart firmware earlier han v2.1.1
		//
		// "Emulators should generate a "PlusStore-ID" http-header with
		// their request, that consists of a nickname given by the user and
		// a generated uuid (starting with "WE") separated by a space
		// character."
		//
		// id := fmt.Sprintf("%s WE%s", n.env.Prefs.PlusROM.Nick.String(), n.env.Prefs.PlusROM.ID.String())
		// req.Header.Set("PlusStore-ID", id)
		// logger.Logf("plusrom [net]", "PlusStore-ID: %s", id)
		// -----------------------------------------------

		// log of complete request
		if httpLogging {
			s, _ := httputil.DumpRequest(req, true)
			logger.Logf("plusrom [net]", "request: %q", s)
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
			logger.Logf("plusrom [net]", "response: %q", s)
		}

		// pass response to main goroutine
		var r bytes.Buffer
		_, err = r.ReadFrom(resp.Body)
		if err != nil {
			logger.Logf("plusrom [net]", "response: %v", err)
		}
		n.respChan <- r
	}(sendBuffer, n.ai)

	// log send buffer
	logger.Log("plusrom [net] sent", fmt.Sprintf("% 02x", n.send.Buffer[:n.send.SendLen]))

	// a copy of the sendBuffer has been passed to the new goroutine so we
	// can now clear the references buffer
	n.send.WritePtr = 0
}

// getResponse is called whenever recv() and recvRemaining() is called. it
// checks for a new HTTP responses and adds the response to the receive buffer.
func (n *network) getResponse() {
	select {
	case r := <-n.respChan:
		logger.Logf("plusrom [net]", "received %d bytes", r.Len())

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

// GetSendState returns the current state of the network
func (cart *PlusROM) GetSendState() SendState {
	return cart.net.send
}

// SetSendBuffer sets the entry that is idx places from the front with the
// specified value.
func (cart *PlusROM) SetSendBuffer(idx int, data uint8) {
	cart.net.send.Buffer[idx] = data
}

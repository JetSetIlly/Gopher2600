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

package debugger

import (
	"github.com/jetsetilly/gopher2600/curated"
	"github.com/jetsetilly/gopher2600/debugger/govern"
	"github.com/jetsetilly/gopher2600/logger"
)

func argLen(args []govern.FeatureReqData, expectedLen int) error {
	if len(args) != expectedLen {
		return curated.Errorf("wrong number of arguments (%d instead of %d)", len(args), expectedLen)
	}
	return nil
}

// ReqFeature implements the govern.Emulation interface.
func (dbg *Debugger) SetFeature(request govern.FeatureReq, args ...govern.FeatureReqData) error {
	var err error

	switch request {
	case govern.ReqSetPause:
		err = argLen(args, 1)
		if err != nil {
			return curated.Errorf("debugger: %v", err)
		}

		switch dbg.Mode() {
		case govern.ModePlay:
			dbg.PushRawEvent(func() {
				if args[0].(bool) {
					dbg.setState(govern.Paused)
				} else {
					dbg.setState(govern.Running)
				}
			})
		case govern.ModeDebugger:
			err = curated.Errorf("not reacting to %s in debug mode (use terminal input instead)", request)
		}

	case govern.ReqSetMode:
		err = argLen(args, 1)
		if err != nil {
			return curated.Errorf("debugger: %v", err)
		}

		dbg.PushRawEventImmediate(func() {
			err := dbg.setMode(args[0].(govern.Mode))
			if err != nil {
				logger.Logf("debugger", err.Error())
			}
		})

	default:
	}

	return nil
}

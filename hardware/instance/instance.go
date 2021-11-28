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

package instance

import (
	"math/rand"
	"time"

	"github.com/jetsetilly/gopher2600/hardware/preferences"
)

type Instance struct {
	Prefs *preferences.Preferences

	// random values generated in the hardware package should use the following
	// number source
	RandSrc *rand.Rand

	// the number used to seed RandSrc
	randSeed int64
}

func NewInstance() (*Instance, error) {
	ins := &Instance{}

	var err error

	ins.Prefs, err = preferences.NewPreferences()
	if err != nil {
		return nil, err
	}

	ins.randSeed = int64(time.Now().Nanosecond())
	ins.RandSrc = rand.New(rand.NewSource(ins.randSeed))

	return ins, nil
}

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

package sdlimgui

import (
	"runtime"
	"time"

	"github.com/jetsetilly/imgui-go/v5"
)

type metric struct {
	ticker   *time.Ticker
	memstats runtime.MemStats
}

func newMetric() metric {
	return metric{
		ticker: time.NewTicker(time.Second),
	}
}

func (m *metric) update() {
	select {
	case <-m.ticker.C:
		runtime.ReadMemStats(&m.memstats)
	default:
		return
	}
}

func (m *metric) draw() {
	const MB = 1048576
	imgui.Textf("Used = %v MB\n", m.memstats.Alloc/MB)
	imgui.Textf("Reserved = %v MB\n", m.memstats.Sys/MB)
	imgui.Textf("GC Sweeps = %v", m.memstats.NumGC)
	imgui.Textf("GC CPU %% = %.2f%%", m.memstats.GCCPUFraction*100)
}

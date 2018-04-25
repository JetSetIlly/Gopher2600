package hardware

import "runtime"

func init() {
	runtime.GOMAXPROCS(1)
}

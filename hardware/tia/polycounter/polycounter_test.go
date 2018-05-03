package polycounter_test

import (
	"fmt"
	"gopher2600/hardware/tia/polycounter"
	"testing"
)

func TestPolycounter(t *testing.T) {
	pk := polycounter.New6BitPolycounter("111111")
	for pk.Tick() == false {
		fmt.Println(pk)
	}
}

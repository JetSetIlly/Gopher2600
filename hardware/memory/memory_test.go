package memory_test

import (
	"fmt"
	"headless/hardware/memory"
	"testing"
)

func TestMemory(t *testing.T) {
	mem := memory.NewVCSMemory()
	mem.Clear()

	fmt.Println(mem.MemoryMap())
}

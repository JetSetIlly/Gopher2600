package main

import (
	"fmt"
	"headlessVCS/hardware"
	"os"
)

func main() {
	vcs := hardware.NewVCS()
	err := vcs.AttachCartridge("flappy.bin")
	if err != nil {
		fmt.Println(err)
		os.Exit(10)
	}
}

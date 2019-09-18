package setup

import "gopher2600/hardware"

const defaultPanelInitialisation = ".gopher/setupDB"

// AttachCartridge to the VCS and apply setup information from the setupDB
func AttachCartridge(vcs *hardware.VCS, filename string) error {
	return vcs.AttachCartridge(filename)
}

package video

// createTriggerList is used by both player and missile sprites to decide on
// which color clocks past the sprite's reset position the sprite drawing
// routines should be triggered
func createTriggerList(playerSize uint8) []int {
	var triggerList []int
	switch playerSize {
	case 0x0, 0x05, 0x07:
		// empty trigger list - single sprite of varying widths
	case 0x01:
		triggerList = []int{4}
	case 0x02:
		triggerList = []int{8}
	case 0x03:
		triggerList = []int{4, 8}
	case 0x04:
		triggerList = []int{16}
	case 0x06:
		triggerList = []int{8, 16}
	}
	return triggerList
}

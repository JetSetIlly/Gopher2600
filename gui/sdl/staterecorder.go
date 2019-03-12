package sdl

import (
	"gopher2600/debugger/monitor"
	"gopher2600/errors"

	"github.com/veandco/go-sdl2/sdl"
)

type rgba struct {
	r byte
	g byte
	b byte
	a byte
}

type systemStateOverlay struct {
	scr *screen

	pixels  []byte
	texture *sdl.Texture

	colors map[string]rgba
	labels [][]string
}

var definedColors = []rgba{
	rgba{r: 255, g: 0, b: 0, a: 100},
	rgba{r: 0, g: 0, b: 255, a: 100},
	rgba{r: 0, g: 255, b: 0, a: 100},
	rgba{r: 255, g: 0, b: 255, a: 100},
	rgba{r: 255, g: 255, b: 0, a: 100},
	rgba{r: 0, g: 255, b: 255, a: 100},
}

func newSystemStateOverlay(scr *screen) (*systemStateOverlay, error) {
	overlay := new(systemStateOverlay)
	overlay.scr = scr
	overlay.colors = make(map[string]rgba)

	// our acutal screen data
	overlay.pixels = make([]byte, overlay.scr.maxWidth*overlay.scr.maxHeight*scrDepth)

	// labels
	overlay.labels = make([][]string, overlay.scr.maxHeight)
	for i := 0; i < len(overlay.labels); i++ {
		overlay.labels[i] = make([]string, overlay.scr.maxWidth)
	}

	var err error

	overlay.texture, err = scr.renderer.CreateTexture(uint32(sdl.PIXELFORMAT_ABGR8888), int(sdl.TEXTUREACCESS_STREAMING), int32(overlay.scr.maxWidth), int32(overlay.scr.maxHeight))
	if err != nil {
		return nil, err
	}
	overlay.texture.SetBlendMode(sdl.BlendMode(sdl.BLENDMODE_BLEND))

	return overlay, nil
}

func (overlay *systemStateOverlay) setPixel(attr monitor.SystemState) error {
	i := (overlay.scr.lastY*overlay.scr.maxWidth + overlay.scr.lastX) * scrDepth

	if i >= int32(len(overlay.pixels)) {
		return nil
	}

	// label required...
	if attr.Label == "" {
		errors.NewFormattedError(errors.CannotRecordState, "recording of system state requires a label")
	}

	// ... however, if a group has been supplied, use that to assign color
	var key string
	if attr.Group != "" {
		key = attr.Group
	} else {
		key = attr.Label
	}

	col, ok := overlay.colors[key]
	if !ok {
		overlay.colors[key] = definedColors[(len(overlay.colors)+1)%len(definedColors)]
		col = overlay.colors[key]
	}

	if col.a > 0 {
		overlay.pixels[i] = col.r   // red
		overlay.pixels[i+1] = col.g // green
		overlay.pixels[i+2] = col.b // blue
		overlay.pixels[i+3] = col.a // alpha
	}

	overlay.labels[overlay.scr.lastY][overlay.scr.lastX] = attr.Label

	return nil
}

func (overlay *systemStateOverlay) clearPixels() {
	for i := 0; i < len(overlay.pixels); i++ {
		overlay.pixels[i] = 0
	}
}

func (overlay *systemStateOverlay) update() error {
	err := overlay.texture.Update(nil, overlay.pixels, int(overlay.scr.maxWidth*scrDepth))
	if err != nil {
		return err
	}
	return nil
}

// SystemStateRecord recieves (and processes) additional emulator information from the emulator
func (tv *GUI) SystemStateRecord(attr monitor.SystemState) error {
	// don't do anything if debugging is not enabled
	if !tv.allowDebugging {
		return nil
	}

	err := tv.HeadlessTV.SystemStateRecord(attr)
	if err != nil {
		return err
	}

	return tv.scr.systemState.setPixel(attr)
}

package imagetv

import (
	"fmt"
	"gopher2600/debugger/monitor"
	"gopher2600/errors"
	"gopher2600/television"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
)

// ImageTV is a television implementation that writes images to disk
type ImageTV struct {
	television.HeadlessTV

	pixelWidth int

	screenGeom image.Rectangle

	// currImage is the image we write to, until newFrame() is called again
	currImage    *image.NRGBA
	currFrameNum int

	// this is the image we'll be saving when Save() is called
	lastImage    *image.NRGBA
	lastFrameNum int
}

// NewImageTV initialises a new instance of ImageTV
func NewImageTV(tvType string) (*ImageTV, error) {
	tv := new(ImageTV)

	err := television.InitHeadlessTV(&tv.HeadlessTV, tvType)
	if err != nil {
		return nil, err
	}

	// screen geometry
	tv.pixelWidth = 2
	tv.screenGeom = image.Rectangle{
		Min: image.Point{X: 0, Y: 0},
		Max: image.Point{X: tv.Spec.ClocksPerScanline * tv.pixelWidth, Y: tv.Spec.ScanlinesTotal},
	}
	// start a new frame
	tv.currFrameNum = -1 // we'll be adding 1 to this value immediately in newFrame()
	err = tv.newFrame()
	if err != nil {
		return nil, err
	}

	// register new frame callback from HeadlessTV to SDLTV
	// leaving SignalNewScanline() hook at its default
	tv.HookNewFrame = tv.newFrame
	tv.HookSetPixel = tv.setPixel

	return tv, nil
}

func (tv *ImageTV) newFrame() error {
	tv.lastImage = tv.currImage
	tv.lastFrameNum = tv.currFrameNum
	tv.currImage = image.NewNRGBA(tv.screenGeom)
	tv.currFrameNum++
	return nil
}

func (tv *ImageTV) setPixel(x, y int32, red, green, blue byte, vblank bool) error {
	col := color.NRGBA{R: red, G: green, B: blue, A: 255}
	tv.currImage.Set(int(x)*tv.pixelWidth, int(y), col)
	tv.currImage.Set(int(x)*tv.pixelWidth+1, int(y), col)
	return nil
}

// SystemStateRecord implements a save facility. the SystemState instance
// should have:
//
//	 o Group attribute set to the image path
//   o Label attribute set to the base filename
//
// final saved image will be stored as
//       <group/<label>_<framenum>.png
func (tv *ImageTV) SystemStateRecord(state monitor.SystemState) error {
	if state.Label == "" || state.Group == "" {
		return errors.NewGopherError(errors.ImageTV, "SystemStateRecord requires a Label and Group")
	}
	return tv.save(filepath.Join(state.Group, state.Label))
}

func (tv *ImageTV) save(fileNameBase string) error {
	if tv.lastImage == nil {
		return errors.NewGopherError(errors.ImageTV, "no data to save")
	}

	// prepare filename for image
	imageName := fmt.Sprintf("%s_%d.png", fileNameBase, tv.lastFrameNum)

	f, err := os.Open(imageName)
	if f != nil {
		f.Close()
		return errors.NewGopherError(errors.ImageTV, fmt.Errorf("image file (%s) already exists", imageName))
	}
	if err != nil && !os.IsNotExist(err) {
		return errors.NewGopherError(errors.ImageTV, err)
	}

	f, err = os.Create(imageName)
	if err != nil {
		return errors.NewGopherError(errors.ImageTV, err)
	}

	defer f.Close()

	err = png.Encode(f, tv.lastImage)
	if err != nil {
		return errors.NewGopherError(errors.ImageTV, err)
	}

	return nil
}

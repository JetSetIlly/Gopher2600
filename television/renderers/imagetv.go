package renderers

import (
	"fmt"
	"gopher2600/errors"
	"gopher2600/television"
	"image"
	"image/color"
	"image/png"
	"os"
)

// ImageTV is a television implementation that writes images to disk
type ImageTV struct {
	television.Television

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
func NewImageTV(tvType string, tv television.Television) (*ImageTV, error) {
	var err error
	imtv := new(ImageTV)

	// create or attach television implementation
	if tv == nil {
		imtv.Television, err = television.NewBasicTelevision(tvType)
		if err != nil {
			return nil, err
		}
	} else {
		// check that the quoted tvType matches the specification of the
		// supplied BasicTelevision instance. we don't really need this but
		// becuase we're implying that tvType is required, even when an
		// instance of BasicTelevision has been supplied, the caller may be
		// expecting an error
		if tvType != tv.GetSpec().ID {
			return nil, errors.NewFormattedError(errors.ImageTV, "trying to piggyback a tv of a different spec")
		}
		imtv.Television = tv
	}

	// screen geometry
	imtv.pixelWidth = 2
	imtv.screenGeom = image.Rectangle{
		Min: image.Point{X: 0, Y: 0},
		Max: image.Point{X: imtv.GetSpec().ClocksPerScanline * imtv.pixelWidth, Y: imtv.GetSpec().ScanlinesTotal},
	}
	// start a new frame
	imtv.currFrameNum = -1 // we'll be adding 1 to this value immediately in newFrame()
	err = imtv.NewFrame()
	if err != nil {
		return nil, err
	}

	// register ourselves as a television.Renderer
	imtv.AddRenderer(imtv)

	return imtv, nil
}

// Save last frame to filename - filename base supplied as an argument, the
// frame number and file extension is appended by the function
//
// return tv.Save(filepath.Join(state.Group, state.Label))
func (imtv *ImageTV) Save(fileNameBase string) error {
	if imtv.lastImage == nil {
		return errors.NewFormattedError(errors.ImageTV, "no data to save")
	}

	// prepare filename for image
	imageName := fmt.Sprintf("%s_%d.png", fileNameBase, imtv.lastFrameNum)

	f, err := os.Open(imageName)
	if f != nil {
		f.Close()
		return errors.NewFormattedError(errors.ImageTV, fmt.Errorf("image file (%s) already exists", imageName))
	}
	if err != nil && !os.IsNotExist(err) {
		return errors.NewFormattedError(errors.ImageTV, err)
	}

	f, err = os.Create(imageName)
	if err != nil {
		return errors.NewFormattedError(errors.ImageTV, err)
	}

	defer f.Close()

	err = png.Encode(f, imtv.lastImage)
	if err != nil {
		return errors.NewFormattedError(errors.ImageTV, err)
	}

	return nil
}

// NewFrame implements television.Renderer interface
func (imtv *ImageTV) NewFrame() error {
	imtv.lastImage = imtv.currImage
	imtv.lastFrameNum = imtv.currFrameNum
	imtv.currImage = image.NewNRGBA(imtv.screenGeom)
	imtv.currFrameNum++
	return nil
}

// NewScanline implements television.Renderer interface
func (imtv *ImageTV) NewScanline() error {
	return nil
}

// SetPixel implements television.Renderer interface
func (imtv *ImageTV) SetPixel(x, y int32, red, green, blue byte, vblank bool) error {
	col := color.NRGBA{R: red, G: green, B: blue, A: 255}
	imtv.currImage.Set(int(x)*imtv.pixelWidth, int(y), col)
	imtv.currImage.Set(int(x)*imtv.pixelWidth+1, int(y), col)
	return nil
}

// SetAltPixel implements television.Renderer interface
func (imtv *ImageTV) SetAltPixel(x, y int32, red, green, blue byte, vblank bool) error {
	return nil
}

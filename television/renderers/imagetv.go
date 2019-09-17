package renderers

import (
	"fmt"
	"gopher2600/errors"
	"gopher2600/television"
	"image"
	"image/color"
	"image/png"
	"os"
	"strings"
)

// ImageTV is a television implementation that writes images to disk
type ImageTV struct {
	television.Television

	pixelWidth int

	screenGeom image.Rectangle

	// currFrameData is the image we write to, until newFrame() is called again
	currFrameData *image.NRGBA
	currFrameNum  int

	// this is the image we'll be saving when Save() is called
	lastFrameData *image.NRGBA
	lastFrameNum  int
}

// NewImageTV initialises a new instance of ImageTV
func NewImageTV(tvType string, tv television.Television) (*ImageTV, error) {
	var err error
	imtv := new(ImageTV)

	// create or attach television implementation
	if tv == nil {
		imtv.Television, err = television.NewStellaTelevision(tvType)
		if err != nil {
			return nil, err
		}
	} else {
		// check that the quoted tvType matches the specification of the
		// supplied BasicTelevision instance. we don't really need this but
		// becuase we're implying that tvType is required, even when an
		// instance of BasicTelevision has been supplied, the caller may be
		// expecting an error
		tvType = strings.ToUpper(tvType)
		if tvType != "AUTO" && tvType != tv.GetSpec().ID {
			return nil, errors.NewFormattedError(errors.ImageTV, "trying to piggyback a tv of a different spec")
		}
		imtv.Television = tv
	}

	// set attributes that depend on the television specification
	imtv.ChangeTVSpec()

	// start a new frame
	imtv.currFrameNum = -1 // we'll be adding 1 to this value immediately in newFrame()
	err = imtv.NewFrame(imtv.currFrameNum)
	if err != nil {
		return nil, err
	}

	// register ourselves as a television.Renderer
	imtv.AddRenderer(imtv)

	return imtv, nil
}

// ChangeTVSpec implements television.Television interface
func (imtv *ImageTV) ChangeTVSpec() error {
	imtv.pixelWidth = 2
	imtv.screenGeom = image.Rectangle{
		Min: image.Point{X: 0, Y: 0},
		Max: image.Point{X: television.ClocksPerScanline * imtv.pixelWidth, Y: imtv.GetSpec().ScanlinesTotal},
	}
	return nil
}

// Save last frame to filename - filename base supplied as an argument, the
// frame number and file extension is appended by the function
//
// currentFrame should be true if the current frame (which may be incomplete)
// should be save. if the value is false then the previous frame will be saved
//
// return tv.Save(filepath.Join(state.Group, state.Label))
func (imtv *ImageTV) Save(fileNameBase string, currentFrame bool) error {
	if imtv.lastFrameData == nil {
		return errors.NewFormattedError(errors.ImageTV, "no data to save")
	}

	// prepare filename for image
	var imageName string
	if currentFrame {
		imageName = fmt.Sprintf("%s_%d.png", fileNameBase, imtv.currFrameNum)
	} else {
		imageName = fmt.Sprintf("%s_%d.png", fileNameBase, imtv.lastFrameNum)
	}

	f, err := os.Open(imageName)
	if f != nil {
		f.Close()
		return errors.NewFormattedError(errors.ImageTV, fmt.Sprintf("image file (%s) already exists", imageName))
	}
	if err != nil && !os.IsNotExist(err) {
		return errors.NewFormattedError(errors.ImageTV, err)
	}

	f, err = os.Create(imageName)
	if err != nil {
		return errors.NewFormattedError(errors.ImageTV, err)
	}

	defer f.Close()

	if currentFrame {
		err = png.Encode(f, imtv.currFrameData)
	} else {
		err = png.Encode(f, imtv.lastFrameData)
	}
	if err != nil {
		return errors.NewFormattedError(errors.ImageTV, err)
	}

	return nil
}

// NewFrame implements television.Renderer interface
func (imtv *ImageTV) NewFrame(frameNum int) error {
	imtv.lastFrameData = imtv.currFrameData
	imtv.lastFrameNum = imtv.currFrameNum
	imtv.currFrameData = image.NewNRGBA(imtv.screenGeom)
	imtv.currFrameNum++
	return nil
}

// NewScanline implements television.Renderer interface
func (imtv *ImageTV) NewScanline(scanline int) error {
	return nil
}

// SetPixel implements television.Renderer interface
func (imtv *ImageTV) SetPixel(x, y int32, red, green, blue byte, vblank bool) error {
	col := color.NRGBA{R: red, G: green, B: blue, A: 255}
	imtv.currFrameData.Set(int(x)*imtv.pixelWidth, int(y), col)
	imtv.currFrameData.Set(int(x)*imtv.pixelWidth+1, int(y), col)
	return nil
}

// SetAltPixel implements television.Renderer interface
func (imtv *ImageTV) SetAltPixel(x, y int32, red, green, blue byte, vblank bool) error {
	return nil
}

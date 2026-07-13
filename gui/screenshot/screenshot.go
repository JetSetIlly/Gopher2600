// This file is part of Gopher2600.
//
// Gopher2600 is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// Gopher2600 is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with Gopher2600.  If not, see <https://www.gnu.org/licenses/>.

package screenshot

import (
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"strings"

	"github.com/jetsetilly/gopher2600/hardware/television/specification"
	"github.com/jetsetilly/gopher2600/logger"
	"github.com/jetsetilly/gopher2600/resources/unique"
	"golang.org/x/image/draw"
)

const (
	// base scaling amount for raw pixel images
	scaling = 3

	// aspectBias transforms the scaling factor for the X axis. this is a different aspect
	// bias to the one we use in win_dbgscr. the difference will be down to the the
	// projection matrix used for rendering; and if not that, a viewport difference. either
	// way, it doesn't matter because the results are consistent
	aspectBias = 0.80
)

// ScaleRawPixels scales a raw pixel image to a 'standard' screenshot format. Don't use this for
// screenshots generated from sources that have applied CRT filters etc. Those images will have been
// scaled according to their own rules already
func ScaleRawPixels(p *image.RGBA) *image.RGBA {
	bounds := p.Bounds()
	scaledBounds := bounds
	scaledBounds.Max.X = int(float64(scaledBounds.Max.X*specification.PixelWidth*scaling) * aspectBias)
	scaledBounds.Max.Y *= scaling
	scaled := image.NewRGBA(scaledBounds)
	draw.NearestNeighbor.Scale(scaled, scaledBounds, p, bounds, draw.Src, nil)
	return scaled
}

// GenerateFilename creates a path using the cartridge name and other information. Paths generated with
// this funcion always have "scrshot" as a prefix. If the 'id' argument is empty the filename is given
// a unique name based on the current timestamp (see unique package). If the 'desc' argument is
// non-empty then that is appended after the 'id' or the unique assigned timestamp
func GenerateFilename(cartName string, id string, desc string) string {
	var path string

	if len(id) == 0 {
		path = unique.Filename("scrshot", cartName)
	} else {
		path = fmt.Sprintf("scrshot_%s_%s", cartName, id)
	}
	if len(desc) > 0 {
		path = fmt.Sprintf("%s_%s", path, desc)
	}

	return path
}

// Save writes the image to the specified path.
func Save(rgba *image.RGBA, path string) {
	ext := filepath.Ext(path)
	switch strings.ToLower(ext) {
	case ".png":
		savePNG(rgba, path)
	case ".jpg", ".jpeg":
		saveJPEG(rgba, path)
	default:
		path = strings.TrimSuffix(path, ext)
		saveJPEG(rgba, fmt.Sprintf("%s.jpg", path))
	}
}

func saveJPEG(rgba *image.RGBA, path string) {
	f, err := os.Create(path)
	if err != nil {
		logger.Logf(logger.Allow, "screenshot", "jpeg save failed: %v", err)
		return
	}

	err = jpeg.Encode(f, rgba, &jpeg.Options{Quality: 100})
	if err != nil {
		logger.Logf(logger.Allow, "screenshot", "jpeg save failed: %v", err)
		_ = f.Close()
		return
	}

	err = f.Close()
	if err != nil {
		logger.Logf(logger.Allow, "screenshot", "jpeg save failed: %v", err)
		return
	}

	// indicate success
	logger.Logf(logger.Allow, "screenshot", "saved: %s", path)
}

func savePNG(rgba *image.RGBA, path string) {
	f, err := os.Create(path)
	if err != nil {
		logger.Logf(logger.Allow, "screenshot", "png save failed: %v", err)
		return
	}

	err = png.Encode(f, rgba)
	if err != nil {
		logger.Logf(logger.Allow, "screenshot", "png save failed: %v", err)
		_ = f.Close()
		return
	}

	err = f.Close()
	if err != nil {
		logger.Logf(logger.Allow, "screenshot", "png save failed: %v", err)
		return
	}

	// indicate success
	logger.Logf(logger.Allow, "screenshot", "saved: %s", path)
}

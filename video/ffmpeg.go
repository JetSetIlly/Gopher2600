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

package video

import (
	"fmt"
	"io"
	"os"
	"os/exec"

	"github.com/jetsetilly/gopher2600/logger"
	"github.com/jetsetilly/gopher2600/resources/unique"
)

// Profile is used to specify which set of ffmpeg settings to use
type Profile int

// List of valid Profile values
const (
	ProfileFast Profile = iota
	Profile1080
	ProfileYouTube1080
	ProfileYouTube4k
)

type Renderer interface {
	ReadPixels(width int32, height int32, pix []uint8)
}

type FFMPEG struct {
	rnd Renderer

	// details set by the preprocess() function. the function checks to see if the parameters change
	// between calls. if they change while ffmpeg is enabled then the preprocess() function returns
	// an error
	cartName string
	width    int32
	height   int32
	hz       float32
	profile  Profile

	// is video recording enabled
	enabled bool

	// the running ffmpeg command and the data pipe from the emulation
	encoder *exec.Cmd
	pipe    io.WriteCloser

	// pixels is read by the ReadPixels() function of the supplied Renderer interface
	pixels []uint8
}

func NewFFMPEG(rnd Renderer) *FFMPEG {
	vid := &FFMPEG{
		rnd: rnd,
	}
	return vid
}

func (vid *FFMPEG) Destroy() {
	if vid.pipe != nil {
		vid.pipe.Close()
		if err := vid.encoder.Wait(); err != nil {
			logger.Log(logger.Allow, "video", err.Error())
		}
		vid.pipe = nil
		vid.encoder = nil
	}
	vid.pixels = nil
	vid.enabled = false
}

func (vid *FFMPEG) Preprocess(cartName string, width int32, height int32, hz float32, profile Profile) error {
	if !vid.enabled {
		vid.Destroy()
		return nil
	}

	if vid.pipe != nil {
		if vid.width != width || vid.height != height {
			vid.Destroy()
			return fmt.Errorf("ffmpeg: size of frame has changed")
		}
		if vid.hz != hz {
			vid.Destroy()
			return fmt.Errorf("ffmpeg: refresh rate of monitor has changed")
		}
		return nil
	}

	vid.cartName = cartName
	vid.width = width
	vid.height = height
	vid.hz = hz
	vid.profile = profile

	outputFile := unique.Filename("video", cartName)

	var ffmpegInput = []string{
		"-f", "rawvideo",
		"-pix_fmt", "rgba",
		"-s", fmt.Sprintf("%dx%d", vid.width, vid.height),
		"-r", fmt.Sprintf("%.02f", hz), // incoming frame rate
		"-i", "-", // stdin pipe created below
	}

	var ffmpegFast = []string{
		"-crf", "18", // amount of compression. 12 and higher starts to lose colour fidelity
		"-preset", "fast", // the amount of time spent optimising compression between frames
		"-vf", "vflip", // the data read from the OpenGL buffer is flipped
		"-r", "60", // output is always 60fps
	}

	var ffmpeg1080p = []string{
		"-crf", "11", // amount of compression. 12 and higher starts to lose colour fidelity unless pix_fmt is yuv420p10le
		// the default and fastest pix_fmt is yuv420p and we
		"-preset", "medium", // the amount of time spent optimising compression between frames
		"-vf", "vflip,scale=-2:1080,pad=1920:1080:(ow-iw)/2:(oh-ih)/2",
		"-r", "60", // output is always 60fps
	}

	var ffmpegYouTube1080 = []string{
		"-c:v", "libx264",
		"-preset", "slow", // the amount of time spent optimising compression between frames
		"-pix_fmt", "yuv420p10le",
		"-crf", "15", // amount of compression. 15 is a good value for yuv420p10le
		"-profile:v", "high10",
		"-vf", "vflip,scale=-2:1080,pad=1920:1080:(ow-iw)/2:(oh-ih)/2",
		"-r", "60", // output is always 60fps
	}

	var ffmpegYouTube4k = []string{
		"-c:v", "libx264",
		"-preset", "slow", // the amount of time spent optimising compression between frames
		"-pix_fmt", "yuv420p10le",
		"-crf", "15", // amount of compression. 15 is a good value for yuv420p10le
		"-profile:v", "high10",
		"-vf", "vflip,scale=-2:2160,pad=3840:2160:(ow-iw)/2:(oh-ih)/2",
		"-r", "60", // output is always 60fps
	}

	var ffmpegOutput = []string{
		"-y", // always overwrite output file
		fmt.Sprintf("%s.mp4", outputFile),
	}

	var opts []string

	opts = append(opts, ffmpegInput...)
	switch vid.profile {
	case ProfileFast:
		opts = append(opts, ffmpegFast...)
	case Profile1080:
		opts = append(opts, ffmpeg1080p...)
	case ProfileYouTube1080:
		opts = append(opts, ffmpegYouTube1080...)
	case ProfileYouTube4k:
		opts = append(opts, ffmpegYouTube4k...)
	}
	opts = append(opts, ffmpegOutput...)

	vid.encoder = exec.Command("ffmpeg", opts...)

	var err error
	vid.pipe, err = vid.encoder.StdinPipe()
	if err != nil {
		return fmt.Errorf("ffmpeg: %w", err)
	}

	vid.encoder.Stderr = os.Stderr
	vid.encoder.Stdout = os.Stdout

	err = vid.encoder.Start()
	if err != nil {
		return fmt.Errorf("ffmpeg: %w", err)
	}

	vid.pixels = make([]uint8, vid.width*vid.height*4)

	return nil
}

func (vid *FFMPEG) Enable(enable bool) {
	vid.enabled = enable
}

func (vid *FFMPEG) IsRecording() bool {
	return vid.pipe != nil
}

func (vid *FFMPEG) Process() {
	if vid.pipe == nil {
		return
	}

	// get pixel data for frame and forward it to the running command
	vid.rnd.ReadPixels(vid.width, vid.height, vid.pixels)

	_, err := vid.pipe.Write(vid.pixels)
	if err != nil {
		logger.Log(logger.Allow, "video", err.Error())
		vid.Destroy()
	}
}

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

//go:build !gl21

package sdlimgui

import (
	"fmt"
	"image"
	"io"
	"os"
	"os/exec"

	"github.com/go-gl/gl/v3.2-core/gl"
	"github.com/jetsetilly/gopher2600/logger"
)

type ffmpegProfile int

const (
	ffmpegProfile1080 ffmpegProfile = iota
	ffmpegProfileYouTube1080
	ffmpegProfileYouTube4k
)

type gl32Video struct {
	width  int32
	height int32

	encoder *exec.Cmd
	pipe    io.WriteCloser

	frame     *image.RGBA
	lastFrame int

	// is video recording enabled
	enabled bool
	profile ffmpegProfile
}

func newGl32Video() *gl32Video {
	vid := &gl32Video{
		enabled: false,
		profile: ffmpegProfile1080,
	}
	return vid
}

func (vid *gl32Video) destroy() {
	if vid.pipe != nil {
		vid.pipe.Close()
		if err := vid.encoder.Wait(); err != nil {
			logger.Log(logger.Allow, "video", err.Error())
		}
		vid.pipe = nil
	}
	vid.frame = nil
}

func (vid *gl32Video) start(outputFile string, frameNum int, width int32, height int32, hz float32) error {
	if !vid.enabled {
		return nil
	}
	if vid.pipe != nil {
		return nil
	}

	vid.width = width
	vid.height = height
	vid.lastFrame = frameNum

	var ffmpegInput = []string{
		"-f", "rawvideo",
		"-pix_fmt", "rgba",
		"-s", fmt.Sprintf("%dx%d", vid.width, vid.height),
		"-r", fmt.Sprintf("%.02f", hz), // incoming frame rate
		"-i", "-", // stdin pipe created below
	}

	var ffmpeg1080p = []string{
		"-crf", "11", // amount of compression. 12 and higher starts to lose colour fidelity
		"-preset", "medium", // the amount of time spent optimising compression between frames
		"-vf", "vflip,scale=1920:1080", // scale to 1080p
		"-r", "60", // output is always 60fps
	}

	var ffmpegYouTube4k = []string{
		"-c:v", "libx264",
		"-preset", "slow", // the amount of time spent optimising compression between frames
		"-crf", "15", // amount of compression
		"-pix_fmt", "yuv420p10le",
		"-profile:v", "high10",
		"-vf", "vflip,scale=3840:2160", // scale output to 4k
		"-r", "60", // output is always 60fps
	}

	var ffmpegYouTube1080 = []string{
		"-c:v", "libx264",
		"-preset", "slow", // the amount of time spent optimising compression between frames
		"-crf", "15", // amount of compression
		"-pix_fmt", "yuv420p10le",
		"-profile:v", "high10",
		"-vf", "vflip,scale=1920:1080", // scale output to 4k
		"-r", "60", // output is always 60fps
	}

	var ffmpegOutput = []string{
		"-y", // always overwrite output file
		fmt.Sprintf("%s.mp4", outputFile),
	}

	var opts []string

	opts = append(opts, ffmpegInput...)
	switch vid.profile {
	case ffmpegProfile1080:
		opts = append(opts, ffmpeg1080p...)
	case ffmpegProfileYouTube1080:
		opts = append(opts, ffmpegYouTube1080...)
	case ffmpegProfileYouTube4k:
		opts = append(opts, ffmpegYouTube4k...)
	}
	opts = append(opts, ffmpegOutput...)

	vid.encoder = exec.Command("ffmpeg", opts...)

	var err error
	vid.pipe, err = vid.encoder.StdinPipe()
	if err != nil {
		return fmt.Errorf("video: %w", err)
	}

	vid.encoder.Stderr = os.Stderr
	vid.encoder.Stdout = os.Stdout

	err = vid.encoder.Start()
	if err != nil {
		return fmt.Errorf("video: %w", err)
	}

	vid.frame = image.NewRGBA(image.Rect(0, 0, int(vid.width), int(vid.height)))

	return nil
}

func (vid *gl32Video) isRecording() bool {
	return vid.pipe != nil
}

func (vid *gl32Video) process(frameNum int, width int32, height int32) {
	if vid.pipe == nil {
		return
	}

	if !vid.enabled {
		vid.destroy()
		return
	}

	if frameNum <= vid.lastFrame {
		return
	}
	if frameNum != vid.lastFrame+1 {
		fmt.Printf("video: skipped %d frames", frameNum-vid.lastFrame+1)
	}
	vid.lastFrame = frameNum

	if vid.pipe == nil {
		return
	}
	if vid.width != width || vid.height != height {
		logger.Logf(logger.Allow, "video", "size of frame has changed")
		vid.destroy()
	}

	gl.ReadPixels(0, 0, width, height, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(vid.frame.Pix))

	// frame is vertically flipped at this point. the frame is corrected during video conversion

	_, err := vid.pipe.Write(vid.frame.Pix)
	if err != nil {
		logger.Log(logger.Allow, "video", err.Error())
		vid.destroy()
	}
}

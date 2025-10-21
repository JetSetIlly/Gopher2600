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
	"strconv"
	"strings"
	"time"

	"github.com/jetsetilly/gopher2600/hardware/television"
	"github.com/jetsetilly/gopher2600/hardware/television/frameinfo"
	"github.com/jetsetilly/gopher2600/resources/unique"
)

type Profile string

const (
	ProfileFast        Profile = "FAST"
	Profile1080        Profile = "1080"
	ProfileYouTube1080 Profile = "YouTube1080"
	ProfileYouTube4k   Profile = "YouTube4k"
)

// Session is used to configure the FFMPEG process on the call to Enable()
type Session struct {
	Log       io.Writer
	LastFrame int
	Profile   Profile
}

type Renderer interface {
	ReadPixels(width int32, height int32, pix []uint8)
}

type Television interface {
	AddAudioMixer(m television.AudioMixer)
	RemoveAudioMixer(m television.AudioMixer)
	GetFrameInfo() frameinfo.Current
}

type FFMPEG struct {
	rnd Renderer
	tv  Television

	// details set by the Preprocess() function. the function checks to see if the parameters change
	// between calls. if they change while ffmpeg is enabled then the Preprocess() function returns
	// an error
	cartName           string
	width              int32
	height             int32
	hz                 float32
	profile            Profile
	finalVideoFilename string
	tempVideoFilename  string
	tempAudioFilename  string

	// session configuration set during the Enable() function
	conf Session

	// is video recording enabled
	enabled bool

	// the time the recording started
	start time.Time

	// the running ffmpeg command and the data pipe from the emulation
	encoder *exec.Cmd
	pipe    io.WriteCloser

	// pixels is read by the ReadPixels() function of the supplied Renderer interface
	pixels            []uint8
	lastFrameRendered int

	// we record audio to a separate file and then mux it with the video in a final step
	wavs television.AudioMixer
}

func NewFFMPEG(rnd Renderer, tv Television) *FFMPEG {
	vid := &FFMPEG{
		rnd: rnd,
		tv:  tv,
	}

	return vid
}

func (vid *FFMPEG) Destroy() {
	vid.enabled = false
	vid.pixels = nil

	// we try to mux if we ever close the ffmpeg pipe or wavwriter
	var muxVideo bool
	var muxAudio bool

	if vid.pipe != nil {
		vid.pipe.Close()
		if err := vid.encoder.Wait(); err != nil {
			if vid.conf.Log != nil {
				fmt.Fprintln(vid.conf.Log, err.Error())
			}
		}
		vid.pipe = nil
		vid.encoder = nil
		muxVideo = true
	}

	if vid.wavs != nil {
		if err := vid.wavs.EndMixing(); err != nil {
			if vid.conf.Log != nil {
				fmt.Fprintln(vid.conf.Log, err.Error())
			}
		}
		vid.tv.RemoveAudioMixer(vid.wavs)
		vid.wavs = nil
		muxAudio = true
	}

	if !(muxVideo && muxAudio) {
		return
	}

	// summarise results
	if vid.conf.Log != nil {
		diff := time.Since(vid.start)
		fps := float64(vid.lastFrameRendered) / diff.Seconds()

		hrs := int(diff.Hours()) % 24
		mins := int(diff.Minutes()) % 60
		secs := int(diff.Seconds()) % 60

		var dur strings.Builder
		if hrs > 0 {
			fmt.Fprintf(&dur, " %dhr", hrs)
			if hrs > 1 {
				fmt.Fprintf(&dur, "s")
			}
		}
		if mins > 0 {
			fmt.Fprintf(&dur, " %dmin", mins)
			if mins > 1 {
				fmt.Fprintf(&dur, "s")
			}
		}
		if secs > 0 {
			fmt.Fprintf(&dur, " %dsec", secs)
			if secs > 1 {
				fmt.Fprintf(&dur, "s")
			}
		}

		fmt.Fprintf(vid.conf.Log, "%d frames recorded in%s (%.02f fps)\n", vid.lastFrameRendered, dur.String(), fps)
	}

	// probe temporary files before muxing
	if vid.conf.Log != nil {
		fmt.Fprintln(vid.conf.Log, "probing intermediary video and audio files")
	}

	// duration of video file
	probeVideo := exec.Command("ffprobe",
		"-v", "error",
		"-show_entries", "format=duration", "-of", "default=noprint_wrappers=1:nokey=1",
		vid.tempVideoFilename)

	probeResult, err := probeVideo.Output()
	if err != nil {
		if vid.conf.Log != nil {
			fmt.Fprintln(vid.conf.Log, err.Error())
		}
		return
	}
	probeResult = []byte(strings.TrimSpace(string(probeResult)))

	videoRate, err := strconv.ParseFloat(string(probeResult), 64)
	if err != nil {
		if vid.conf.Log != nil {
			fmt.Fprintln(vid.conf.Log, err.Error())
		}
		return
	}

	// duration of audio file
	probeAudio := exec.Command("ffprobe",
		"-v", "error",
		"-show_entries", "format=duration", "-of", "default=noprint_wrappers=1:nokey=1",
		vid.tempAudioFilename)

	probeResult, err = probeAudio.Output()
	if err != nil {
		if vid.conf.Log != nil {
			fmt.Fprintln(vid.conf.Log, err.Error())
		}
		return
	}
	probeResult = []byte(strings.TrimSpace(string(probeResult)))

	audioRate, err := strconv.ParseFloat(string(probeResult), 64)
	if err != nil {
		if vid.conf.Log != nil {
			fmt.Fprintln(vid.conf.Log, err.Error())
		}
		return
	}

	// calculate stretch value
	stretch := audioRate / videoRate

	if vid.conf.Log != nil {
		fmt.Fprintf(vid.conf.Log, "stretching audio by a factor of %0.2f\n", 1.0/stretch)
	}

	// muxing with ffmpeg using the probed and calculated rates
	if vid.conf.Log != nil {
		fmt.Fprintf(vid.conf.Log, "muxing final output file: %s\n", vid.finalVideoFilename)
	}

	muxer := exec.Command("ffmpeg",
		"-v", "error",
		"-i", vid.tempVideoFilename, "-i", vid.tempAudioFilename,
		"-vcodec", "copy", "-acodec", "mp3",
		"-filter:a", fmt.Sprintf("atempo=%f", stretch),
		vid.finalVideoFilename)

	// using Run() function because we want to wait for ffmpeg to complete
	err = muxer.Run()
	if err != nil {
		if vid.conf.Log != nil {
			fmt.Fprintln(vid.conf.Log, err.Error())
		}
		return
	}

	// removing temp files only if probing and muxing has succeeded
	err = os.Remove(vid.tempVideoFilename)
	if err != nil {
		if vid.conf.Log != nil {
			fmt.Fprintln(vid.conf.Log, err.Error())
		}
	}
	err = os.Remove(vid.tempAudioFilename)
	if err != nil {
		if vid.conf.Log != nil {
			fmt.Fprintln(vid.conf.Log, err.Error())
		}
	}
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

	// the base for the output file's name. we append .mp4 for the video file and .wav for the audio file
	outputFilenameBase := unique.Filename("video", cartName)
	vid.finalVideoFilename = fmt.Sprintf("%s.mp4", outputFilenameBase)
	vid.tempVideoFilename = fmt.Sprintf("_tmp_%s.mp4", outputFilenameBase)
	vid.tempAudioFilename = fmt.Sprintf("_tmp_%s.wav", outputFilenameBase)

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
		"-v", "error", // less noisy output from the ffmpeg command
		"-y", // always overwrite output file
		vid.tempVideoFilename,
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
	default:
		return fmt.Errorf("ffmpeg: unknown profile: %d", vid.profile)
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

	// create wavwriter via local audio type
	vid.wavs, err = newAudio(vid.tempAudioFilename, vid.tv.GetFrameInfo().Spec)
	if err != nil {
		return fmt.Errorf("ffmpeg: %w", err)
	}
	vid.tv.AddAudioMixer(vid.wavs)

	if vid.conf.Log != nil {
		fmt.Fprintln(vid.conf.Log, "recording video")
	}

	return nil
}

func (vid *FFMPEG) Enable(enable bool, conf Session) error {
	vid.conf = conf
	vid.enabled = enable
	vid.start = time.Now()

	if vid.conf.Log != nil {
		fmt.Fprintln(vid.conf.Log, "testing for ffmpeg and ffprobe")
	}

	// check that both ffprobe and ffmpeg are available in the executable path
	if vid.enabled {
		if _, err := exec.LookPath("ffmpeg"); err != nil {
			vid.enabled = false
			return fmt.Errorf("ffmpeg not installed")
		} else {
			if _, err := exec.LookPath("ffprobe"); err != nil {
				vid.enabled = false
				return fmt.Errorf("ffprobe not installed")
			}
		}
	}

	return nil
}

func (vid *FFMPEG) IsRecording() bool {
	return vid.pipe != nil
}

func (vid *FFMPEG) Process(framenum int) {
	if vid.pipe == nil {
		return
	}

	if framenum != -1 && framenum <= vid.lastFrameRendered {
		return
	}
	vid.lastFrameRendered = framenum

	if vid.conf.Log != nil {
		if framenum > vid.conf.LastFrame {
			fmt.Fprintf(vid.conf.Log, "frame %d\r", framenum)
		} else {
			fmt.Fprintf(vid.conf.Log, "frame %d of %d\r", framenum, vid.conf.LastFrame)
		}
	}

	// get pixel data for frame and forward it to the running command
	vid.rnd.ReadPixels(vid.width, vid.height, vid.pixels)

	_, err := vid.pipe.Write(vid.pixels)
	if err != nil {
		if vid.conf.Log != nil {
			fmt.Fprintln(vid.conf.Log, err.Error())
		}
		vid.Destroy()
	}
}

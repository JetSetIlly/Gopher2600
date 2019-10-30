package sdl

import (
	"gopher2600/hardware/tia/audio"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/veandco/go-sdl2/mix"
	"github.com/veandco/go-sdl2/sdl"
)

const samplePath = ".gopher2600/samples/little-scale_atari_2600_sample_pack"
const sampleSubPath = "Atari_2600_Cropped"

type sound struct {
	prevAud audio.Audio

	samples [16][32]*mix.Chunk
}

func newSound(gtv *GUI) (*sound, error) {
	snd := &sound{}

	// prerequisite: SDL_INIT_AUDIO must be included in the call to sdl.Init()
	mix.OpenAudio(22050, sdl.AUDIO_S16SYS, 2, 640)

	path := filepath.Join(samplePath, sampleSubPath)

	walkFn := func(p string, info os.FileInfo, err error) error {
		t := p
		t = strings.TrimPrefix(t, path)
		t = strings.TrimPrefix(t, string(os.PathSeparator))
		t = strings.TrimPrefix(t, sampleSubPath)
		t = strings.TrimPrefix(t, "--Wave_")

		s := strings.Split(t, string(os.PathSeparator))
		if len(s) != 2 {
			return nil
		}

		control, e := strconv.Atoi(s[0])
		if e != nil {
			return nil
		}

		s[1] = strings.TrimPrefix(s[1], sampleSubPath)
		s[1] = strings.TrimPrefix(s[1], "_")
		s[1] = strings.TrimSuffix(s[1], ".wav")

		freq, e := strconv.Atoi(s[1])
		if e != nil {
			return nil
		}
		freq = ((freq - 1) % 32)

		snd.samples[control][freq], e = mix.LoadWAV(p)
		if e != nil {
			return nil
		}

		return nil
	}

	err := filepath.Walk(path, walkFn)
	if err != nil {
		return nil, err
	}

	return snd, nil
}

// SetAudio implements the television.AudioMixer interface
func (gtv *GUI) SetAudio(aud audio.Audio) error {
	if aud.Volume0 != gtv.snd.prevAud.Volume0 {
		mix.Volume(0, int(aud.Volume0*8))
	}
	if aud.Volume1 != gtv.snd.prevAud.Volume1 {
		mix.Volume(1, int(aud.Volume1*8))
	}

	if aud.Control0 != gtv.snd.prevAud.Control0 || aud.Freq0 != gtv.snd.prevAud.Freq0 {
		if aud.Control0 == 0 {
			mix.HaltChannel(0)
		} else {
			gtv.snd.samples[aud.Control0][31-aud.Freq0].Play(0, -1)
		}
	}

	if aud.Control1 != gtv.snd.prevAud.Control1 || aud.Freq1 != gtv.snd.prevAud.Freq1 {
		if aud.Control1 == 0 {
			mix.HaltChannel(1)
		} else {
			gtv.snd.samples[aud.Control1][31-aud.Freq1].Play(1, -1)
		}
	}

	gtv.snd.prevAud = aud

	return nil
}

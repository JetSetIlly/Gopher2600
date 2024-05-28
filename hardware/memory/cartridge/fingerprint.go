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

package cartridge

import (
	"bytes"
	"fmt"
	"io"

	"github.com/jetsetilly/gopher2600/cartridgeloader"
)

// if anwhere parameter is true then the ELF magic number can appear anywhere
// in the data (the b parameter). otherwise it must appear at the beginning of
// the data
func fingerprintElf(loader cartridgeloader.Loader, anywhere bool) bool {
	if anywhere {
		if loader.Contains([]byte{0x7f, 'E', 'L', 'F'}) {
			return true
		}
	} else {
		b := make([]byte, 4)
		loader.Seek(0, io.SeekStart)
		if n, err := loader.Read(b); n != len(b) || err != nil {
			return false
		}
		if bytes.Equal(b, []byte{0x7f, 'E', 'L', 'F'}) {
			return true
		}
	}

	return false
}

func fingerprintAce(loader cartridgeloader.Loader) (bool, bool) {
	// some ACE files embed an ELF file inside the ACE data. these files are
	// identified by the presence of "elf-relocatable" in the data premable
	wrappedELF := loader.ContainsLimit(144, []byte("elf-relocatable"))

	// make double sure this is actually an elf file. otherwise it's just an
	// ACE file with elf-relocatable in the data preamble
	wrappedELF = wrappedELF && fingerprintElf(loader, true)

	if loader.ContainsLimit(144, []byte("ACE-2600")) {
		return true, wrappedELF
	}

	if loader.ContainsLimit(144, []byte("ACE-PC00")) {
		return true, wrappedELF
	}

	if loader.ContainsLimit(144, []byte("ACE-UF00")) {
		return true, wrappedELF
	}

	return false, false
}

func (cart *Cartridge) fingerprintPlusROM(loader cartridgeloader.Loader) bool {
	// there is a second fingerprint that occurs in the NewPlusROM() function

	b := make([]byte, 3)
	loader.Seek(0, io.SeekStart)

	// fingerprinting beyond the first 1k can easily result in a false positive
	const fingerprintLimit = 1024

	for i := 0; i < fingerprintLimit-len(b); i++ {
		n, err := loader.Read(b)
		if n < len(b) {
			break
		}
		if b[0] == 0x8d && b[1] == 0xf1 && b[2]&0x10 == 0x10 {
			return true
		}
		if err == io.EOF {
			break
		}
		loader.Seek(int64(1-len(b)), io.SeekCurrent)
	}

	return false
}

func fingerprint3e(loader cartridgeloader.Loader) bool {
	// 3E cart bankswitching is triggered by storing the bank number in address
	// 3E using 'STA $3E', commonly followed by an  immediate mode LDA
	//
	// fingerprint method taken from:
	//
	// https://gitlab.com/firmaplus/atari-2600-pluscart/-/blob/master/source/STM32firmware/PlusCart/Src/cartridge_detection.c#L140
	return loader.Contains([]byte{0x85, 0x3e, 0xa9, 0x00})
}

func fingerprint3ePlus(loader cartridgeloader.Loader) bool {
	// previous versions of this function worked similarly to the tigervision
	// method but this is more accurate
	//
	// fingerprint method taken from:
	//
	// https://gitlab.com/firmaplus/atari-2600-pluscart/-/blob/master/source/STM32firmware/PlusCart/Src/cartridge_detection.c#L148
	return loader.Contains([]byte{'T', 'J', '3', 'E'})
}

func fingerprintMnetwork(loader cartridgeloader.Loader) bool {
	// limit size of MNetwork cartridges to 16k
	if loader.Size() > 16384 {
		return false
	}

	// Bump 'n' Jump is the fussiest mnetwork cartridge I've found. Matching
	// hotspots:
	//
	//	$fdd5 LDA      BANK5
	//	$fde3 LDA      BANK6
	//	$fe0e LDA      BANK5
	//	$fe16 LDA      BANK4
	//
	// This also catches modern games not created by mnetwork but which use the
	// format, eg Pitkat
	//
	// (24/01/21)
	//
	// Splendidnut's Congobongo demo is even fussier than Bump 'n' Jump. While
	// I expect the ROM to get more complex we should support the demo ROM if
	// only because it exists.
	//
	// notably, it uses a non-primary-mirror cartridge address and there are
	// only two bankswitch instructions in the data (although I didn't search
	// it exhaustively - just LDA instructions).
	//
	// threshold has been reduced to two.
	//
	// (19/06/21)
	//
	// Thomas Jentzsch's Elite demo requires a threshold of one
	//
	// https://atariage.com/forums/topic/155657-elite-3d-graphics/?do=findComment&comment=2444328
	//
	// with such a low threshold, mnetwork should probably be the very last type to check for
	threshold := 1

	b := make([]byte, 3)
	loader.Seek(0, io.SeekStart)

	for i := 0; i < loader.Size()-len(b); i++ {
		n, err := loader.Read(b)
		if n < len(b) {
			break
		}

		if b[0] == 0xad && b[1] >= 0xe0 && b[1] <= 0xe7 {
			// bank switching can address any cartidge mirror so mask off
			// insignificant bytes
			//
			// (09/03/21) mask wasn't correct (0x0f selects non-cartridge
			// mirrors too) correct mask is 0x1f.
			//
			// the incorrect mask caused a false positive for Solaris when the
			// threshold is 2.
			//
			// (20/06/21) this caused a false positive for "Hack Em Hangly Pacman"
			// when the threshold is 1
			//
			// change to only look for mirrors 0x1f and 0xff
			if b[2] == 0x1f || b[2] == 0xff {
				threshold--
				if threshold == 0 {
					return true
				}
			}
		}

		if err == io.EOF {
			break
		}

		loader.Seek(int64(1-len(b)), io.SeekCurrent)
	}

	return false
}

func fingerprintParkerBros(loader cartridgeloader.Loader) bool {
	// parker bros fingerprint taken from Stella
	fingerprint := [][]byte{
		{0x8d, 0xe0, 0x1f}, // STA $1FE0
		{0x8d, 0xe0, 0x5f}, // STA $5FE0
		{0x8d, 0xe9, 0xff}, // STA $FFE9
		{0x0c, 0xe0, 0x1f}, // NOP $1FE0
		{0xad, 0xe0, 0x1f}, // LDA $1FE0
		{0xad, 0xe9, 0xff}, // LDA $FFE9
		{0xad, 0xed, 0xff}, // LDA $FFED
		{0xad, 0xf3, 0xbf}, // LDA $BFF3
	}
	for _, f := range fingerprint {
		if loader.Contains(f) {
			return true
		}
	}
	return false
}

func fingerprintDF(loader cartridgeloader.Loader) bool {
	b := make([]byte, 4)
	loader.Seek(0x0ff8, io.SeekStart)
	if n, err := loader.Read(b); n != len(b) || err != nil {
		return false
	}
	return bytes.Equal(b, []byte{'D', 'F', 'S', 'C'})
}

func fingerprintWickstead(loader cartridgeloader.Loader) bool {
	// wickstead design fingerprint taken from Stella
	return loader.Contains([]byte{0xa5, 0x39, 0x4c})
}

func fingerprintSCABS(loader cartridgeloader.Loader) bool {
	// SCABS fingerprint taken from Stella
	fingerprint := [][]byte{
		{0x20, 0x00, 0xd0, 0xc6, 0xc5}, // JSR $D000; DEC $C5
		{0x20, 0xc3, 0xf8, 0xa5, 0x82}, // JSR $F8C3; LDA $82
		{0xd0, 0xfB, 0x20, 0x73, 0xfe}, // BNE $FB; JSR $FE73
		{0x20, 0x00, 0xf0, 0x84, 0xd6}, // JSR $F000; $84, $D6
	}
	for _, f := range fingerprint {
		if loader.Contains(f) {
			return true
		}
	}
	return false
}

func fingerprintUA(loader cartridgeloader.Loader) bool {
	// ua fingerprint taken from Stella
	fingerprint := [][]byte{
		{0x8D, 0x40, 0x02}, // STA $240 (Funky Fish, Pleiades)
		{0xAD, 0x40, 0x02}, // LDA $240 (???)
		{0xBD, 0x1F, 0x02}, // LDA $21F,X (Gingerbread Man)
		{0x2C, 0xC0, 0x02}, // BIT $2C0 (Time Pilot)
		{0x8D, 0xC0, 0x02}, // STA $2C0 (Fathom, Vanguard)
		{0xAD, 0xC0, 0x02}, // LDA $2C0 (Mickey)
	}
	for _, f := range fingerprint {
		if loader.Contains(f) {
			return true
		}
	}
	return false
}

func fingerprintDPCplus(loader cartridgeloader.Loader) bool {
	b := make([]byte, 4)
	loader.Seek(0x0020, io.SeekStart)
	if n, err := loader.Read(b); n != len(b) || err != nil {
		return false
	}
	return bytes.Equal(b, []byte{0x1e, 0xab, 0xad, 0x10})
}

func fingerprintCDF(loader cartridgeloader.Loader) (bool, string) {
	if loader.ContainsLimit(2048, []byte("PLUSCDFJ")) {
		return true, "CDFJ+"
	}

	if loader.ContainsLimit(2048, []byte("CDFJ")) {
		return true, "CDFJ"
	}

	// old-school CDF version detection

	b := make([]byte, 4)
	loader.Seek(0, io.SeekStart)

	// fingerprinting beyond the first 1k can easily result in a false positive
	const fingerprintLimit = 1024

	for i := 0; i < fingerprintLimit-len(b); i++ {
		n, err := loader.Read(b)
		if n < len(b) {
			break
		}

		if bytes.Equal(b[:3], []byte("CDF")) {
			return true, fmt.Sprintf("CDF%1d", b[3])
		}

		if err == io.EOF {
			break
		}

		loader.Seek(int64(1-len(b)), io.SeekCurrent)
	}

	return false, ""
}

func fingerprintSuperchargerFastLoad(cartload cartridgeloader.Loader) bool {
	return cartload.Size() > 0 && cartload.Size()%8448 == 0
}

func fingerprintTigervision(loader cartridgeloader.Loader) bool {
	// tigervision cartridges change banks by writing to memory address 0x3f. we
	// can hypothesise that these types of cartridges will have that instruction
	// sequence "85 3f" many times in a ROM whereas other cartridge types will not
	threshold := 5
	return loader.Count([]byte{0x85, 0x3f}) > threshold
}

func fingerprintWF8(loader cartridgeloader.Loader) bool {
	// only one cartridge is known to use this. for now we'll use the MD5 sum
	// of the only known dump to match. the cartridge is an early version of
	// Smurf Rescue which apart from the different bank switching method is
	// exactly the same as the regular F8 version of the game
	//
	// https://forums.atariage.com/topic/367157-smurf-rescue-alternative-rom-with-wf8-bankswitch-format/
	//
	// [28th May 2024] second cartridge found. a variant of Zaxxon. we'll
	// continue to use the full MD5 sum for matching either of the examples. if
	// any more cartridges are found a more generalised fingerprint will be
	// found
	//
	// https://forums.atariage.com/topic/367200-zaxxon-alternative-rom-with-wf8-bankswitch-format/

	const (
		smurf  = "7b0ebb6bc1d700927f6efe34bac2ecd2"
		zaxxon = "494c0fb944d8d0d6b13c6b4b50ccbd11"
	)

	return loader.HashMD5 == smurf || loader.HashMD5 == zaxxon
}

func fingerprint8k(loader cartridgeloader.Loader) string {
	if fingerprintWF8(loader) {
		return "WF8"
	}

	if fingerprintTigervision(loader) {
		return "3F"
	}

	if fingerprintParkerBros(loader) {
		return "E0"
	}

	// mnetwork has the lowest threshold so place it at the end
	if fingerprintMnetwork(loader) {
		return "E7"
	}

	if fingerprintWickstead(loader) {
		return "WD"
	}

	if fingerprintSCABS(loader) {
		return "FE"
	}

	if fingerprintUA(loader) {
		return "UA"
	}

	return "F8"
}

func fingerprint16k(loader cartridgeloader.Loader) string {
	if fingerprintTigervision(loader) {
		return "3F"
	}

	if fingerprintMnetwork(loader) {
		return "E7"
	}

	return "F6"
}

func fingerprint32k(loader cartridgeloader.Loader) string {
	if fingerprintTigervision(loader) {
		return "3F"
	}
	return "F4"
}

func fingerprint64k(loader cartridgeloader.Loader) string {
	return "EF"
}

func fingerprint128k(loader cartridgeloader.Loader) string {
	if fingerprintDF(loader) {
		return "DF"
	}
	return "SB"
}

func fingerprint256k(loader cartridgeloader.Loader) string {
	return "SB"
}

func (cart *Cartridge) fingerprint(cartload cartridgeloader.Loader) (string, error) {
	// moviecart fingerprinting is done in cartridge loader. this is to avoid
	// loading the entire file into memory, which we definitely don't want to do
	// with moviecart files due to the large size

	if ok := fingerprintElf(cartload, false); ok {
		return "ELF", nil
	}

	if ok, wrappedElf := fingerprintAce(cartload); ok {
		if wrappedElf {
			return "ACE_wrapped_ELF", nil
		}
		return "ACE", nil
	}

	if ok, version := fingerprintCDF(cartload); ok {
		return version, nil
	}

	if fingerprintDPCplus(cartload) {
		return "DPC+", nil
	}

	if fingerprintSuperchargerFastLoad(cartload) {
		return "AR", nil
	}

	if fingerprint3ePlus(cartload) {
		return "3E+", nil
	}

	if fingerprint3e(cartload) {
		return "3E", nil
	}

	switch cartload.Size() {
	case 4096:
		return "4K", nil

	case 8195:
		// a widely distributed bad ROM dump of the Pink Panther prototype is
		// 8195 bytes long. we'll treat it like an 8k ROM and see if it's
		// recognised as a Wickstead Design ROM. if it's not then it's just a
		// file that's 8195 bytes long and will be rejected
		fallthrough

	case 8192:
		return fingerprint8k(cartload), nil

	case 10240:
		fallthrough

	case 10495:
		return "DPC", nil

	case 12288:
		return "FA", nil

	case 16384:
		return fingerprint16k(cartload), nil

	case 32768:
		return fingerprint32k(cartload), nil

	case 65536:
		return fingerprint64k(cartload), nil

	case 131072:
		return fingerprint128k(cartload), nil

	case 262144:
		return fingerprint256k(cartload), nil
	}

	if cartload.Size() >= 4096 {
		return "", fmt.Errorf("unrecognised size (%d bytes)", cartload.Size())
	}
	return "2K", nil
}

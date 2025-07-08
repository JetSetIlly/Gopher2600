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

package msa

import "fmt"

type Command interface {
	String() string
}

// List of all MSA Commands
var Commands [256]Command

func init() {
	for _, c := range controlCodes {
		if Commands[c.Code] != nil {
			panic(fmt.Sprintf("atarivox code tables are malformed: %d already used", c.Code))
		}
		Commands[c.Code] = c
	}
	for _, a := range allophones {
		if Commands[a.Code] != nil {
			panic(fmt.Sprintf("atarivox code tables are malformed: %d already used", a.Code))
		}
		Commands[a.Code] = a
	}
}

// ControlCode describes the MSA codes that control the output of future phonemes
type ControlCode struct {
	Code        byte
	Description string
	Double      bool
}

func (c ControlCode) String() string {
	return c.Description
}

// Allophone describes the MSA codes that cause a phoneme to be generated
type Allophone struct {
	Code    byte
	Phoneme string
	Sample  string
	msec    int
	Type    string
}

func (a Allophone) String() string {
	return a.Phoneme
}

var controlCodes = [...]ControlCode{
	{0, "Pause 0ms", false},
	{1, "Pause 1000ms", false},
	{2, "Pause 200ms", false},
	{3, "Pause 700ms", false},
	{4, "Pause 30ms", false},
	{5, "Pause 60ms", false},
	{6, "Pause 90ms", false},
	{7, "Fast", false},
	{8, "Slow", false},
	{14, "Stress", false},
	{15, "Relax", false},
	{16, "Wait", false},
	{20, "Volume", true},
	{21, "Speed", true},
	{22, "Pitch", true},
	{23, "Bend", true},
	{24, "PortCtr", true},
	{25, "Port", true},
	{26, "Repeat", true},
	{28, "Call Phrase", true},
	{29, "Goto Phrase", true},
	{30, "Delay", true},
	{31, "Reset", false},
}

var allophones = [...]Allophone{
	{128, "IY", "See, Even, Feed", 70, "Voiced Long Vowel"},
	{129, "IH", "Sit, Fix, Pin", 70, "Voiced Short Vowel"},
	{130, "EY", "Hair, Gate, Beige", 70, "Voiced Long Vowel"},
	{131, "EH", "Met, Check, Red", 70, "Voiced Short Vowel"},
	{132, "AY", "Hat, Fast, Fan", 70, "Voiced Short Vowel"},
	{133, "AX", "Cotten", 70, "Voiced Short Vowel"},
	{134, "UX", "Luck, Up, Uncle", 70, "Voiced Short Vowel"},
	{135, "OH", "Hot, Clock, Fox", 70, "Voiced Short Vowel"},
	{136, "AW", "Father, Fall", 70, "Voiced Short Vowel"},
	{137, "OW", "Comb, Over, Hold", 70, "Voiced Long Vowel"},
	{138, "UH", "Book, Could, Should", 70, "Voiced Short Vowel"},
	{139, "UW", "Food, June", 70, "Voiced Long Vowel"},
	{140, "MM", "Milk, Famous,", 70, "Voiced Nasal"},
	{141, "NE", "Nip, Danger, Thin", 70, "Voiced Nasal"},
	{142, "NO", "No, Snow, On", 70, "Voiced Nasal"},
	{143, "NGE", "Think, Ping", 70, "Voiced Nasal"},
	{144, "NGO", "Hung, Song", 70, "Voiced Nasal"},
	{145, "LE", "Lake, Alarm, Lapel", 70, "Voiced Resonate"},
	{146, "LO", "Clock, Plus, Hello", 70, "Voiced Resonate"},
	{147, "WW", "Wool, Sweat", 70, "Voiced Resonate"},
	{148, "RR", "Ray, Brain, Over", 70, "Voiced Resonate"},
	{149, "IYRR", "Clear, Hear, Year", 200, "Voiced R Color Vowel"},
	{150, "EYRR", "Hair, Stair, Repair", 200, "Voiced R Color Vowel"},
	{151, "AXRR", "Fir, Bird, Burn", 190, "Voiced R Color Vowel"},
	{152, "AWRR", "Part, Farm, Yarn", 200, "Voiced R Color Vowel"},
	{153, "OWRR", "Corn, Four, Your", 185, "Voiced R Color Vowel"},
	{154, "EYIY", "Gate, Ate, Ray", 165, "Voiced Diphthong"},
	{155, "OHIY", "Mice, Fight, White", 200, "Voiced Diphthong"},
	{156, "OWIY", "Boy, Toy, Voice", 225, "Voiced Diphthong"},
	{157, "OHIH", "Sky, Five, I", 185, "Voiced Diphthong"},
	{158, "IYEH", "Yes, Yarn, Million", 170, "Voiced Diphthong"},
	{159, "EHLL", "Saddle, Angle, Spell", 140, "Voiced Diphthong"},
	{160, "IYUW", "Cute, Few,", 180, "Voiced Diphthong"},
	{161, "AXUW", "Brown, Clown, Thousand", 170, "Voiced Diphthong"},
	{162, "IHWW", "Two, New, Zoo", 170, "Voiced Diphthong"},
	{163, "AYWW", "Our, Ouch, Owl", 200, "Voiced Diphthong"},
	{164, "OWWW", "Go, Hello, Snow", 131, "Voiced Diphthong"},
	{165, "JH", "Dodge, Jet, Savage", 70, "Voiced Affricate"},
	{166, "VV", "Vest, Even,", 70, "Voiced Fictive"},
	{167, "ZZ", "Zoo, Zap", 70, "Voiced Fictive"},
	{168, "ZH", "Azure, Treasure", 70, "Voiced Fictive"},
	{169, "DH", "There, That, This", 70, "Voiced Fictive"},
	{170, "BE", "Bear, Bird, Beed", 45, "Voiced Stop"},
	{171, "BO", "Bone, Book Brown", 45, "Voiced Stop"},
	{172, "EB", "Cab, Crib, Web", 10, "Voiced Stop"},
	{173, "OB", "Bob, Sub, Tub", 10, "Voiced Stop"},
	{174, "DE", "Deep, Date, Divide", 45, "Voiced Stop"},
	{175, "DO", "Do, Dust, Dog", 45, "Voiced Stop"},
	{176, "ED", "Could, Bird", 10, "Voiced Stop"},
	{177, "OD", "Bud, Food", 10, "Voiced Stop"},
	{178, "GE", "Get, Gate, Guest,", 55, "Voiced Stop"},
	{179, "GO", "Got, Glue, Goo", 55, "Voiced Stop"},
	{180, "EG", "Peg, Wig", 55, "Voiced Stop"},
	{181, "OG", "Dog, Peg", 55, "Voiced Stop"},
	{182, "CH", "Church, Feature, March", 70, "Voiceless Affricate"},
	{183, "HE", "Help, Hand, Hair", 70, "Voiceless Fricative"},
	{184, "HO", "Hoe, Hot, Hug", 70, "Voiceless Fricative"},
	{185, "WH", "Who, Whale, White", 70, "Voiceless Fricative"},
	{186, "FF", "Food, Effort, Off", 70, "Voiceless Fricative"},
	{187, "SE", "See, Vest, Plus", 40, "Voiceless Fricative"},
	{188, "SO", "So, Sweat", 40, "Voiceless Fricative"},
	{189, "SH", "Ship, Fiction, Leash", 50, "Voiceless Fricative"},
	{190, "TH", "Thin, month", 40, "Voiceless Fricative"},
	{191, "TT", "Part, Little, Sit", 50, "Voiceless Stop"},
	{192, "TU", "To, Talk, Ten", 70, "Voiceless Stop"},
	{193, "TS", "Parts, Costs, Robots", 170, "Voiceless Stop"},
	{194, "KE", "Can't, Clown, Key", 55, "Voiceless Stop"},
	{195, "KO", "Comb, Quick, Fox", 55, "Voiceless Stop"},
	{196, "EK", "Speak, Task", 55, "Voiceless Stop"},
	{197, "OK", "Book, Took, October", 45, "Voiceless Stop"},
	{198, "PE", "People, Computer", 99, "Voiceless Stop"},
	{199, "PO", "Paw, Copy", 99, "Voiceless Stop"},
}

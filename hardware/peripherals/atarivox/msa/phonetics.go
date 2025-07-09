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

// map of phoneme strings to phonetic information. phoneme strings are found in
// the Allophone type and so we can use this to map MSA commands to Phonetics
var Phonetics map[string]Phonetic

func init() {
	Phonetics = make(map[string]Phonetic)
	for _, p := range phonetics {
		Phonetics[p.Phoneme] = p
	}

	// check that all phonemes in allophone commands have an entry in the phonetics list
	for _, c := range Commands {
		if a, ok := c.(Allophone); ok {
			if _, ok := Phonetics[a.Phoneme]; !ok {
				panic("missing phoneme from Phonetics map")
			}
		}
	}
}

type Phonetic struct {
	Group       PhoneticGroup
	Description string
	Phoneme     string
	Notes       string
	Phonetic    string
}

type PhoneticGroup string

const (
	Vowels           PhoneticGroup = "Vowels"
	VowelsWithR      PhoneticGroup = "Vowels with R"
	Resonates        PhoneticGroup = "Resonates"
	Nasal            PhoneticGroup = "Nasal"
	VoicedFictive    PhoneticGroup = "Voiced Fictive"
	VoicelessFictive PhoneticGroup = "Voiceless Fictive"
	VoicedStop       PhoneticGroup = "Voiced Stop"
	VoicelessStop    PhoneticGroup = "Voiceless Stop"
	Affrictave       PhoneticGroup = "Affrictave"
)

var phonetics = [...]Phonetic{
	{Vowels, "Long A - 'Gate' & 'Ate'", "EYIY", "", "A"},
	{Vowels, "Long E - 'See' & 'Even'", "IY", "", "E"},
	{Vowels, "Long I - 'Sky' & 'Five'", "OHIH", "Also see IE", "I"},
	{Vowels, "Long O - 'Comb' & 'Over'", "OW", "Also see OA", "O"},
	{Vowels, "Long U - 'June' & 'Food'", "UW", "Also see OO", "U"},
	{Vowels, "Short A - 'Hat' & 'Fast'", "AY", "", "A"},
	{Vowels, "Short E - 'Cent' & 'Egg'", "EY", "Stressed", "E"},
	{Vowels, "Short E - 'Met' & 'Check'", "EH", "Normal", "E"},
	{Vowels, "Short E - 'Cotton' & 'dust'", "AX", "Relaxed", "E"},
	{Vowels, "Short I - 'Sit' & 'Fix'", "IH", "", "I"},
	{Vowels, "Short O - 'Hot' & 'Clock'", "OH", "Also see AW", "O"},
	{Vowels, "Short U - 'Luck' & 'Up'", "UX", "", "U"},
	{Vowels, "Pair OO - 'Book' & 'Could'", "UH", "Also see Long U", "OO"},
	{Vowels, "Pair AW - 'Saw' & 'Father'", "AW", "Also see Short O", "AW"},
	{Vowels, "Pair OA - 'Coat' & 'Hello'", "OWWW", "Also see Long O", "OA"},
	{Vowels, "Pair EW - 'New' & 'Two'", "IHWW", "eh-oo", "EW"},
	{Vowels, "Pair EW - 'Few' & 'Cute'", "IYUW", "ee-oo", "EW"},
	{Vowels, "Pair IE - 'Tie' & 'Fight'", "OHIY", "Also see Long I", "IE"},
	{Vowels, "Pair OW - “Owl' & 'Our'", "AYWW", "ah-ww", "OW"},
	{Vowels, "Pair OW - 'Brown'", "AXUW", "eh-uw", "OW"},
	{Vowels, "Pair OY - 'Boy' & 'Toy'", "OWIY", "", "OY"},
	{VowelsWithR, "Y - 'Yes' & 'Yarn'", "IYEH", "", "Y"},
	{VowelsWithR, "R - 'Ray' & 'Brain'", "RR", "", "R"},
	{VowelsWithR, "AIR - 'Hair' & 'Stair'", "EYRR", "", "AIR"},
	{VowelsWithR, "AR - 'Part' & 'Farm'", "AWRR", "", "AR"},
	{VowelsWithR, "EAR - 'Clear' & 'Hear'", "IYRR", "", "EAR"},
	{VowelsWithR, "ER - “Center” & 'Fir'", "AXRR", "", "ER"},
	{VowelsWithR, "OR - 'Corn' & 'Four'", "OWRR", "", "OR"},
	{Resonates, "EL - 'Saddle' & 'Angle'", "EHLL", "", "EL"},
	{Resonates, "L - 'Lake' & 'Alarm'", "LE", "Front", "L"},
	{Resonates, "L - 'Clock' & 'Plus'", "LO", "Back", "L"},
	{Resonates, "W - 'Wool' & 'Sweat'", "WW", "", "W"},
	{Nasal, "M - 'Milk' & 'Famous'", "MM", "", "M"},
	{Nasal, "N - 'Nip' & 'Danger'", "NE", "Front", "N"},
	{Nasal, "N - 'No' & 'Snow'", "NO", "Back", "N"},
	{Nasal, "N - 'Think' & 'Ping'", "NGE", "Front", "N"},
	{Nasal, "N - 'Hung' & 'Song'", "NGO", "Back", "N"},
	{VoicedFictive, "V - 'Vest' & 'Even'", "VV", "", "V"},
	{VoicedFictive, "Z - 'Zoo' & 'Zap'", "ZZ", "", "Z"},
	{VoicedFictive, "ZH - 'Azure' & 'Treasure'", "ZH", "", "ZH"},
	{VoicedFictive, "TH - 'There' & 'That'", "DH", "", "TH"},
	{VoicelessFictive, "H - 'Help' & 'Hand'", "HE", "Front", "H"},
	{VoicelessFictive, "H - 'Hoe' & 'Hot'", "HO", "Back", "H"},
	{VoicelessFictive, "WH - 'Who' & 'Whale'", "WH", "", "WH"},
	{VoicelessFictive, "F - 'Food' & 'Effort'", "FF", "", "F"},
	{VoicelessFictive, "S - 'See' & 'Vest'", "SE", "Front", "S"},
	{VoicelessFictive, "S - 'So' & 'Sweat'", "SO", "Back", "S"},
	{VoicelessFictive, "SH - 'Ship' & 'Fiction'", "SH", "", "SH"},
	{VoicelessFictive, "TH - 'Thin' & 'month'", "TH", "", "TH"},
	{VoicedStop, "B - 'Bear' & 'Bird'", "BE", "Front - Initial", "B"},
	{VoicedStop, "B - 'Bone' & 'Book'", "BO", "Back - Initial", "B"},
	{VoicedStop, "B - 'Cab' & 'Crib'", "EB", "Front", "B"},
	{VoicedStop, "B - 'Job' & 'Sub'", "OB", "Back", "B"},
	{VoicedStop, "D - 'Deep' & 'Date'", "DE", "Front - Initial", "D"},
	{VoicedStop, "D - 'Do' & 'Dust'", "DO", "Back - Initial", "D"},
	{VoicedStop, "D - 'Could' & 'Bird'", "ED", "Front", "D"},
	{VoicedStop, "D - 'Bud' & 'Food'", "OD", "Back", "D"},
	{VoicedStop, "G - 'Get' & 'Gate'", "GE", "Front - Initial", "G"},
	{VoicedStop, "G - 'Got' & 'Glue'", "GO", "Back - Initial", "G"},
	{VoicedStop, "G - 'Peg' & 'Wig'", "EG", "Front", "G"},
	{VoicedStop, "G - 'Dog' & 'Peg'", "OG", "Back", "G"},
	{VoicelessStop, "T - 'Part' & 'Little'", "TT", "", "T"},
	{VoicelessStop, "T - 'Tea' & 'Take'", "TU", "Front - Initial", "T"},
	{VoicelessStop, "TS - 'Parts' & 'Costs'", "TS", "", "TS"},
	{VoicelessStop, "K - 'Can't' & 'Clown'", "KE", "Front - Initial", "K"},
	{VoicelessStop, "K - 'Comb' & 'Quick'", "KO", "Back - Initial", "K"},
	{VoicelessStop, "K - 'Speak' & 'Task'", "EK", "Front", "K"},
	{VoicelessStop, "K - 'Book' & 'Took'", "OK", "Back", "K"},
	{VoicelessStop, "P - 'People' & 'Carpet'", "PE", "Front", "P"},
	{VoicelessStop, "P - 'Pod' & 'Paw'", "PO", "Back", "P"},
	{Affrictave, "JH - 'Dodge' & 'Jet'", "JH", "", "JH"},
	{Affrictave, "CH - 'Church' & 'Feature", "CH", "", "CH"},
}

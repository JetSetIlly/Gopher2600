// This file is part of Gopher2600.
//
// Gopher2600 is free software: you can redistribute it and/or modify
// it under the terms of the gnu general public license as published by
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

package cartridgeloader

// FileExtensions is the list of file extensions that are recognised by the
// cartridgeloader package.
var FileExtensions = [...]string{
	".BIN", ".ROM", ".A26", ".2K", ".4K", ".F8", ".F6",
	".F4", ".2K+", ".2KSC", ".4K+", ".4KSC", ".F8+", ".F8SC", ".F6+", ".F6SC", ".F4+", ".F4SC",
	".FA", ".FE", ".E0", ".E7", ".3F", ".AR", ".DF", "3E", "3E+", "SB",
	".DPC", ".DP+", "CDF", ".WAV", ".MP3", ".MVC",
}

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

// Package cdf implemnents the various CDF type cartridge mappers including
// CDFJ. It was developed with reference to Darrell Spice's CDJF blog and the
// source to the various example ROMs therein
//
// https://atariage.com/forums/forum/262-cdfj/
//
// Also, it seems that most complete survey of details for this cartridge type
// is the Stella source code. Therefore, I have resorted to the study of the
// CartCDF.cxx file as found in Stella 6.4.
//
// Note that all CDF formats rely on the arm7 package.
package cdf

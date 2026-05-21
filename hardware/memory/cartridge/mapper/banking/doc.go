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

// Package banking defines how to get information about banking into and out of
// a mapper implementation.
//
// # Selection
//
// The Selection type is created from a string specification using the
// SingleSelection() or SegmentedSelection() function. Which of these functions
// is used (by the mapper implementation) depends on the type of mapper.
//
// The SegmentedSelection() function is used by mappers that can have cartridge
// addresses mapped to multiple banks at once, like mnetwork or tigervision.
//
// SingleSelection() function is used for more traditional bank-switching
// mappers that map a single bank to the entire cartridge address space.
//
// The selection string is mostly useful for selecting the starting bank for the
// cartridge (ie. the banks that are mapped to the cartridge addresses on
// console reset)
//
// To help with this process there is also IsAutoSelection() function. This
// function checks if the selection string indicates that the mapper should
// automatically choose the correct starting bank. The string to indicate
// automatic bank selection is "AUTO".
//
// The SingleSelection() function expects a single number, whereas
// SegmentedSelection expects at least two numbers separated by a colon. The
// number of segments varies between the segemented mapper types.
//
// Some cartridge types can map additional RAM into cartridge space. To indicate
// a RAM mapping an 'R' should be appended to the number.
//
// Individual mappers can handle special cases by checking the string
// themselves. A good pattern for bank selection with special cases would looks
// something like this.
//
//	func (cart *myMapper) SetBank(bank string) error {
//		if banking.IsAutoSelection(selection) {
//			...
//			return nil
//		}
//
//		switch selection {
//		case special_case_1:
//			...
//			return nil
//		case special_case_2:
//			...
//			return nil
//		}
//
//		// or banking.SegmentedSelection() for cartridges with segmented mapping
//		sel, err := banking.SingleSelection(selection)
//		if err != nil {
//			return err
//		}
//
//		...
//
//		return nil
//	}
package banking

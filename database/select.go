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
//
// *** NOTE: all historical versions of this file, as found in any
// git repository, are also covered by the licence, even when this
// notice is not present ***

package database

import "gopher2600/errors"

// SelectAll entries in the database. onSelect can be nil.
//
// onSelect() should return true if select process is to continue. Continue
// flag is ignored if error is not nil.
//
// Returns last matched entry in selection or an error with the last entry
// matched before the error occurred.
func (db Session) SelectAll(onSelect func(Entry) (bool, error)) (Entry, error) {
	var entry Entry

	if onSelect == nil {
		onSelect = func(_ Entry) (bool, error) { return true, nil }
	}

	keyList := db.SortedKeyList()

	for k := range keyList {
		entry := db.entries[keyList[k]]
		cont, err := onSelect(entry)
		if err != nil {
			return entry, err
		}
		if !cont {
			break // for loop
		}
	}

	return entry, nil
}

// SelectKeys matches entries with the specified key(s). keys can be singular.
// if list of keys is empty then all keys are matched (SelectAll() maybe more
// appropriate in that case). onSelect can be nil.
//
// onSelect() should return true if select process is to continue. Continue
// flag is ignored if error is not nil.
//
// Returns last matched entry in selection or an error with the last entry
// matched before the error occurred.
func (db Session) SelectKeys(onSelect func(Entry) (bool, error), keys ...int) (Entry, error) {
	var entry Entry

	if onSelect == nil {
		onSelect = func(_ Entry) (bool, error) { return true, nil }
	}

	keyList := keys
	if len(keys) == 0 {
		keyList = db.SortedKeyList()
	}

	for i := range keyList {
		entry = db.entries[keyList[i]]
		cont, err := onSelect(entry)
		if err != nil {
			return entry, err
		}
		if !cont {
			break // for loop
		}
	}

	if entry == nil {
		return nil, errors.New(errors.DatabaseSelectEmpty)
	}

	return entry, nil
}

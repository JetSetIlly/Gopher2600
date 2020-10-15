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

// Package database is a very simple way of storing structured and arbitrary
// entry types. It's as simple as simple can be but is still useful in helping
// to organise what is essentially a flat file.
//
// Use of a database requires starting a "session". We do this with the
// StartSession() function, coupled with an EndSession() once we're done. For
// example (error handling removed for clarity):
//
//	db, _ := database.StartSession(dbPath, database.ActivityCreating, initDBSession)
//	defer db.EndSession(true)
//
// The first agument is the path to the database file on the local disk. The
// second argument is a description of the type of activity that will be
// happening during the session. In this instance, we saying that the database
// will be created if it does not already exist. If the database already exists
// ActivityCreating is treated the same as ActivityModifying. If we don't want
// to modify the database at all, then we can use ActivityReading.
//
// The third argument is the database initialisation function. An important part
// of this database package is its ability to handle arbitrary entry types. The
// initialisation function takes a pointer to the new database session as its
// sole argument:
//
//	func initSession(db *database.Session) {
//		db.RegisterEntryType("foo", deserialseFoo)
//		db.RegisterEntryType("bar", deserialseBar
//	}
//
// The RegisterEntryType() lets the database know what entry types it might
// expect. The first argument specifies the entry ID that will be stored in the
// database. On reading, the database will call the deserialisation function
// specified in the second argument.
//
// The deserialise function takes an array of strings as it's only argument and
// returns a new database.Entry and any errors. Database entries are
// deserialised as part of the StartSession() function. Any errors created by
// the deserialiser function cause the StartSession() to fail and to propagate
// the error outwards.
//
//	func deserialiseFoo(fields []string) (database.Entry, error) {
//		ent := &fooEntry{}
//		ent.numOfFoos = fields[0]
//		return ent, nil
//	}
//
// In this example, a Foo entry contains just one field. Fields are numbered
// from zero (the database entry will contain other fields but they are not
// passed to the deserialise function).
//
// For convenience, we copy the field entry to the fooEntry() type. In this
// instance, we do not need to convert the field type but if we did (it might
// be more convenient to treat a field as a boolean, for example) we would do
// it here.
//
// Deserialisation functions return a value that satisfies the database.Entry
// interface. See the Entry interface definition for details.
//
// Once a database session has successfully initialised, entries can be added,
// removed and selected/listed; activity type permitted.
package database

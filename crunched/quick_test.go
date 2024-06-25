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

package crunched_test

import (
	"crypto/md5"
	"math/rand"
	"testing"

	"github.com/jetsetilly/gopher2600/crunched"
	"github.com/jetsetilly/gopher2600/test"
)

func TestEmptyData_Quick(t *testing.T) {
	// create 100 bytes of empty data
	qa := crunched.NewQuick(100)

	// take hash of data before crunching
	preCrunchHash := md5.Sum(*qa.Data())

	// data should not be crunched
	test.ExpectFailure(t, qa.IsCrunched())

	// take a snapshot of the data
	qb := qa.Snapshot()

	// the snapshotted data should be crunched
	test.ExpectSuccess(t, qb.IsCrunched())

	// the original data should be left uncrunched
	test.ExpectSuccess(t, !qa.IsCrunched())

	// inspect the crunched data
	inspection := qb.(crunched.Inspection).Inspect()
	expectedData := []byte{0, 99}
	test.DemandEquality(t, len(*inspection), len(expectedData))
	for i, v := range *inspection {
		test.ExpectEquality(t, v, expectedData[i])
	}

	// check that hash of uncrunched data is the same as it was before
	postCrunchedHash := md5.Sum(*qb.Data())
	test.ExpectEquality(t, preCrunchHash, postCrunchedHash)

	// obtaining the data from the snapshot should leave the data in the
	// snapshot in an uncrunched state
	test.ExpectSuccess(t, !qb.IsCrunched())
}

func TestUncompressableData_quick(t *testing.T) {
	// create 256 bytes of empty data
	qa := crunched.NewQuick(256)

	// insert data that can't be compressed by the quick method
	data := qa.Data()
	for i := 0; i < len(*data); i++ {
		(*data)[i] = byte(i)
	}

	// take hash of data before crunching
	preCrunchHash := md5.Sum(*data)

	// take a snapshot of the data
	qb := qa.Snapshot()

	// the snapshotted data should not be crunched
	test.ExpectSuccess(t, !qb.IsCrunched())

	// check that hash of uncrunched data is the same as it was before
	postCrunchedHash := md5.Sum(*qb.Data())
	test.ExpectEquality(t, preCrunchHash, postCrunchedHash)
}

func TestEmptyData_ExampleData(t *testing.T) {
	// create 100 bytes of empty data
	qa := crunched.NewQuick(20)

	// insert data that can't be compressed by the quick method
	data := qa.Data()
	copy(*data, []byte{1, 2, 3, 3, 3, 3, 4, 4, 5, 6})

	// snapshot should successfully crunch the data
	qb := qa.Snapshot()
	test.ExpectSuccess(t, qb.IsCrunched())

	inspection := qb.(crunched.Inspection).Inspect()

	expectedData := []byte{1, 0, 2, 0, 3, 3, 4, 1, 5, 0, 6, 0, 0, 9}
	test.DemandEquality(t, len(*inspection), len(expectedData))
	for i, v := range *inspection {
		test.ExpectEquality(t, v, expectedData[i])
	}
}

func FuzzQuick(f *testing.F) {
	f.Fuzz(func(t *testing.T, size uint) {
		qa := crunched.NewQuick(int(size))

		// insert data that can't be compressed by the quick method
		data := qa.Data()
		(*data)[0] = byte(rand.Intn(255))
		for i := 1; i < len(*data); i++ {
			b := byte(rand.Intn(255))
			for b == (*data)[i-1] {
				b = byte(rand.Intn(255))
			}
			(*data)[i] = b
		}

		// take hash of data before crunching
		preCrunchHash := md5.Sum(*data)

		// take a snapshot of the data
		qb := qa.Snapshot()

		// the snapshotted data should not be crunched because it should be
		// impossible with the data we've given it
		test.ExpectSuccess(t, !qb.IsCrunched())

		// check that hash of uncrunched data is the same as it was before
		postCrunchedHash := md5.Sum(*qb.Data())
		test.ExpectEquality(t, preCrunchHash, postCrunchedHash)
	})
}

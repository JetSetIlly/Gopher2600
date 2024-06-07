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

package regression

import (
	"errors"
	"fmt"
	"io"
	"os"
	"slices"
	"sort"
	"strings"

	"github.com/jetsetilly/gopher2600/resources"
)

func saveFails(keys []string) error {
	sort.Strings(keys)
	keys = slices.Compact(keys)

	p, err := resources.JoinPath(regressionPath, fails)
	if err != nil {
		return fmt.Errorf("save fails: %w", err)
	}

	f, err := os.Create(p)
	if err != nil {
		return fmt.Errorf("save fails: %w", err)
	}
	defer func() {
		_ = f.Close()
	}()

	for _, v := range keys {
		f.WriteString(fmt.Sprintf("%s\n", v))
	}

	return nil
}

func loadFails() ([]string, error) {
	p, err := resources.JoinPath(regressionPath, fails)
	if err != nil {
		return []string{}, fmt.Errorf("load fails: %w", err)
	}

	f, err := os.Open(p)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return []string{}, fmt.Errorf("load fails: %w", err)
	}
	defer func() {
		_ = f.Close()
	}()

	b, err := io.ReadAll(f)
	if err != nil {
		return []string{}, fmt.Errorf("load fails: %w", err)
	}

	keys := strings.Split(string(b), "\n")

	sort.Strings(keys)
	keys = slices.Compact(keys)

	keys = slices.DeleteFunc(keys, func(s string) bool {
		s = strings.TrimSpace(s)
		return len(s) == 0
	})

	return keys, nil
}

var noPreviousFails = errors.New("no previous fails")

func addFailsToKeys(keys []string) ([]string, error) {
	sort.Strings(keys)
	keys = slices.Compact(keys)

	n := slices.IndexFunc(keys, func(s string) bool {
		return strings.ToUpper(s) == "FAILS"
	})
	if n >= 0 {
		keys = slices.Delete(keys, n, n+1)

		// load previous fails from disk
		prevFails, err := loadFails()
		if err != nil {
			return keys, err
		}

		if len(prevFails) == 0 {
			return keys, noPreviousFails
		}

		// merge previous fails with keys
		keys = append(keys, prevFails...)
	}

	return keys, nil
}

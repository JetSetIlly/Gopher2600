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

package archivefs

// SetSelectedFilename is called after a successful Set()
type FilenameSetter interface {
	SetSelectedFilename(string)
}

// AsyncResults are copies of archivefs path information that are safe to access asynchronously
type AsyncResults struct {
	Entries  []Node
	Selected string
	IsDir    bool
	Dir      string
	Base     string
}

// AsyncPath provides asynchronous access to an archivefs
type AsyncPath struct {
	setter FilenameSetter

	Set     chan string
	Close   chan bool
	Destroy chan bool

	results chan AsyncResults
	err     chan error

	// Results of most recent change of path settings
	Results AsyncResults
}

// NewAsyncPath is the preferred method of initialisation for the AsyncPath type
func NewAsyncPath(setter FilenameSetter) AsyncPath {
	pth := AsyncPath{
		setter:  setter,
		Set:     make(chan string, 1),
		Close:   make(chan bool, 1),
		Destroy: make(chan bool, 1),
		results: make(chan AsyncResults, 1),
		err:     make(chan error, 1),
	}

	go func() {
		var afs Path
		var done bool

		for !done {
			select {
			case <-pth.Destroy:
				done = true

			case <-pth.Close:
				afs.Close()

			case path := <-pth.Set:
				afs.Set(path)

				entries, err := afs.List()
				if err != nil {
					pth.err <- err
					return
				}

				pth.results <- AsyncResults{
					Entries:  entries,
					Selected: afs.String(),
					IsDir:    afs.IsDir(),
					Dir:      afs.Dir(),
					Base:     afs.Base(),
				}
			}
		}
	}()

	return pth
}

// Process asynchronous requests. Must be called in order to receive the results
// of a Set(). Suitable to be called as part of a render loop
func (pth *AsyncPath) Process() error {
	select {
	case err := <-pth.err:
		return err

	case results := <-pth.results:
		pth.Results = results

		if pth.setter != nil {
			if pth.Results.IsDir {
				pth.setter.SetSelectedFilename("")
			} else {
				pth.setter.SetSelectedFilename(pth.Results.Selected)
			}
		}
	default:
	}

	return nil
}

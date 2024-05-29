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
	Entries  []Entry
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
	entry   chan Entry
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

		// results must be an unbuffered channel to make sure that content from
		// the Entry channel comes after a new response from the results channle
		results: make(chan AsyncResults, 0),
		entry:   make(chan Entry, 100),
		err:     make(chan error, 0),
	}

	go func() {
		var afs Path
		var done bool

		// keep track of the most recent directory that has been read
		var currentDir string

		// cancel channel to communicate with any ongoing calls to Path.list()
		cancel := make(chan bool)

		for !done {
			select {
			case <-pth.Destroy:
				done = true

			case <-pth.Close:
				afs.Close()

			case path := <-pth.Set:
				// 1) send cancel to existing list goroutine
				select {
				case cancel <- true:
				default:
				}

				// 2) drain entry and err channel
				var listDrainDone bool
				for !listDrainDone {
					select {
					case <-pth.entry:
					case <-pth.err:
					default:
						listDrainDone = true
					}
				}

				// 3) drain cancel channel - in case the cancel signal we sent
				//    in (1) didn't go anywhere
				select {
				case <-cancel:
				default:
				}

				err := afs.Set(path, true)
				if err != nil {
					pth.err <- err
					continue // for loop
				}

				result := AsyncResults{
					Entries:  nil,
					Selected: afs.String(),
					IsDir:    afs.IsDir(),
					Dir:      afs.Dir(),
					Base:     afs.Base(),
				}

				// directory hasn't changed so there's no need to
				// call the list() function
				if currentDir == result.Dir {
					pth.results <- result
					continue // for loop
				}
				currentDir = result.Dir

				// this is a new directory being scanned. indicate that by
				// setting the Entries field to an empty list rather than nil
				result.Entries = []Entry{}
				pth.results <- result

				// all communication happens over channels so launching another
				// goroutine is fine. in fact, we have to do this in order for
				// the Set channel to be serviced. if we don't service
				go afs.list(pth.entry, pth.err, cancel)
			}
		}
	}()

	return pth
}

// Process results from previous push to Set channel
func (pth *AsyncPath) Process() error {
	done := false
	for !done {
		select {
		case err := <-pth.err:
			return err

		case ent := <-pth.entry:
			pth.Results.Entries = append(pth.Results.Entries, ent)
			select {
			case ent := <-pth.entry:
				pth.Results.Entries = append(pth.Results.Entries, ent)
			default:
				done = true
			}
			Sort(pth.Results.Entries)

		case results := <-pth.results:
			entries := pth.Results.Entries
			pth.Results = results
			if pth.Results.Entries == nil {
				pth.Results.Entries = entries
			}

			if pth.setter != nil {
				if pth.Results.IsDir {
					pth.setter.SetSelectedFilename("")
				} else {
					pth.setter.SetSelectedFilename(pth.Results.Selected)
				}
			}
		default:
			done = true
		}
	}

	return nil
}

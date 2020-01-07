// +build release

package paths

import (
	"os"
	"path"
)

const gopherConfigDir = "gopher2600"

// the release version of getBasePath looks for and if necessary creates the
// gopherConfigDir (and child directories) in the User's configuration
// directory, which is dependent on the host OS (see os.UserConfigDir()
// documentation for details)
func getBasePath(subPth string) (string, error) {
	cnf, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}

	pth := path.Join(cnf, gopherConfigDir, subPth)

	if _, err := os.Stat(pth); err == nil {
		return pth, nil
	}

	if err := os.MkdirAll(pth, 0700); err != nil {
		return "", err
	}

	return pth, nil
}

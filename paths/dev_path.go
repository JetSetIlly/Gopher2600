// +build !release

package paths

import (
	"os"
	"path"
)

const gopherConfigDir = ".gopher2600"

// the non-release version of getBasePath looks for and if necessary creates
// the gopherConfigDir (and child directories) in the current working
// directory
func getBasePath(subPth string) (string, error) {
	pth := path.Join(gopherConfigDir, subPth)

	if _, err := os.Stat(pth); err == nil {
		return pth, nil
	}

	if err := os.MkdirAll(pth, 0700); err != nil {
		return "", err
	}

	return pth, nil
}

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

package hiscore

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/jetsetilly/gopher2600/errors"
	"github.com/jetsetilly/gopher2600/paths"
)

const serverFile = "hiscoreServer"
const authFile = "hiscoreAuthentication"

// SetServer
func SetServer(input io.Reader, output io.Writer, server string) error {
	// server has not been provided so prompt for it
	if server == "" {
		output.Write([]byte("Enter server: "))
		var b []byte
		b = make([]byte, 255)
		_, err := input.Read(b)
		if err != nil {
			return errors.New(errors.HiScore, err)
		}
		server = string(b)
	}

	// limit extent of server setting
	s := strings.Split(server, "\n")
	server = s[0]
	if len(server) > 64 {
		server = server[:64]
	}

	// get path to server file
	serverFilePath, err := paths.ResourcePath("", serverFile)
	if err != nil {
		return errors.New(errors.HiScore, err)
	}

	// create a new file and write server to it
	f, err := os.Create(serverFilePath)
	if err != nil {
		return errors.New(errors.HiScore, err)
	}
	defer f.Close()
	fmt.Fprintf(f, "%s", server)

	return nil
}

// Login prepares the authentication token for the hiscore server
func Login(input io.Reader, output io.Writer, username string) error {
	sess, err := NewSession()
	if err != nil {
		if !errors.Is(err, errors.HiScoreNoAuthentication) {
			return errors.New(errors.HiScore, err)
		}
	}

	// prompt for username if it has not been supplied
	if strings.TrimSpace(username) == "" {
		output.Write([]byte("Enter username: "))
		var b []byte
		b = make([]byte, 255)
		_, err := input.Read(b)
		if err != nil {
			return errors.New(errors.HiScore, err)
		}
		username = strings.Split(string(b), "\n")[0]
	}

	// prompt for password
	// !TODO: noecho hiscore server password
	output.Write([]byte("(WARNING: password will be visible)\n"))
	output.Write([]byte("Enter password: "))
	var b []byte
	b = make([]byte, 255)
	_, err = input.Read(b)
	if err != nil {
		return errors.New(errors.HiScore, err)
	}
	password := strings.Split(string(b), "\n")[0]

	// send login for to server
	var cl http.Client
	data := url.Values{"username": {username}, "password": {password}}
	resp, err := cl.PostForm(fmt.Sprintf("%s/rest-auth/login/", sess.server), data)
	if err != nil {
		return errors.New(errors.HiScore, err)
	}

	// get response
	response, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return errors.New(errors.HiScore, err)
	}

	// unmarshal response
	var key map[string]string
	err = json.Unmarshal(response, &key)
	if err != nil {
		return errors.New(errors.HiScore, err)
	}

	// get path to auth file
	authFilePath, err := paths.ResourcePath("", authFile)
	if err != nil {
		return errors.New(errors.HiScore, err)
	}

	// create a new file (overwriting old file it exists) and write auth token
	f, err := os.Create(authFilePath)
	if err != nil {
		return errors.New(errors.HiScore, err)
	}
	defer f.Close()
	fmt.Fprintf(f, "%s", key["key"])

	return nil
}

// Logoff forgets the authentication token for the hiscore server
func Logoff() error {

	// !TODO: require hiscore server logoff confirmation

	// get path to auth file
	authFilePath, err := paths.ResourcePath("", authFile)
	if err != nil {
		return errors.New(errors.HiScore, err)
	}

	// forget token by overwriting any existing auth file
	f, err := os.Create(authFilePath)
	if err != nil {
		return errors.New(errors.HiScore, err)
	}
	defer f.Close()

	return nil
}

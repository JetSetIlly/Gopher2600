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
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"time"

	"github.com/jetsetilly/gopher2600/errors"
	"github.com/jetsetilly/gopher2600/paths"
)

type Session struct {
	id        string
	authToken string
	server    string
}

// NewSession is the preferred method of initialisation of the Session type.
//
// Session value will be valid if error value is:
//
//		errors.HiScoreNoAuthentication
//
// This is useful for the login procedure where we might expect there to be no
// authentication.
func NewSession() (*Session, error) {
	sess := &Session{}

	// get hiscore server
	serverFilePath, err := paths.ResourcePath("", serverFile)
	if err != nil {
		return nil, errors.New(errors.HiScore, err)
	}

	sf, err := os.Open(serverFilePath)
	if err != nil {
		switch err.(type) {
		case *os.PathError:
			return nil, errors.New(errors.HiScoreNoServer)
		}
		return nil, errors.New(errors.HiScore, err)
	}
	defer sf.Close()

	_, err = fmt.Fscanf(sf, "%s", &sess.server)
	if err != nil {
		if err == io.EOF {
			return nil, errors.New(errors.HiScoreNoServer)
		}
		return nil, errors.New(errors.HiScore, err)
	}

	// get authentication details
	authFilePath, err := paths.ResourcePath("", authFile)
	if err != nil {
		return nil, errors.New(errors.HiScore, err)
	}

	af, err := os.Open(authFilePath)
	if err != nil {
		switch err.(type) {
		case *os.PathError:
			return sess, errors.New(errors.HiScoreNoAuthentication)
		}
		return nil, errors.New(errors.HiScore, err)
	}
	defer af.Close()

	_, err = fmt.Fscanf(af, "%s", &sess.authToken)
	if err != nil {
		if err == io.EOF {
			return sess, errors.New(errors.HiScoreNoAuthentication)
		}
		return nil, errors.New(errors.HiScore, err)
	}

	return sess, nil
}

// StartSession notifies the HiScore server that a game is about to start.
func (sess *Session) StartSession(name string, hash string) error {
	values := map[string]string{"name": name, "game_id": hash}
	jsonValue, _ := json.Marshal(values)
	statusCode, response, err := sess.post("/HiScore/rest/game/", jsonValue)
	if err != nil {
		return errors.New(errors.HiScore, err)
	}

	switch statusCode {
	case 200:
		// game is known and session has been started
	case 201:
		// game is new and has been added to the database
	default:
		err = fmt.Errorf("register game: unexpected response from HiScore server [%d: %s]", statusCode, response)
		return errors.New(errors.HiScore, err)
	}

	err = json.Unmarshal(response, &sess.id)
	if err != nil {
		return errors.New(errors.HiScore, err)
	}

	return nil
}

// EndSession notifies the the HiScore server that a game has finished, with
// details of the game session (time spent, score, etc.)
func (sess *Session) EndSession(playTime time.Duration) error {
	values := map[string]interface{}{"session": sess.id, "duration": fmt.Sprintf("%.0f", playTime.Seconds())}
	jsonValue, _ := json.Marshal(values)
	statusCode, response, err := sess.post("/HiScore/rest/play/", jsonValue)
	if err != nil {
		return errors.New(errors.HiScore, err)
	}

	switch statusCode {
	case 201:
		// hiscore has been posted
	default:
		err = fmt.Errorf("register hiscore: unexpected response from HiScore server [%d: %s]", statusCode, response)
		return errors.New(errors.HiScore, err)
	}

	return nil
}

// url should not contain the session server, it will be added automatically
func (sess *Session) post(url string, data []byte) (int, []byte, error) {
	// add server information to url
	url = fmt.Sprintf("%s%s", sess.server, url)

	// prepare POST request
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(data))
	if err != nil {
		return 0, []byte{}, err
	}

	// add authorization head
	req.Header.Add("Authorization", fmt.Sprintf("Token %s", sess.authToken))

	// Send req using http Client
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return 0, []byte{}, err
	}

	// get response
	response, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return resp.StatusCode, []byte{}, err
	}

	return resp.StatusCode, response, nil
}

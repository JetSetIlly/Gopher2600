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

package prefs

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/jetsetilly/gopher2600/errors"
)

// types support by the prefs system must implement the pref interface
type pref interface {
	fmt.Stringer
	Set(value interface{}) error
	Get() interface{}
}

// Bool implements a boolean type in the prefs system.
type Bool struct {
	pref
	value    bool
	callback func(value interface{}) error
}

func (p Bool) String() string {
	return fmt.Sprintf("%v", p.value)
}

// Set new value to Bool type. New value must be of type bool or string. A
// string value of anything other than "true" (case insensitive) will set the
// value to false.
func (p *Bool) Set(v interface{}) error {
	switch v := v.(type) {
	case bool:
		p.value = v
	case string:
		switch strings.ToLower(v) {
		case "true":
			p.value = true
		default:
			p.value = false
		}
	default:
		return errors.New(errors.Prefs, fmt.Sprintf("cannot convert %T to prefs.Bool", v))
	}

	if p.callback != nil {
		return p.callback(p.value)
	}

	return nil
}

// Get returns the raw pref value
func (p Bool) Get() interface{} {
	return p.value
}

// RegisterCallback sets the callback function to be called when the value has
// changed. Not required but is useful in some contexts.
func (p *Bool) RegisterCallback(f func(value interface{}) error) {
	p.callback = f
}

// String implements a string type in the prefs system.
type String struct {
	pref
	value    string
	callback func(value interface{}) error
}

func (p String) String() string {
	return p.value
}

// Set new value to String type. New value must be of type string.
func (p *String) Set(v interface{}) error {
	p.value = fmt.Sprintf("%s", v)

	if p.callback != nil {
		return p.callback(p.value)
	}

	return nil
}

// Get returns the raw pref value
func (p String) Get() interface{} {
	return p.value
}

// RegisterCallback sets the callback function to be called when the value has
// changed. Not required but is useful in some contexts.
func (p *String) RegisterCallback(f func(value interface{}) error) {
	p.callback = f
}

// Int implements a string type in the prefs system.
type Int struct {
	pref
	value    int
	callback func(value interface{}) error
}

func (p Int) String() string {
	return fmt.Sprintf("%d", p.value)
}

// Set new value to Int type. New value can be an int or string.
func (p *Int) Set(v interface{}) error {
	switch v := v.(type) {
	case int:
		p.value = v
	case string:
		var err error
		p.value, err = strconv.Atoi(v)
		if err != nil {
			return errors.New(errors.Prefs, fmt.Sprintf("cannot convert %T to prefs.Int", v))
		}
	default:
		return errors.New(errors.Prefs, fmt.Sprintf("cannot convert %T to prefs.Int", v))
	}

	if p.callback != nil {
		return p.callback(p.value)
	}

	return nil
}

// Get returns the raw pref value
func (p Int) Get() interface{} {
	return p.value
}

// RegisterCallback sets the callback function to be called when the value has
// changed. Not required but is useful in some contexts.
func (p *Int) RegisterCallback(f func(value interface{}) error) {
	p.callback = f
}

// Generic is a general purpose prefererences type, useful for values that
// cannot be represented by a single live value.  You must use the NewGeneric()
// function to initialise a new instance of Generic.
//
// The Generic prefs type does not have a way of registering a callback function.
type Generic struct {
	pref
	set func(string) error
	get func() string
}

// NewGeneric is the preferred method of initialisation for the Generic type.
func NewGeneric(set func(string) error, get func() string) *Generic {
	return &Generic{
		set: set,
		get: get,
	}
}

func (p Generic) String() string {
	return p.get()
}

// Set triggers the set value procedure for the generic type
func (p *Generic) Set(v interface{}) error {
	err := p.set(v.(string))
	if err != nil {
		return err
	}

	return nil
}

// Get triggers the get value procedure for the generic type.
func (p Generic) Get() interface{} {
	return p.get()
}

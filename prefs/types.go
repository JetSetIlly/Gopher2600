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
	"sync"

	"github.com/jetsetilly/gopher2600/curated"
)

// Value represents the actual Go preference value.
type Value interface{}

// types support by the prefs system must implement the pref interface.
type pref interface {
	fmt.Stringer
	Set(value Value) error
	Get() Value
	Reset() error
}

// Bool implements a boolean type in the prefs system.
type Bool struct {
	pref
	crit     sync.Mutex
	value    bool
	callback func(value Value) error
}

func (p *Bool) String() string {
	p.crit.Lock()
	defer p.crit.Unlock()

	return fmt.Sprintf("%v", p.value)
}

// Set new value to Bool type. New value must be of type bool or string. A
// string value of anything other than "true" (case insensitive) will set the
// value to false.
func (p *Bool) Set(v Value) error {
	p.crit.Lock()
	// no defer: we have to be careful when we unlock so as not to call
	// deadlocks in the callback function

	// new value
	var nv bool
	switch v := v.(type) {
	case bool:
		nv = v
	case string:
		switch strings.ToLower(v) {
		case "true":
			nv = true
		default:
			nv = false
		}
	default:
		p.crit.Unlock()
		return curated.Errorf("prefs: %v", fmt.Errorf("cannot convert %T to prefs.Bool", v))
	}

	// value hasn't changed so do not do anything
	if nv == p.value {
		p.crit.Unlock()
		return nil
	}

	// store new value
	p.value = nv

	if p.callback != nil {
		p.crit.Unlock()
		return p.callback(p.value)
	}

	p.crit.Unlock()
	return nil
}

// Get returns the raw pref value.
func (p *Bool) Get() Value {
	p.crit.Lock()
	defer p.crit.Unlock()

	return p.value
}

// Reset sets the boolean value to false.
func (p *Bool) Reset() error {
	return p.Set(false)
}

// RegisterCallback sets the callback function to be called when the value has
// changed. Not required but is useful in some contexts.
func (p *Bool) RegisterCallback(f func(value Value) error) {
	p.crit.Lock()
	defer p.crit.Unlock()

	p.callback = f
}

// String implements a string type in the prefs system.
type String struct {
	pref
	crit     sync.Mutex
	maxLen   int
	value    string
	callback func(value Value) error
}

func (p *String) String() string {
	p.crit.Lock()
	defer p.crit.Unlock()

	return p.value
}

// SetMaxLen sets the maximum length for a string when it is set. To set no
// limit use a value less than or equal to zero. Note that the existing string
// will be cropped if necessary - cropped string information will be lost.
func (p *String) SetMaxLen(max int) {
	p.crit.Lock()
	defer p.crit.Unlock()

	p.maxLen = max

	// crop existing string if necessary
	if p.maxLen > 0 && len(p.value) > p.maxLen {
		p.value = p.value[:p.maxLen]
	}
}

// Set new value to String type. New value must be of type string.
func (p *String) Set(v Value) error {
	p.crit.Lock()
	// no defer: we have to be careful when we unlock so as not to call
	// deadlocks in the callback function

	// new value
	nv := fmt.Sprintf("%s", v)
	if p.maxLen > 0 && len(nv) > p.maxLen {
		nv = nv[:p.maxLen]
	}

	// value hasn't changed so do not do anything
	if nv == p.value {
		p.crit.Unlock()
		return nil
	}

	// store new value
	p.value = nv

	if p.callback != nil {
		p.crit.Unlock()
		return p.callback(p.value)
	}

	p.crit.Unlock()
	return nil
}

// Get returns the raw pref value.
func (p *String) Get() Value {
	p.crit.Lock()
	defer p.crit.Unlock()

	return p.value
}

// Reset sets the string value to the empty string.
func (p *String) Reset() error {
	return p.Set("")
}

// RegisterCallback sets the callback function to be called when the value has
// changed. Not required but is useful in some contexts.
func (p *String) RegisterCallback(f func(value Value) error) {
	p.crit.Lock()
	defer p.crit.Unlock()

	p.callback = f
}

// Int implements a string type in the prefs system.
type Int struct {
	pref
	crit     sync.Mutex
	value    int
	callback func(value Value) error
}

func (p *Int) String() string {
	p.crit.Lock()
	defer p.crit.Unlock()

	return fmt.Sprintf("%d", p.value)
}

// Set new value to Int type. New value can be an int or string.
func (p *Int) Set(v Value) error {
	p.crit.Lock()
	// no defer: we have to be careful when we unlock so as not to call
	// deadlocks in the callback function

	// new value
	var nv int
	switch v := v.(type) {
	case int64:
		nv = int(v)
	case int32:
		nv = int(v)
	case int:
		nv = v
	case string:
		var err error
		nv, err = strconv.Atoi(v)
		if err != nil {
			p.crit.Unlock()
			return curated.Errorf("prefs: %v", fmt.Errorf("cannot convert %T to prefs.Int", v))
		}
	default:
		p.crit.Unlock()
		return curated.Errorf("prefs: %v", fmt.Errorf("cannot convert %T to prefs.Int", v))
	}

	// value hasn't changed so do not do anything
	if nv == p.value {
		p.crit.Unlock()
		return nil
	}

	// update stored value
	p.value = nv

	if p.callback != nil {
		p.crit.Unlock()
		return p.callback(p.value)
	}

	p.crit.Unlock()
	return nil
}

// Get returns the raw pref value.
func (p *Int) Get() Value {
	p.crit.Lock()
	defer p.crit.Unlock()

	return p.value
}

// Reset sets the int value to zero.
func (p *Int) Reset() error {
	return p.Set(0)
}

// RegisterCallback sets the callback function to be called when the value has
// changed. Not required but is useful in some contexts.
func (p *Int) RegisterCallback(f func(value Value) error) {
	p.crit.Lock()
	defer p.crit.Unlock()

	p.callback = f
}

// Int implements a string type in the prefs system.
type Float struct {
	pref
	crit     sync.Mutex
	value    float64
	callback func(value Value) error
}

func (p *Float) String() string {
	p.crit.Lock()
	defer p.crit.Unlock()

	return fmt.Sprintf("%.2f", p.value)
}

// Set new value to Int type. New value can be an int or string.
func (p *Float) Set(v Value) error {
	p.crit.Lock()
	// no defer: we have to be careful when we unlock so as not to call
	// deadlocks in the callback function

	// new value
	var nv float64
	switch v := v.(type) {
	case float64:
		nv = v
	case float32:
		nv = float64(v)
	case int:
		nv = float64(v)
	case string:
		var err error
		nv, err = strconv.ParseFloat(v, 64)
		if err != nil {
			p.crit.Unlock()
			return curated.Errorf("prefs: %v", fmt.Errorf("cannot convert %T to prefs.Float", v))
		}
	default:
		p.crit.Unlock()
		return curated.Errorf("prefs: %v", fmt.Errorf("cannot convert %T to prefs.Float", v))
	}

	// value hasn't changed so do not do anything
	if nv == p.value {
		p.crit.Unlock()
		return nil
	}

	// update stored value
	p.value = nv

	if p.callback != nil {
		p.crit.Unlock()
		return p.callback(p.value)
	}

	p.crit.Unlock()
	return nil
}

// Get returns the raw pref value.
func (p *Float) Get() Value {
	p.crit.Lock()
	defer p.crit.Unlock()

	return p.value
}

// Reset sets the int value to zero.
func (p *Float) Reset() error {
	return p.Set(0.0)
}

// RegisterCallback sets the callback function to be called when the value has
// changed. Not required but is useful in some contexts.
func (p *Float) RegisterCallback(f func(value Value) error) {
	p.crit.Lock()
	defer p.crit.Unlock()

	p.callback = f
}

// Generic is a general purpose prefererences type, useful for values that
// cannot be represented by a single live value.  You must use the NewGeneric()
// function to initialise a new instance of Generic.
//
// The Generic prefs type does not have a way of registering a callback function.
type Generic struct {
	pref
	crit sync.Mutex
	set  func(string) error
	get  func() string
}

// NewGeneric is the preferred method of initialisation for the Generic type.
func NewGeneric(set func(string) error, get func() string) *Generic {
	return &Generic{
		set: set,
		get: get,
	}
}

func (p *Generic) String() string {
	p.crit.Lock()
	defer p.crit.Unlock()

	return p.get()
}

// Set triggers the set value procedure for the generic type.
func (p *Generic) Set(v Value) error {
	p.crit.Lock()
	defer p.crit.Unlock()

	err := p.set(v.(string))
	if err != nil {
		return err
	}

	return nil
}

// Get triggers the get value procedure for the generic type.
func (p *Generic) Get() Value {
	p.crit.Lock()
	defer p.crit.Unlock()

	return p.get()
}

// Reset sets the generic value to the empty string.
func (p *Generic) Reset() error {
	return p.Set("")
}

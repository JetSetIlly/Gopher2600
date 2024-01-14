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
	"sync/atomic"
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
	value    atomic.Value // bool
	hookPre  func(value Value) error
	hookPost func(value Value) error
}

func (p *Bool) String() string {
	ov := p.value.Load()
	if ov == nil {
		return "false"
	}
	return fmt.Sprintf("%v", ov.(bool))
}

// Set new value to Bool type. New value must be of type bool or string. A
// string value of anything other than "true" (case insensitive) will set the
// value to false.
func (p *Bool) Set(v Value) error {
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
		return fmt.Errorf("set: cannot convert %T to prefs.Bool", v)
	}

	if p.hookPre != nil {
		err := p.hookPre(nv)
		if err != nil {
			return err
		}
	}

	// store new value
	p.value.Store(nv)

	if p.hookPost != nil {
		err := p.hookPost(nv)
		if err != nil {
			return err
		}
	}

	return nil
}

// Get returns the raw pref value.
func (p *Bool) Get() Value {
	ov := p.value.Load()
	if ov == nil {
		return false
	}
	return ov.(bool)
}

// Reset sets the boolean value to false.
func (p *Bool) Reset() error {
	return p.Set(false)
}

// SetHookPre sets the callback function to be called just before the prefs
// value is updated. Note that even if the value hasn't changed, the callback
// will be executed.
//
// Not required but is useful in some contexts.
func (p *Bool) SetHookPre(f func(value Value) error) {
	p.hookPre = f
}

// SetHookPost sets the callback function to be called just after the prefs
// value is updated. Note that even if the value hasn't changed, the callback
// will be executed.
//
// Not required but is useful in some contexts.
func (p *Bool) SetHookPost(f func(value Value) error) {
	p.hookPost = f
}

// String implements a string type in the prefs system.
type String struct {
	pref
	maxLen   int
	value    atomic.Value // string
	hookPre  func(value Value) error
	hookPost func(value Value) error
}

func (p *String) String() string {
	ov := p.value.Load()
	if ov == nil {
		return ""
	}
	return ov.(string)
}

// SetMaxLen sets the maximum length for a string when it is set. To set no
// limit use a value less than or equal to zero. Note that the existing string
// will be cropped if necessary - cropped string information will be lost.
func (p *String) SetMaxLen(max int) {
	p.maxLen = max

	// crop existing string if necessary

	ov := p.value.Load()
	if ov == nil {
		// no need to crop string
		return
	}

	if p.maxLen > 0 && len(ov.(string)) > p.maxLen {
		p.value.Store(ov.(string)[:p.maxLen])
	}
}

// Set new value to String type. New value must be of type string.
func (p *String) Set(v Value) error {
	// new value
	nv := fmt.Sprintf("%s", v)
	if p.maxLen > 0 && len(nv) > p.maxLen {
		nv = nv[:p.maxLen]
	}

	if p.hookPre != nil {
		err := p.hookPre(nv)
		if err != nil {
			return err
		}
	}

	// store new value
	p.value.Store(nv)

	if p.hookPost != nil {
		err := p.hookPost(nv)
		if err != nil {
			return err
		}
	}

	return nil
}

// Get returns the raw pref value.
func (p *String) Get() Value {
	return p.String()
}

// Reset sets the string value to the empty string.
func (p *String) Reset() error {
	return p.Set("")
}

// SetHookPre sets the callback function to be called just before the prefs
// value is updated. Note that even if the value hasn't changed, the callback
// will be executed.
//
// Not required but is useful in some contexts.
func (p *String) SetHookPre(f func(value Value) error) {
	p.hookPre = f
}

// SetHookPost sets the callback function to be called just after the prefs
// value is updated. Note that even if the value hasn't changed, the callback
// will be executed.
//
// Not required but is useful in some contexts.
func (p *String) SetHookPost(f func(value Value) error) {
	p.hookPost = f
}

// Int implements a string type in the prefs system.
type Int struct {
	pref
	value    atomic.Value // int
	hookPre  func(value Value) error
	hookPost func(value Value) error
}

func (p *Int) String() string {
	ov := p.value.Load()
	if ov == nil {
		return "0"
	}
	return fmt.Sprintf("%d", ov.(int))
}

// Set new value to Int type. New value can be an int or string.
func (p *Int) Set(v Value) error {
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
			return fmt.Errorf("set: cannot convert %T to prefs.Int: %w", v, err)
		}
	default:
		return fmt.Errorf("set: cannot convert %T to prefs.Int", v)
	}

	if p.hookPre != nil {
		err := p.hookPre(nv)
		if err != nil {
			return err
		}
	}

	// update stored value
	p.value.Store(nv)

	if p.hookPost != nil {
		err := p.hookPost(nv)
		if err != nil {
			return err
		}
	}

	return nil
}

// Get returns the raw pref value.
func (p *Int) Get() Value {
	ov := p.value.Load()
	if ov == nil {
		return 0
	}
	return ov.(int)
}

// Reset sets the int value to zero.
func (p *Int) Reset() error {
	return p.Set(0)
}

// SetHookPre sets the callback function to be called just before the prefs
// value is updated. Note that even if the value hasn't changed, the callback
// will be executed.
//
// Not required but is useful in some contexts.
func (p *Int) SetHookPre(f func(value Value) error) {
	p.hookPre = f
}

// SetHookPost sets the callback function to be called just after the prefs
// value is updated. Note that even if the value hasn't changed, the callback
// will be executed.
//
// Not required but is useful in some contexts.
func (p *Int) SetHookPost(f func(value Value) error) {
	p.hookPost = f
}

// Int implements a string type in the prefs system.
type Float struct {
	pref
	value    atomic.Value // float64
	hookPre  func(value Value) error
	hookPost func(value Value) error
}

func (p *Float) String() string {
	ov := p.value.Load()
	if ov == nil {
		return "0.000"
	}
	return fmt.Sprintf("%.3f", ov.(float64))
}

// Set new value to Int type. New value can be an int or string.
func (p *Float) Set(v Value) error {
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
			return fmt.Errorf("set: cannot convert %T to prefs.Float: %w", v, err)
		}
	default:
		return fmt.Errorf("set: cannot convert %T to prefs.Float", v)
	}

	if p.hookPre != nil {
		err := p.hookPre(nv)
		if err != nil {
			return err
		}
	}

	// update stored value
	p.value.Store(nv)

	if p.hookPost != nil {
		err := p.hookPost(nv)
		if err != nil {
			return err
		}
	}

	return nil
}

// Get returns the raw pref value.
func (p *Float) Get() Value {
	ov := p.value.Load()
	if ov == nil {
		return float64(0.0)
	}
	return ov.(float64)
}

// Reset sets the int value to zero.
func (p *Float) Reset() error {
	return p.Set(0.0)
}

// SetHookPre sets the callback function to be called just before the prefs
// value is updated. Note that even if the value hasn't changed, the callback
// will be executed.
//
// Not required but is useful in some contexts.
func (p *Float) SetHookPre(f func(value Value) error) {
	p.hookPre = f
}

// SetHookPost sets the callback function to be called just after the prefs
// value is updated. Note that even if the value hasn't changed, the callback
// will be executed.
//
// Not required but is useful in some contexts.
func (p *Float) SetHookPost(f func(value Value) error) {
	p.hookPost = f
}

// Generic is a general purpose prefererences type, useful for values that
// cannot be represented by a single live value.  You must use the NewGeneric()
// function to initialise a new instance of Generic.
//
// The Generic prefs type does not have a way of registering a callback
// function. It is also slower than other prefs types because it must protect
// potential critical sections with a mutex (other types can use an atomic
// value).
type Generic struct {
	pref
	crit sync.Mutex
	set  func(Value) error
	get  func() Value

	// the last value sent to set() function
	mostRecentSetValue Value
}

// GenericGetValueUndefined is a special return value for the get() function
// (see NewGeneric()). It indicates that the value is currently unavailable and
// the most recent previous value should be used.
const GenericGetValueUndefined = "GenericGetValueUndefined"

// NewGeneric is the preferred method of initialisation for the Generic type.
func NewGeneric(set func(Value) error, get func() Value) *Generic {
	return &Generic{
		set: set,
		get: get,
	}
}

func (p *Generic) String() string {
	return fmt.Sprintf("%v", p.Get())
}

// Set triggers the set value procedure for the generic type.
func (p *Generic) Set(v Value) error {
	p.crit.Lock()
	defer p.crit.Unlock()

	p.mostRecentSetValue = v.(string)

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

	s := p.get()
	if s == GenericGetValueUndefined {
		s = p.mostRecentSetValue
	} else {
		p.mostRecentSetValue = s
	}

	return s
}

// Reset sets the generic value to the empty string.
func (p *Generic) Reset() error {
	return p.Set("")
}

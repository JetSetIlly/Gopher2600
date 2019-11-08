package gui

import (
	"gopher2600/television"
)

// GUI defines the operations that can be performed on visual user interfaces.
//
// Currently, GUI implementations expect also to be an instance of
// television.Television. This way a single object can be used in both GUI and
// television contexts. In practice, the GUI instance may also implement the
// Renderer and AudioMixer interfaces from the television packages but this is
// not mandated by the GUI interface.
type GUI interface {
	television.Television

	// All GUIs should implement a MetaPixelRenderer even if only as a stub
	MetaPixelRenderer

	// returns true if GUI is currently visible. false if not
	IsVisible() bool

	// send a request to set a gui feature
	SetFeature(request FeatureReq, args ...interface{}) error

	// the event channel is used to by the GUI implementation to send
	// information back to the main program. the GUI may or may not be in its
	// own go routine but in regardless, the event channel is used for this
	// purpose.
	SetEventChannel(chan (Event))
}

// FeatureReq is used to request the setting of a gui attribute
// eg. toggling the overlay
type FeatureReq int

// list of valid feature requests. argument must be of the type specified or
// else the interface{} type conversion will fail and the application will
// probably crash
const (
	ReqSetVisibility       FeatureReq = iota // bool, optional bool (update on show) default true
	ReqToggleVisibility                      // optional bool (update on show) default true
	ReqSetVisibilityStable                   // none
	ReqSetPause                              // bool
	ReqSetMasking                            // bool
	ReqToggleMasking                         // none
	ReqSetAltColors                          // bool
	ReqToggleAltColors                       // none
	ReqSetOverlay                            // bool
	ReqToggleOverlay                         // none
	ReqSetScale                              // float
	ReqIncScale                              // none
	ReqDecScale                              // none
)

// EventID idintifies the type of event taking place
type EventID int

// list of valid events
const (
	EventWindowClose EventID = iota
	EventKeyboard
	EventMouseLeft
	EventMouseRight
)

// KeyMod identifies
type KeyMod int

// list of valud key modifiers
const (
	KeyModNone KeyMod = iota
	KeyModShift
	KeyModCtrl
	KeyModAlt
)

// EventData represents the data that is associated with an event
type EventData interface{}

// Event is the structure that is passed over the event channel
type Event struct {
	ID   EventID
	Data EventData
}

// EventDataKeyboard is the data that accompanies EvenKeyboard events
type EventDataKeyboard struct {
	Key  string
	Down bool
	Mod  KeyMod
}

// EventDataMouse is the data that accompanies EventMouse events
type EventDataMouse struct {
	Down     bool
	X        int
	Y        int
	HorizPos int
	Scanline int
}

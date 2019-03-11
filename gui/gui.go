package gui

import "gopher2600/television"

// MetaStateReq is used to identify what information is being requested with the
// with the GetMetaState() function
type MetaStateReq int

// CallbackReq is used to identify which callback to register
type CallbackReq int

// FeatureReq is used to request the setting of a gui attribute
// eg. setting debugging overscan
type FeatureReq int

// list of valid metastate requests
const (
	ReqLastKeyboard MetaStateReq = iota
	ReqLastMouse
	ReqLastMouseHorizPos
	ReqLastMouseScanline
)

// list of valid callback requests
const (
	ReqOnWindowClose CallbackReq = iota
	ReqOnKeyboard
	ReqOnMouseButtonLeft
	ReqOnMouseButtonRight
)

// list of valid feature requests
const (
	ReqSetVisibility         FeatureReq = iota // bool, optional bool (update on show)
	ReqSetVisibilityStable                     // none
	ReqSetAllowDebugging                       // bool
	ReqSetPause                                // bool
	ReqSetMasking                              // bool
	ReqToggleMasking                           // none
	ReqSetAltColors                            // bool
	ReqToggleAltColors                         // none
	ReqSetShowSystemState                      // bool
	ReqToggleShowSystemState                   // none
	ReqSetScale                                // float
	ReqIncScale                                // none
	ReqDecScale                                // none
)

// GUI defines the operations that can be performed on GUIs
type GUI interface {
	television.Television
	GetMetaState(MetaStateReq) (interface{}, error)
	RegisterCallback(CallbackReq, chan func(), func()) error
	SetFeature(request FeatureReq, args ...interface{}) error
}

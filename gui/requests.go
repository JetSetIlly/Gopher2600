package gui

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

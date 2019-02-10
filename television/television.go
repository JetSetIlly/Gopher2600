package television

// StateReq is used to identify which television attribute is being asked
// with the GetState() function
type StateReq string

// MetaStateReq is used to identify what information is being requested with the
// with the GetMetaState() function
type MetaStateReq string

// CallbackReq is used to identify which callback to register
type CallbackReq string

// FeatureReq is used to request the setting of a television attribute
// eg. setting debugging overscan
type FeatureReq string

// list of valid requests for television implementations. it is not
// required that every implementation does something useful for every request.
// for instance, ONWINDOWCLOSE is meaningless if the implementation has no
// display window
const (
	ReqFramenum StateReq = "FRAME"
	ReqScanline StateReq = "SCANLINE"
	ReqHorizPos StateReq = "HORIZPOS"

	ReqTVSpec            MetaStateReq = "TVSPEC"
	ReqLastMouse         MetaStateReq = "MOUSE"
	ReqLastMouseHorizPos MetaStateReq = "MOUSEHORIZPOS"
	ReqLastMouseScanline MetaStateReq = "MOUSESCANLINE"

	ReqOnWindowClose      CallbackReq = "ONWINDOWCLOSE"
	ReqOnMouseButtonLeft  CallbackReq = "ONMOUSEBUTTONLEFT"
	ReqOnMouseButtonRight CallbackReq = "ONMOUSEBUTTONRIGHT"

	ReqSetVisibility       FeatureReq = "SETVISIBILITY"           // bool, optional bool (update on show)
	ReqSetVisibilityStable FeatureReq = "SETVISIBILITYWHENSTABLE" // none
	ReqSetPause            FeatureReq = "SETPAUSE"                // bool
	ReqSetDebug            FeatureReq = "SETDEBUG"                // bool
	ReqSetScale            FeatureReq = "SETSCALE"                // float
)

// SignalAttributes represents the data sent to the television
type SignalAttributes struct {
	VSync, VBlank, FrontPorch, HSync, CBurst bool
	Pixel                                    ColorSignal
}

// Television defines the operations that can be performed on the television
type Television interface {
	MachineInfoTerse() string
	MachineInfo() string

	Reset() error
	Signal(SignalAttributes) error

	GetState(StateReq) (TVState, error)
	GetMetaState(MetaStateReq) (string, error)
	RegisterCallback(CallbackReq, chan func(), func()) error
	SetFeature(request FeatureReq, args ...interface{}) error
}

package television

// TVStateReq is used to identify which television attribute is being asked
// for with the GetTVState() function
type TVStateReq string

// TVInfoReq is used to identiry what information is being requested with the
// GetTVInfo() function
type TVInfoReq string

// CallbackReq is used to identify which callback to register
type CallbackReq string

// SetAttrReq is used to request the setting of a television attribute
// eg. setting debugging overscan
type SetAttrReq string

// list of valid requests for television implementations. it is not
// required that every implementation does something useful for every request.
// for instance, ONWINDOWCLOSE is meaningless if the implementation has no
// display window
const (
	ReqFramenum TVStateReq = "FRAME"
	ReqScanline TVStateReq = "SCANLINE"
	ReqHorizPos TVStateReq = "HORIZPOS"

	ReqTVSpec     TVInfoReq = "TVSPEC"
	ReqLastMouse  TVInfoReq = "MOUSE"
	ReqLastMouseX TVInfoReq = "MOUSEX"
	ReqLastMouseY TVInfoReq = "MOUSEY"

	ReqOnWindowClose      CallbackReq = "ONWINDOWCLOSE"
	ReqOnMouseButtonLeft  CallbackReq = "ONMOUSEBUTTONLEFT"
	ReqOnMouseButtonRight CallbackReq = "ONMOUSEBUTTONRIGHT"

	ReqSetVisibility SetAttrReq = "SETVISIBILITY" // bool
	ReqSetPause      SetAttrReq = "SETPAUSE"      // bool
	ReqSetDebug      SetAttrReq = "SETDEBUG"      // bool
	ReqSetScale      SetAttrReq = "SETSCALE"      // float
)

// SignalAttributes represents the data sent to the television
type SignalAttributes struct {
	VSync, VBlank, FrontPorch, HSync, CBurst bool
	Pixel                                    PixelSignal
}

// Television defines the operations that can be performed on the television
type Television interface {
	MachineInfoTerse() string
	MachineInfo() string

	Signal(SignalAttributes)

	RequestTVState(TVStateReq) (*TVState, error)
	RequestTVInfo(TVInfoReq) (string, error)
	RequestCallbackRegistration(CallbackReq, chan func(), func()) error
	RequestSetAttr(request SetAttrReq, args ...interface{}) error
}

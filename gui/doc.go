// Package gui is an abstraction layer for real GUI implementations. It defines
// the Events that can be passed from the GUI to the emulation code and also
// the Requests that can be made from the emulation code to the GUI.
// Implementations need to convert their specific signals and requests to and
// from these abstractions.
//
// It also introduces the idea of metapixels. Metapixels can be thought of as
// supplementary signals to the underlying television. The GUI can then present
// the metapixels as an overlay. The ReqToggleOverlay request is intended for
// this.
package gui

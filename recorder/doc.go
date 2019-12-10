// Package recorder handles recording and playback of user input. The Recorder
// type implements the peripheraps.Transcriber() interface. Once added as a
// transcriber to the VCS port, it will record all user input to the specified
// file.
//
// Currently, a recorder and playback is attached to all ports of the VCS,
// including the panel. Future versions may allow which ports are to be
// recording to be definable.
package recorder

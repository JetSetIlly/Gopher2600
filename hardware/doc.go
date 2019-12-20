// Package hardware is the base package for the VCS emulation. It and its
// sub-package contain everything required for a headless emulation.
//
// The VCS type is the root of the emulation and contains external references
// to all the VCS sub-systems. From here, the emulation can either be started
// to run continuously (with optional callback to check for continuation); or
// it can be stepped cycle by cycle. Both CPU and video cycle stepping are
// supported.
package hardware


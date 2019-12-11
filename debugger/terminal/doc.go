// Package terminal defines the operations required for user interaction with
// the debugger. While an implementation of the GUI interface, found in the GUI
// package, may allow some interaction with the debugger (eg. visually setting
// a breakpoint) the principle means of interaction with the debugger is the
// terminal.
//
// For flexibility, terminal interaction happens through the Terminal Go
// interface. There are two implementations of this interface: the
// PlainTerminal and the ColorTerminal, found respectively in the plainterm and
// colorterm sub-packages. We should think of these implementations as
// reference implementions. Other implementations of the Terminal interface
// should probably be placed somewhere else.
package terminal

// Package setup is used to preset the emulation depending on the attached
// cartridge.
//
// This package is not yet complete. It currently only supports panel setup.
// ie. the setting of the switches on the frontpanel.
//
// Other setup option idea: POKEs. For example, bug fixing the ET cartridge on
// startup.
//
// Eventually we would probably require a menu driven selection of setups. In
// other words, a cartridge is loaded and there but there are several setup
// options to choose from (eg. bug-fixed or original ROM)
//
// The setup pacakge currently doesn't facilitate editing of the setup
// database, only reading.
package setup

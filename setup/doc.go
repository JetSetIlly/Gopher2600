// Package setup is used to preset the emulation depending on the attached
// cartridge. It is currently quite limited but is useful none-the-less.
// Currently support entry types:
//
//	Toggling of panel switches
//	Apply patches to cartridge
//
// Menu driven selection of patches would be a nice feature to have in the
// future. But at the moment, the package doesn't even facilitate editing of
// entries. Adding new entries to the setup database therefore requires editing
// the DB file by hand. For reference the following describes the format of
// each entry type:
//
//	Panel Toggles
//
//	<DB Key>, panel, <SHA-1 Hash>, <player 0 (bool)>, .<player 1 (bool)>, <color (bool)>, <notes>
//
// When editing the DB file, make sure the DB Key is unique
//
//	Patch Cartridge
//
//	<DB Key>, patch, <SHA-1 Hash>, <patch file>, <notes>
//
// Patch files are located in the patches sub-directory of the resources path.
package setup

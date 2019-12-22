// Package paths contains functions to prepare paths to gopher2600 resources.
//
// The ResourcePath() function modifies the supplied resource string such that
// it is prepended with the appropriate config directory. For example, the
// following will return the path to a cartridge patch.
//
//	d := paths.ResourcePath("patches", "ET")
//
// The policy of ResourcePath() is simple: if the base resource path, currently
// defined to be ".gopher2600", is present in the program's current directory
// then that is the base path that will used. If it is not preseent not, then
// the user's config directory is used. The package uses os.UserConfigDir()
// from go standard library for this.
//
// In the example above, on a modern Linux system, the path returned will be:
//
//	/home/user/.config/gopher2600/patches/ET
package paths

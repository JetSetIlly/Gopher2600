// Package paths should be used whenever a request to the filesystem is
// made. The functions herein make sure that the correct path (depending on the
// operating system being targeted) is used for the resource.
//
// Because this package handles project specific details it should be used
// instead of the Go standard path package
package paths

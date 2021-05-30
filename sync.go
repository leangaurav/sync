// This package provides some features available in golang's sync package with some enhancements.
//
// Once
//
// The Once defined by this package is a stateful implementation which can be passed around to other fuctions.
// It adds additional features to test whether the operations have already been Done and also allows Reset.
// These features especially the stateful nature makes it useful at multiple places where it needs to be shared by go-routines.
//
// Once is concurrency safe and has well defined behavior for concurrent access. See the test cases.
package sync

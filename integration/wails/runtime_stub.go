//go:build !linux && !windows && !darwin
// +build !linux,!windows,!darwin

package wails

import "fmt"

// Window is a stub for unsupported platforms.
type Window struct{}

// Runtime is a stub for unsupported platforms.
type Runtime struct{}

// DefaultWindowOptions returns a default stub.
func DefaultWindowOptions() WindowOptions { return WindowOptions{} }

// Initialize returns an error for unsupported platforms.
func Initialize(args []string) (*Runtime, int, error) {
	_ = args
	return nil, -1, fmt.Errorf("wails integration runtime is currently implemented for linux, windows, and darwin only")
}

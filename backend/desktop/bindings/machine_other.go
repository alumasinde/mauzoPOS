//go:build !windows

package bindings

func readMachineGUID() string { return "" }

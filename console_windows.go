package main

import "golang.org/x/sys/windows"

func setupWindowsConsole(stdoutFd int) error {
	console := windows.Handle(stdoutFd)
	var originalMode uint32
	windows.GetConsoleMode(console, &originalMode)
	return windows.SetConsoleMode(console, originalMode|windows.ENABLE_VIRTUAL_TERMINAL_PROCESSING)
}

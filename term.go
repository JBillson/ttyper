package main

import (
	"os"

	"golang.org/x/term"
)

func enableRawMode() (func(), error) {
	fd := int(os.Stdin.Fd())
	oldState, err := term.MakeRaw(fd)
	if err != nil {
		return nil, err
	}
	return func() { term.Restore(fd, oldState) }, nil
}

func getTermSize() (width, height int) {
	w, h, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		return 80, 24
	}
	return w, h
}

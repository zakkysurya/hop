package main

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"
)

func SSHConnect(h Host, path string) error {
	defer resetTerminalTitle()

	addr := fmt.Sprintf("%s@%s", h.User, h.Host)
	port := fmt.Sprintf("%d", h.Port)

	cmd := exec.Command("ssh", "-t", "-p", port, addr, fmt.Sprintf("cd '%s' && exec bash -l", path))
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			if status, ok := exitErr.Sys().(syscall.WaitStatus); ok {
				resetTerminalTitle()
				os.Exit(status.ExitStatus())
			}
		}
		return err
	}
	return nil
}

func resetTerminalTitle() {
	fmt.Print("\033]0;\007")
}

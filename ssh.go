package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"
)

func withSpinner(label string, fn func() error) (time.Duration, error) {
	done := make(chan struct{})
	var elapsed time.Duration
	start := time.Now()

	go func() {
		frames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
		i := 0
		ticker := time.NewTicker(80 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				fmt.Printf("\r%s %s", frames[i%len(frames)], label)
				i++
			}
		}
	}()

	err := fn()
	elapsed = time.Since(start)
	close(done)
	fmt.Printf("\r\033[K")
	return elapsed, err
}

func controlPath(h Host) string {
	return fmt.Sprintf("/tmp/hop-cm-%s-%d.sock", h.Host, h.Port)
}

func testConnection(h Host) error {
	cp := controlPath(h)
	addr := fmt.Sprintf("%s@%s", h.User, h.Host)
	port := fmt.Sprintf("%d", h.Port)

	cmd := exec.Command("ssh",
		"-T",
		"-o", "ConnectTimeout=10",
		"-o", "ControlMaster=auto",
		"-o", "ControlPath="+cp,
		"-o", "ControlPersist=30",
		"-p", port, addr, "echo ok")

	out, err := cmd.CombinedOutput()
	if err != nil {
		return parseSSHError(string(out), err)
	}
	return nil
}

func parseSSHError(output string, err error) error {
	lower := strings.ToLower(output)
	switch {
	case strings.Contains(lower, "connection refused"):
		return fmt.Errorf("koneksi ditolak — pastikan IP/port benar dan service SSH aktif di server")
	case strings.Contains(lower, "permission denied"):
		return fmt.Errorf("autentikasi gagal — periksa user/SSH key Anda")
	case strings.Contains(lower, "could not resolve hostname"):
		return fmt.Errorf("host tidak ditemukan — periksa alamat IP/hostname")
	case strings.Contains(lower, "timed out") || strings.Contains(lower, "timeout"):
		return fmt.Errorf("koneksi timeout — server tidak merespon, cek jaringan Anda")
	case strings.Contains(lower, "no route to host"):
		return fmt.Errorf("tidak ada rute ke host — cek koneksi jaringan/VPN Anda")
	default:
		return fmt.Errorf("gagal terhubung: %s", strings.TrimSpace(output))
	}
}

func checkPathExists(h Host, path string) (bool, error) {
	cp := controlPath(h)
	addr := fmt.Sprintf("%s@%s", h.User, h.Host)
	port := fmt.Sprintf("%d", h.Port)

	cmd := exec.Command("ssh",
		"-T",
		"-o", "ControlPath="+cp,
		"-p", port, addr,
		fmt.Sprintf("test -d '%s' && echo FOUND || echo MISSING", path))

	out, err := cmd.CombinedOutput()
	if err != nil {
		return false, parseSSHError(string(out), err)
	}
	return strings.Contains(string(out), "FOUND"), nil
}

func SSHConnect(h Host, path string, command string) error {
	defer resetTerminalTitle()
	defer closeControlMaster(h)

	elapsed, err := withSpinner(fmt.Sprintf("Menghubungkan ke %s (%s)...", h.Alias, h.Host), func() error {
		return testConnection(h)
	})
	if err != nil {
		fmt.Printf("✗ Gagal terhubung ke '%s': %v\n", h.Alias, err)
		return nil
	}
	fmt.Printf("✓ Terhubung ke %s dalam %dms\n", h.Alias, elapsed.Milliseconds())

	finalPath := path
	_, err = withSpinner(fmt.Sprintf("Memeriksa direktori '%s'...", path), func() error {
		found, checkErr := checkPathExists(h, path)
		if checkErr != nil {
			return checkErr
		}
		if !found {
			finalPath = ""
		}
		return nil
	})
	if err != nil {
		fmt.Printf("⚠ Tidak bisa memeriksa direktori: %v\n", err)
		finalPath = ""
	} else if finalPath == "" && path != "" {
		fmt.Printf("⚠ Direktori '%s' tidak ditemukan di server. Masuk ke direktori default.\n", path)
	}

	var remoteCmd string
	switch {
	case finalPath != "" && command != "":
		remoteCmd = fmt.Sprintf("cd '%s' && %s; exec bash -l", finalPath, command)
	case finalPath != "":
		remoteCmd = fmt.Sprintf("cd '%s' && exec bash -l", finalPath)
	default:
		remoteCmd = "exec bash -l"
	}

	if finalPath != "" && command != "" {
		fmt.Printf("→ Menjalankan: %s\n", command)
	}

	addr := fmt.Sprintf("%s@%s", h.User, h.Host)
	port := fmt.Sprintf("%d", h.Port)
	cp := controlPath(h)

	cmd := exec.Command("ssh", "-t", "-o", "ControlPath="+cp, "-p", port, addr, remoteCmd)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			if status, ok := exitErr.Sys().(syscall.WaitStatus); ok {
				resetTerminalTitle()
				closeControlMaster(h)
				os.Exit(status.ExitStatus())
			}
		}
		return err
	}
	return nil
}

func closeControlMaster(h Host) {
	cp := controlPath(h)
	addr := fmt.Sprintf("%s@%s", h.User, h.Host)
	port := fmt.Sprintf("%d", h.Port)
	exec.Command("ssh", "-T", "-o", "ControlPath="+cp, "-p", port, addr, "-O", "exit").Run()
}

func resetTerminalTitle() {
	fmt.Print("\033]0;\007")
}

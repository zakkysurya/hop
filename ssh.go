package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"
)

func baseSSHArgs(h Host) []string {
	args := []string{"-p", fmt.Sprintf("%d", h.Port)}
	if h.IdentityFile != "" {
		args = append(args, "-i", h.IdentityFile, "-o", "IdentitiesOnly=yes")
	} else if h.Password != "" {
		args = append(args, "-o", "PubkeyAuthentication=no", "-o", "PasswordAuthentication=yes")
	}
	return args
}

func keyOnlySSHArgs(h Host) []string {
	args := []string{"-p", fmt.Sprintf("%d", h.Port)}
	if h.IdentityFile != "" {
		args = append(args, "-i", h.IdentityFile, "-o", "IdentitiesOnly=yes")
	}
	if h.Password != "" {
		args = append(args, "-o", "PasswordAuthentication=no")
	}
	return args
}

func passwordSSHArgs(h Host) []string {
	args := []string{"-p", fmt.Sprintf("%d", h.Port)}
	if h.Password != "" {
		args = append(args, "-o", "PubkeyAuthentication=no", "-o", "PasswordAuthentication=yes")
	}
	if h.IdentityFile != "" {
		args = append(args, "-i", h.IdentityFile)
	}
	return args
}

func hostAddr(h Host) string {
	return fmt.Sprintf("%s@%s", h.User, h.Host)
}

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
	if h.IdentityFile != "" {
		args := keyOnlySSHArgs(h)
		cp := controlPath(h)
		args = append(args,
			"-T",
			"-o", "ConnectTimeout=10",
			"-o", "ControlMaster=auto",
			"-o", "ControlPath="+cp,
			"-o", "ControlPersist=30",
			hostAddr(h), "echo ok")
		cmd := exec.Command("ssh", args...)
		out, err := cmd.CombinedOutput()
		if err == nil {
			return nil
		}
		if h.Password != "" {
			fmt.Printf("\n⚠ Gagal menggunakan SSH key, mencoba autentikasi dengan password...\n")
		} else {
			return parseSSHError(string(out), err)
		}
	}

	if h.Password != "" {
		args := passwordSSHArgs(h)
		args = append(args,
			"-T",
			"-o", "ConnectTimeout=10",
			hostAddr(h), "echo ok")
		cmd := exec.Command("ssh", args...)
		out, err := cmd.CombinedOutput()
		if err != nil {
			if h.IdentityFile != "" {
				return fmt.Errorf("autentikasi gagal — baik SSH key maupun password ditolak server: %s", strings.TrimSpace(string(out)))
			}
			return parseSSHError(string(out), err)
		}
		return nil
	}

	args := baseSSHArgs(h)
	cp := controlPath(h)
	args = append(args,
		"-T",
		"-o", "ConnectTimeout=10",
		"-o", "ControlMaster=auto",
		"-o", "ControlPath="+cp,
		"-o", "ControlPersist=30",
		hostAddr(h), "echo ok")
	cmd := exec.Command("ssh", args...)
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
	args := baseSSHArgs(h)
	args = append(args,
		"-T",
		"-o", "ControlPath="+cp,
		hostAddr(h),
		fmt.Sprintf("test -d '%s' && echo FOUND || echo MISSING", path))

	cmd := exec.Command("ssh", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return false, parseSSHError(string(out), err)
	}
	return strings.Contains(string(out), "FOUND"), nil
}

func SSHConnect(h Host, path string, command string) error {
	defer resetTerminalTitle()
	// Tidak perlu defer closeControlMaster(h) di sini, hapus

	var elapsed time.Duration
	var err error

	// Penentuan auth method
	usePassword := h.Password != ""
	useKey := h.IdentityFile != ""

	if usePassword {
		// Password auth — langsung SSH, skip testConnection/ControlMaster
		var remoteCmd string
		switch {
		case path != "" && command != "":
			remoteCmd = fmt.Sprintf("cd '%s' && %s; exec bash -l", path, command)
		case path != "":
			remoteCmd = fmt.Sprintf("cd '%s' && exec bash -l", path)
		default:
			remoteCmd = "exec bash -l"
		}

		if path != "" && command != "" {
			fmt.Printf("→ Menjalankan: %s\n", command)
		}

		args := baseSSHArgs(h)
		args = append(args, "-t", hostAddr(h), remoteCmd)

		cmd := exec.Command("ssh", args...)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		if err := cmd.Run(); err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				if status, ok := exitErr.Sys().(syscall.WaitStatus); ok {
					os.Exit(status.ExitStatus())
				}
			}
			return err
		}
		return nil
	}

	// Identity file (atau default SSH agent)
	if useKey {
		// Coba dengan key terlebih dahulu
		elapsed, err = withSpinner(fmt.Sprintf("Mencoba koneksi menggunakan SSH key...", h.Alias, h.Host), func() error {
			return testConnection(h)
		})
		if err == nil {
			// Key auth berhasil
			return doInteractiveSSH(h, path, command)
		}
		fmt.Printf("\n⚠ SSH key gagal — mencoba password (ada field password)...\n")
	}

	// Sisa proses (key gagal + no password, atau no key)
	defer closeControlMaster(h)

	elapsed, err = withSpinner(fmt.Sprintf("Menghubungkan ke %s (%s)...", h.Alias, h.Host), func() error {
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

	cp := controlPath(h)
	sshArgs := baseSSHArgs(h)
	sshArgs = append(sshArgs, "-t", "-o", "ControlPath="+cp, hostAddr(h), remoteCmd)

	cmd := exec.Command("ssh", sshArgs...)
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

// Fungsi helper untuk interactive SSH (key auth punya control master)
func doInteractiveSSH(h Host, path string, command string) error {
	var elapsed time.Duration
	var err error

	elapsed, err = withSpinner(fmt.Sprintf("Menyambungkan ke %s (%s)...", h.Alias, h.Host), func() error {
		return testConnection(h)
	})
	if err != nil {
		fmt.Printf("✗ Gagal terhubung ke '%s': %v\n", h.Alias, err)
		closeControlMaster(h)
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

	cp := controlPath(h)
	sshArgs := baseSSHArgs(h)
	sshArgs = append(sshArgs, "-t", "-o", "ControlPath="+cp, hostAddr(h), remoteCmd)

	cmd := exec.Command("ssh", sshArgs...)
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

func ExecRemote(h Host, path string, command string) int {
	var remoteCmd string
	if path != "" {
		remoteCmd = fmt.Sprintf("cd '%s' && %s", path, command)
	} else {
		remoteCmd = command
	}

	if h.Password != "" {
		// Password auth — langsung SSH tanpa ControlMaster
		args := baseSSHArgs(h)
		args = append(args, "-T", hostAddr(h), remoteCmd)
		cmd := exec.Command("ssh", args...)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				return exitErr.ExitCode()
			}
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			return 1
		}
		return 0
	}

	elapsed, err := withSpinner(fmt.Sprintf("Menghubungkan ke %s (%s)...", h.Alias, h.Host), func() error {
		return testConnection(h)
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "✗ Gagal terhubung ke '%s': %v\n", h.Alias, err)
		return 1
	}
	fmt.Printf("✓ Terhubung ke %s dalam %dms\n", h.Alias, elapsed.Milliseconds())

	cp := controlPath(h)
	args := baseSSHArgs(h)
	args = append(args, "-T", "-o", "ControlPath="+cp, hostAddr(h), remoteCmd)

	cmd := exec.Command("ssh", args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return exitErr.ExitCode()
		}
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}
	return 0
}

func closeControlMaster(h Host) {
	cp := controlPath(h)
	args := baseSSHArgs(h)
	args = append(args, "-T", "-o", "ControlPath="+cp, hostAddr(h), "-O", "exit")
	exec.Command("ssh", args...).Run()
}

func resetTerminalTitle() {
	fmt.Print("\033]0;\007")
}

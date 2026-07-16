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

func resolvePassword(h Host) (string, bool) {
	if pw, ok := getSecret(h.Alias); ok {
		return pw, true
	}
	if h.Password != "" {
		return h.Password, true
	}
	return "", false
}

// buildSSHCmd SATU-SATUNYA tempat yang memutuskan cara ssh terhubung:
// key/agent biasa, atau lewat sshpass kalau ada password tersimpan.
// Dipakai oleh SEMUA fungsi — supaya tidak ada lagi kasus "1 tempat lupa disamakan".
func buildSSHCmd(h Host, args []string) *exec.Cmd {
	pw, hasPassword := resolvePassword(h)
	if hasPassword {
		if _, err := exec.LookPath("sshpass"); err == nil {
			fullArgs := append([]string{"-e", "ssh"}, args...)
			cmd := exec.Command("sshpass", fullArgs...)
			cmd.Env = append(os.Environ(), "SSHPASS="+pw)
			return cmd
		}
	}
	return exec.Command("ssh", args...)
}

func testConnection(h Host) error {
	logEvent(h.Alias, "CONNECT mulai menghubungkan ke %s:%d", h.Host, h.Port)

	cp := controlPath(h)
	args := baseSSHArgs(h)
	args = append(args,
		"-T",
		"-o", "ConnectTimeout=10",
		"-o", "ControlMaster=auto",
		"-o", "ControlPath="+cp,
		"-o", "ControlPersist=30",
		hostAddr(h), "echo ok")

	cmd := buildSSHCmd(h, args)
	out, err := cmd.CombinedOutput()
	if err != nil {
		logEvent(h.Alias, "CONNECT GAGAL — %s", parseSSHError(string(out), err).Error())
		if raw := strings.TrimSpace(string(out)); raw != "" {
			logEvent(h.Alias, "CONNECT respons mentah server: %s", raw)
		}
		return parseSSHError(string(out), err)
	}
	logEvent(h.Alias, "CONNECT berhasil")
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
	logEvent(h.Alias, "CEK_PATH memeriksa '%s'", path)

	cp := controlPath(h)
	args := baseSSHArgs(h)
	args = append(args, "-T", "-o", "ControlPath="+cp, hostAddr(h),
		fmt.Sprintf("test -d '%s' && echo FOUND || echo MISSING", path))

	cmd := buildSSHCmd(h, args)
	out, err := cmd.CombinedOutput()
	if err != nil {
		logEvent(h.Alias, "CEK_PATH GAGAL — %s", parseSSHError(string(out), err).Error())
		return false, parseSSHError(string(out), err)
	}
	found := strings.Contains(string(out), "FOUND")
	if found {
		logEvent(h.Alias, "CEK_PATH '%s' ditemukan", path)
	} else {
		logEvent(h.Alias, "CEK_PATH '%s' TIDAK ditemukan", path)
	}
	return found, nil
}

func SSHConnect(h Host, path string, command string) error {
	defer resetTerminalTitle()
	defer closeControlMaster(h)

	logEvent(h.Alias, "SESI sesi interaktif dimulai")
	defer logEvent(h.Alias, "SESI sesi interaktif selesai")

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

	cp := controlPath(h)
	args := baseSSHArgs(h)
	args = append(args, "-t", "-o", "ControlPath="+cp, hostAddr(h), remoteCmd)

	cmd := buildSSHCmd(h, args)
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
	elapsed, err := withSpinner(fmt.Sprintf("Menghubungkan ke %s (%s)...", h.Alias, h.Host), func() error {
		return testConnection(h)
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "✗ Gagal terhubung ke '%s': %v\n", h.Alias, err)
		closeControlMaster(h)
		return 1
	}
	fmt.Printf("✓ Terhubung ke %s dalam %dms\n", h.Alias, elapsed.Milliseconds())
	defer closeControlMaster(h)

	var remoteCmd string
	if path != "" {
		remoteCmd = fmt.Sprintf("cd '%s' && %s", path, command)
	} else {
		remoteCmd = command
	}

	cp := controlPath(h)
	args := baseSSHArgs(h)
	args = append(args, "-T", "-o", "ControlPath="+cp, hostAddr(h), remoteCmd)

	logEvent(h.Alias, "EXEC menjalankan: %s", remoteCmd)

	cmd := buildSSHCmd(h, args)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			logEvent(h.Alias, "EXEC selesai, exit code %d", exitErr.ExitCode())
			return exitErr.ExitCode()
		}
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		logEvent(h.Alias, "EXEC selesai, exit code 1")
		return 1
	}
	logEvent(h.Alias, "EXEC selesai, exit code 0")
	return 0
}

func closeControlMaster(h Host) {
	cp := controlPath(h)
	args := baseSSHArgs(h)
	args = append(args, "-T", "-o", "ControlPath="+cp, hostAddr(h), "-O", "exit")
	buildSSHCmd(h, args).Run()
}

func resetTerminalTitle() {
	fmt.Print("\033]0;\007")
}

package main

import (
	"bytes"
	"os/exec"
	"strings"
)

// secretToolAvailable memverifikasi apakah `secret-tool` dapat dijalankan dari PATH saat ini.
func secretToolAvailable() bool {
	_, err := exec.LookPath("secret-tool")
	return err == nil
}

// secretAttrs mengembalikan daftar atribut yang uniquely mengidentifikasi sebuah secret host di keyring.
// Format: service=hop, host=key, <key-ainnya>
func secretAttrs(alias string) []string {
	return []string{"service", "hop", "host", alias}
}

// storeSecret menyimpan password di OS keyring menggunakan `secret-tool store`.
func storeSecret(alias string, password string) error {
	args := append([]string{"store", "--label=hop: " + alias}, secretAttrs(alias)...)
	cmd := exec.Command("secret-tool", args...)
	cmd.Stdin = strings.NewReader(password)
	return cmd.Run()
}

// getSecret mengambil password yang tersimpan di OS keyring.
func getSecret(alias string) (string, bool) {
	args := append([]string{"lookup"}, secretAttrs(alias)...)
	cmd := exec.Command("secret-tool", args...)
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return "", false
	}
	pass := strings.TrimSpace(out.String())
	if pass == "" {
		return "", false
	}
	return pass, true
}

// deleteSecret menghapus password dari OS keyring.
func deleteSecret(alias string) error {
	args := append([]string{"clear"}, secretAttrs(alias)...)
	return exec.Command("secret-tool", args...).Run()
}

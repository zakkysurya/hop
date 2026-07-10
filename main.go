package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"text/tabwriter"

	"golang.org/x/term"
)

func main() {
	if err := migrateOldConfig(); err != nil {
		fmt.Fprintf(os.Stderr, "Error migrasi konfigurasi: %v\n", err)
	}
	if err := migrateSchemaV1ToV2(); err != nil {
		fmt.Fprintf(os.Stderr, "Error migrasi skema konfigurasi: %v\n", err)
	}
	if err := migratePasswordsToKeyring(); err != nil {
		fmt.Fprintf(os.Stderr, "Error migrasi password ke keyring: %v\n", err)
	}

	var overrideCmd string
	args := os.Args[1:]
	for i, a := range os.Args {
		if a == "--" {
			overrideCmd = strings.Join(os.Args[i+1:], " ")
			args = os.Args[1:i]
			break
		}
	}

	if len(args) < 1 {
		printUsage(true)
		return
	}

	if args[0] == "--complete-hosts" {
		cfg, err := LoadConfig()
		if err != nil {
			return
		}
		for _, h := range cfg.Hosts {
			fmt.Println(h.Alias)
		}
		return
	}

	if args[0] == "--complete-paths" && len(args) >= 2 {
		cfg, err := LoadConfig()
		if err != nil {
			return
		}
		for _, h := range cfg.Hosts {
			if h.Alias == args[1] {
				for _, p := range h.Paths {
					fmt.Println(p.Alias)
				}
				break
			}
		}
		return
	}

	if args[0] != "init" && !configExists() {
		fmt.Println("Konfigurasi tidak ditemukan, membuat konfigurasi default...")
		if err := InitConfig(); err != nil {
			fmt.Fprintf(os.Stderr, "Error membuat konfigurasi: %v\n", err)
			return
		}
		fmt.Println("Konfigurasi default dibuat di", configPath)
	}

	if len(args) >= 1 && args[0] == "exec" {
		cmdExec(args, overrideCmd)
		return
	}

	cmd := args[0]
	switch cmd {
	case "list":
		cmdList()
	case "add":
		cmdAdd()
	case "edit":
		cmdEdit(args)
	case "remove":
		cmdRemove(args)
	case "doctor":
		cmdDoctor()
	case "path-add":
		cmdPathAdd(args)
	case "path-edit":
		cmdPathEdit(args)
	case "path-list":
		cmdPathList(args)
	case "path-remove":
		cmdPathRemove(args)
	case "init":
		cmdInit()
	case "completion":
		cmdCompletion(args)
	case "help", "--help", "-h":
		printUsage(false)
	case "secret-remove":
		cmdSecretRemove(args)
	default:
		pathAlias := ""
		if len(args) >= 2 {
			pathAlias = args[1]
		}
		cmdConnect(cmd, pathAlias, overrideCmd)
	}
}

func printUsage(withBanner bool) {
	if withBanner {
		printBanner()
	}
	fmt.Println(`Penggunaan: hop <host-alias> [path-alias] [-- <command>]

Manajemen:
  list            Tampilkan semua Host dan path-nya
  add             Tambah Host baru secara interaktif
  edit    <host>  Edit Host
  remove  <host>  Hapus Host

Path:
  path-list   [<host>]        Tampilkan path (semua Host, atau Host spesifik)
  path-add    <host>          Tambah path ke Host
  path-edit   <host> <path>   Edit alias path
  path-remove <host> <path>   Hapus path dari Host

Lainnya:
  doctor                     Cek konektivitas untuk semua Host yang terkonfigurasi
  exec    <host> [-- <cmd>]  Eksekusi Command secara non-interaktif
  secret-remove <host>       Hapus password Host dari OS keyring
  help                       Tampilkan pesan bantuan ini`)
}

var scanner = bufio.NewScanner(os.Stdin)

func readLine() string {
	scanner.Scan()
	return strings.TrimSpace(scanner.Text())
}

func prompt(label, defaultVal string) string {
	fmt.Printf("%s [%s]: ", label, defaultVal)
	val := readLine()
	if val == "" {
		return defaultVal
	}
	return val
}

func promptRequired(label string) string {
	for {
		fmt.Printf("%s: ", label)
		val := readLine()
		if val != "" {
			return val
		}
		fmt.Println("⚠ Kolom ini wajib diisi.")
	}
}

func findHost(cfg *Config, alias string) int {
	for i, h := range cfg.Hosts {
		if h.Alias == alias {
			return i
		}
	}
	return -1
}

func findPathAlias(host *Host, alias string) int {
	for i, p := range host.Paths {
		if p.Alias == alias {
			return i
		}
	}
	return -1
}

func pathAliasesString(paths []PathAlias) string {
	aliases := make([]string, len(paths))
	for i, p := range paths {
		aliases[i] = p.Alias
	}
	return strings.Join(aliases, ", ")
}

func cmdList() {
	cfg, err := LoadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error membaca konfigurasi: %v\n", err)
		return
	}
	if len(cfg.Hosts) == 0 {
		fmt.Println("Tidak ada Host yang terkonfigurasi.")
		return
	}
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "HOST\tHOST (alias)\tUSER\tPORT\tPATHS (alias)")
	fmt.Fprintln(w, "------------\t------------\t----\t----\t-----")
	for _, h := range cfg.Hosts {
		fmt.Fprintf(w, "%s\t%s\t%s\t%d\t%s\n", h.Host, h.Alias, h.User, h.Port, pathAliasesString(h.Paths))
	}
	w.Flush()
}

func cmdAdd() {
	cfg, err := LoadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return
	}

	h := Host{}
	h.Alias = promptRequired("🏷 Alias host")

	if idx := findHost(cfg, h.Alias); idx >= 0 {
		fmt.Printf("⚠ Host dengan alias '%s' sudah ada. Gunakan 'hop edit %s' untuk mengubahnya.\n", h.Alias, h.Alias)
		return
	}

	h.Host = promptRequired("🌐 Host")
	h.User = promptRequired("👤 User")
	portStr := prompt("🔌 Port", "22")
	h.Port, _ = strconv.Atoi(portStr)
	if h.Port == 0 {
		h.Port = 22
	}

	usePassword := prompt("🔐 Gunakan autentikasi password? (y/N)", "N")
	if strings.ToLower(usePassword) == "y" {
		ensurePasswordToolsInstalled()
		pw := promptRequired("   🔒 Password")
		if secretToolAvailable() {
			if err := storeSecret(h.Alias, pw); err != nil {
				fmt.Printf("⚠ Gagal simpan ke keyring (%v), password disimpan di config sebagai fallback.\n", err)
				h.Password = pw
			} else {
				fmt.Println("   🔒 Password disimpan di OS keyring (sistem penyimpanan kredensial aman bawaan OS untuk melindungi password Anda).")
			}
		} else {
			fmt.Println("⚠ secret-tool tidak ditemukan. Install: sudo apt install libsecret-tools")
			fmt.Println("  Sementara, password disimpan di config.yaml (kurang aman).")
			h.Password = pw
		}
	}

	h.IdentityFile = prompt("🔑 Path File SSH Key (opsional)", "")

	addPaths := prompt("📁 Tambah path untuk Host ini?", "N")
	if strings.ToLower(addPaths) == "y" {
		fmt.Println()
		for {
			pa := PathAlias{}
			pa.Alias = promptRequired("  📁 Alias path")
			pa.Path = promptRequired("  📁 Path")
			pa.Command = prompt("  ▶ Command (opsional, kosongkan jika tidak ada)", "")

			if findPathAlias(&h, pa.Alias) >= 0 {
				fmt.Printf("  ⚠ Alias path '%s' sudah ada di Host ini.\n", pa.Alias)
				continue
			}

			h.Paths = append(h.Paths, pa)

			more := prompt("  📁 Tambah path lain?", "N")
			if strings.ToLower(more) != "y" {
				break
			}
		}
	}

	cfg.Hosts = append(cfg.Hosts, h)
	if err := SaveConfig(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "Error menyimpan konfigurasi: %v\n", err)
		return
	}
	fmt.Printf("✓ Host '%s' berhasil ditambahkan.\n", h.Alias)
}

func cmdEdit(args []string) {
	if len(args) < 2 {
		fmt.Println("Penggunaan: hop edit <host-alias>")
		return
	}
	alias := args[1]
	cfg, err := LoadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return
	}
	idx := findHost(cfg, alias)
	if idx < 0 {
		fmt.Printf("⚠ Host '%s' tidak ditemukan.\n", alias)
		return
	}
	h := &cfg.Hosts[idx]
	fmt.Println("Kosongkan untuk mempertahankan nilai saat ini.")
	h.Alias = prompt("🏷 Alias host", h.Alias)
	h.Host = prompt("🌐 Host", h.Host)
	h.User = prompt("👤 User", h.User)
	h.Port, _ = strconv.Atoi(prompt("🔌 Port", strconv.Itoa(h.Port)))

	if h.Alias != alias {
		if pw, ok := getSecret(alias); ok {
			if err := storeSecret(h.Alias, pw); err == nil {
				deleteSecret(alias)
				fmt.Printf("🔒 Password dipindahkan dari '%s' ke '%s' di keyring.\n", alias, h.Alias)
			}
		}
	}

	updatePass := prompt("🔒 Ubah password? (y/N)", "N")
	if strings.ToLower(updatePass) == "y" {
		ensurePasswordToolsInstalled()
		pw := promptRequired("   🔒 Password")
		if secretToolAvailable() {
			if err := storeSecret(h.Alias, pw); err != nil {
				fmt.Printf("⚠ Gagal simpan ke keyring (%v), password disimpan di config sebagai fallback.\n", err)
				h.Password = pw
			} else {
				fmt.Println("   🔒 Password disimpan di OS keyring (sistem penyimpanan kredensial aman bawaan OS untuk melindungi password Anda).")
				h.Password = ""
			}
		} else {
			fmt.Println("⚠ secret-tool tidak ditemukan. Install: sudo apt install libsecret-tools")
			fmt.Println("  Sementara, password disimpan di config.yaml (kurang aman).")
			h.Password = pw
		}
	}
	h.IdentityFile = prompt("🔑 Path File SSH Key (opsional)", h.IdentityFile)
	if err := SaveConfig(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "Error menyimpan konfigurasi: %v\n", err)
		return
	}
	fmt.Printf("✓ Host '%s' berhasil diperbarui.\n", h.Alias)
}

func cmdRemove(args []string) {
	if len(args) < 2 {
		fmt.Println("Penggunaan: hop remove <host-alias>")
		return
	}
	alias := args[1]
	cfg, err := LoadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return
	}
	idx := findHost(cfg, alias)
	if idx < 0 {
		fmt.Printf("⚠ Host '%s' tidak ditemukan.\n", alias)
		return
	}
	
	fmt.Printf("🗑️ Hapus Host '%s'? (y/N): ", alias)
	answer := readLine()
	if strings.ToLower(answer) != "y" {
		fmt.Println("Dibatalkan.")
		return
	}
	deleteSecret(alias)
	cfg.Hosts = append(cfg.Hosts[:idx], cfg.Hosts[idx+1:]...)
	if err := SaveConfig(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "Error menyimpan konfigurasi: %v\n", err)
		return
	}
	
	fmt.Printf("✓ Host '%s' berhasil dihapus.\n", alias)
}

func cmdConnect(hostAlias string, pathAlias string, overrideCmd string) {
	cfg, err := LoadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return
	}
	idx := findHost(cfg, hostAlias)
	if idx < 0 {
		fmt.Printf("⚠ Host '%s' tidak ditemukan. Gunakan 'hop list' untuk melihat Host yang tersedia.\n", hostAlias)
		return
	}
	host := cfg.Hosts[idx]

	if len(host.Paths) == 0 {
		if pathAlias != "" {
			fmt.Printf("⚠ Alias path '%s' tidak ditemukan untuk Host '%s'.\n", pathAlias, hostAlias)
			return
		}
		if err := SSHConnect(host, "", overrideCmd); err != nil {
			fmt.Fprintf(os.Stderr, "SSH error: %v\n", err)
		}
		return
	}

	if pathAlias == "" {
		pathAlias = host.Paths[0].Alias
	}

	pathIdx := findPathAlias(&host, pathAlias)
	if pathIdx < 0 {
		fmt.Printf("⚠ Alias path '%s' tidak ditemukan untuk Host '%s'.\n", pathAlias, hostAlias)
		fmt.Println("Path yang tersedia:")
		for _, p := range host.Paths {
			fmt.Printf("  - %s\n", p.Alias)
		}
		return
	}

	targetPath := host.Paths[pathIdx].Path
	defaultCmd := host.Paths[pathIdx].Command
	finalCmd := defaultCmd
	if overrideCmd != "" {
		finalCmd = overrideCmd
	}

	if err := SSHConnect(host, targetPath, finalCmd); err != nil {
		fmt.Fprintf(os.Stderr, "SSH error: %v\n", err)
	}
}

func cmdPathAdd(args []string) {
	if len(args) < 2 {
		fmt.Println("Penggunaan: hop path-add <host-alias>")
		return
	}
	alias := args[1]
	cfg, err := LoadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return
	}
	idx := findHost(cfg, alias)
	if idx < 0 {
		fmt.Printf("⚠ Host '%s' tidak ditemukan.\n", alias)
		return
	}
	h := &cfg.Hosts[idx]

	pa := PathAlias{}
	pa.Alias = promptRequired("📁 Alias path")
	pa.Path = promptRequired("📁 Path")
	pa.Command = prompt("▶ Command (opsional, kosongkan jika tidak ada)", "")

	if findPathAlias(h, pa.Alias) >= 0 {
		fmt.Printf("⚠ Alias path '%s' sudah ada di Host '%s'.\n", pa.Alias, alias)
		return
	}

	h.Paths = append(h.Paths, pa)
	if err := SaveConfig(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "Error menyimpan konfigurasi: %v\n", err)
		return
	}
	fmt.Printf("✓ Path '%s' berhasil ditambahkan ke Host '%s'.\n", pa.Alias, alias)
}

func cmdPathList(args []string) {
	cfg, err := LoadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return
	}

	if len(args) >= 2 {
		alias := args[1]
		hIdx := findHost(cfg, alias)
		if hIdx < 0 {
			fmt.Printf("⚠ Host '%s' tidak ditemukan.\n", alias)
			return
		}
		h := cfg.Hosts[hIdx]
		if len(h.Paths) == 0 {
			fmt.Printf("Tidak ada path yang terkonfigurasi untuk Host '%s'.\n", alias)
			return
		}
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
		fmt.Fprintln(w, "PATH ALIAS\tPATH\tCOMMAND")
		fmt.Fprintln(w, "----------\t----\t-------")
		for _, p := range h.Paths {
			fmt.Fprintf(w, "%s\t%s\t%s\n", p.Alias, p.Path, p.Command)
		}
		w.Flush()
		return
	}

	hasPaths := false
	for _, h := range cfg.Hosts {
		if len(h.Paths) > 0 {
			hasPaths = true
			break
		}
	}
	if !hasPaths {
		fmt.Println("Tidak ada path yang terkonfigurasi.")
		return
	}
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "HOST\tHOST (alias)\tPATH\tPATH (alias)\tCOMMAND")
	fmt.Fprintln(w, "------------\t------------\t----\t------------\t-------")
	for _, h := range cfg.Hosts {
		for _, p := range h.Paths {
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", h.Host, h.Alias, p.Path, p.Alias, p.Command)
		}
	}
	w.Flush()
}

func cmdPathEdit(args []string) {
	if len(args) < 3 {
		fmt.Println("Penggunaan: hop path-edit <host-alias> <path-alias>")
		return
	}
	hostAlias := args[1]
	pathAlias := args[2]
	cfg, err := LoadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return
	}
	hIdx := findHost(cfg, hostAlias)
	if hIdx < 0 {
		fmt.Printf("⚠ Host '%s' tidak ditemukan.\n", hostAlias)
		return
	}
	h := &cfg.Hosts[hIdx]

	pIdx := findPathAlias(h, pathAlias)
	if pIdx < 0 {
		fmt.Printf("⚠ Alias path '%s' tidak ditemukan untuk Host '%s'.\n", pathAlias, hostAlias)
		return
	}

	pa := &h.Paths[pIdx]
	fmt.Println("Kosongkan untuk mempertahankan nilai saat ini.")
	pa.Alias = prompt("📁 Alias path", pa.Alias)
	pa.Path = prompt("📁 Path", pa.Path)
	pa.Command = prompt("▶ Command (opsional)", pa.Command)

	if err := SaveConfig(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "Error menyimpan konfigurasi: %v\n", err)
		return
	}
	fmt.Printf("✓ Path '%s' berhasil diperbarui untuk Host '%s'.\n", pa.Alias, hostAlias)
}

func cmdPathRemove(args []string) {
	if len(args) < 3 {
		fmt.Println("Penggunaan: hop path-remove <host-alias> <path-alias>")
		return
	}
	hostAlias := args[1]
	pathAlias := args[2]
	cfg, err := LoadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return
	}
	hIdx := findHost(cfg, hostAlias)
	if hIdx < 0 {
		fmt.Printf("⚠ Host '%s' tidak ditemukan.\n", hostAlias)
		return
	}
	h := &cfg.Hosts[hIdx]

	pIdx := findPathAlias(h, pathAlias)
	if pIdx < 0 {
		fmt.Printf("⚠ Alias path '%s' tidak ditemukan untuk Host '%s'.\n", pathAlias, hostAlias)
		return
	}
	
	fmt.Printf("🗑️ Hapus path '%s' dari Host '%s'? (y/N): ", pathAlias, hostAlias)
	answer := readLine()
	if strings.ToLower(answer) != "y" {
		fmt.Println("Dibatalkan.")
		return
	}

	h.Paths = append(h.Paths[:pIdx], h.Paths[pIdx+1:]...)
	if err := SaveConfig(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "Error menyimpan konfigurasi: %v\n", err)
		return
	}
	
	fmt.Printf("✓ Path '%s' berhasil dihapus dari Host '%s'.\n", pathAlias, hostAlias)
}

func cmdCompletion(args []string) {
	if len(args) >= 2 && args[1] == "bash" {
		fmt.Print(`_hop_completions() {
    local cur prev
    cur="${COMP_WORDS[COMP_CWORD]}"
    prev="${COMP_WORDS[COMP_CWORD-1]}"

    if [ "$COMP_CWORD" -eq 1 ]; then
        COMPREPLY=($(compgen -W "$(hop --complete-hosts) list add edit remove path-list path-add path-edit path-remove init help secret-remove" -- "$cur"))
    elif [ "$COMP_CWORD" -eq 2 ]; then
        case "$prev" in
            edit|remove|path-list|path-add|path-edit|path-remove)
                COMPREPLY=($(compgen -W "$(hop --complete-hosts)" -- "$cur"))
                ;;
            *)
                COMPREPLY=($(compgen -W "$(hop --complete-paths "$prev")" -- "$cur"))
                ;;
        esac
    elif [ "$COMP_CWORD" -eq 3 ]; then
        case "${COMP_WORDS[1]}" in
            path-remove|path-edit)
                COMPREPLY=($(compgen -W "$(hop --complete-paths "${COMP_WORDS[2]}")" -- "$cur"))
                ;;
        esac
    fi
}
complete -F _hop_completions hop
`)
		return
	}
	fmt.Println("Penggunaan: hop completion bash")
}

func cmdInit() {
	if configExists() {
		fmt.Println("Konfigurasi sudah ada di", configPath)
		return
	}
	if err := InitConfig(); err != nil {
		fmt.Fprintf(os.Stderr, "Error membuat konfigurasi: %v\n", err)
		return
	}
	fmt.Println("Konfigurasi default dibuat di", configPath)
}

func cmdSecretRemove(args []string) {
	if len(args) < 2 {
		fmt.Println("Penggunaan: hop secret-remove <host-alias>")
		return
	}
	alias := args[1]
	if err := deleteSecret(alias); err != nil {
		fmt.Printf("⚠ Tidak ada secret tersimpan untuk '%s', atau secret-tool tidak tersedia.\n", alias)
		return
	}
	fmt.Printf("✓ Password untuk '%s' dihapus dari keyring.\n", alias)
}

func checkTool(name string, required bool, note string) {
	_, err := exec.LookPath(name)
	if err == nil {
		fmt.Printf("✓ %-12s terpasang\n", name)
		return
	}
	if required {
		fmt.Printf("✗ %-12s TIDAK ditemukan — %s\n", name, note)
	} else {
		fmt.Printf("⚠ %-12s tidak ditemukan — %s\n", name, note)
	}
}

func getSudoPasswordIfNeeded() string {
	err := exec.Command("sudo", "-n", "true").Run()
	if err == nil {
		return ""
	}

	fmt.Print("   🔐 Membutuhkan akses sudo untuk instalasi. Masukkan password OS Anda: ")

	bytePassword, err := term.ReadPassword(int(syscall.Stdin))
	fmt.Println()

	if err != nil {
		return ""
	}
	return string(bytePassword)
}

func ensurePasswordToolsInstalled() {
	needsSecretTool := !secretToolAvailable()

	_, err := exec.LookPath("sshpass")
	needsSshpass := (err != nil)

	if !needsSecretTool && !needsSshpass {
		return
	}

	sudoPass := getSudoPasswordIfNeeded()

	if needsSecretTool {
		_, err := withSpinner("Memasang libsecret-tools...", func() error {
			cmd := exec.Command("sudo", "-S", "env", "DEBIAN_FRONTEND=noninteractive", "apt-get", "install", "-y", "-qq", "libsecret-tools")
			if sudoPass != "" {
				cmd.Stdin = strings.NewReader(sudoPass + "\n")
			}
			return cmd.Run()
		})

		if err != nil {
			fmt.Println("    ✗ Gagal memasang libsecret-tools (pastikan password OS benar)")
		} else {
			fmt.Println("    ✓ libsecret-tools berhasil dipasang")
		}
	}

	if needsSshpass {
		_, err := withSpinner("Memasang sshpass...", func() error {
			cmd := exec.Command("sudo", "-S", "env", "DEBIAN_FRONTEND=noninteractive", "apt-get", "install", "-y", "-qq", "sshpass")
			if sudoPass != "" {
				cmd.Stdin = strings.NewReader(sudoPass + "\n")
			}
			return cmd.Run()
		})

		if err != nil {
			fmt.Println("    ✗ Gagal memasang sshpass (pastikan password OS benar)")
		} else {
			fmt.Println("    ✓ sshpass berhasil dipasang")
		}
	}
}

func cmdDoctor() {
	fmt.Println("🔧 Memeriksa tools yang dibutuhkan...")
	fmt.Println()

	checkTool("ssh", true, "wajib untuk semua fungsi hop — pastikan OpenSSH client terpasang")
	checkTool("secret-tool", false, "opsional, untuk simpan password di OS keyring (sudo apt install libsecret-tools)")
	checkTool("sshpass", false, "opsional, untuk autentikasi password non-interaktif (sudo apt install sshpass)")

	fmt.Println()

	cfg, err := LoadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return
	}
	if len(cfg.Hosts) == 0 {
		fmt.Println("Tidak ada Host yang terkonfigurasi.")
		return
	}

	fmt.Println("🌐 Memeriksa koneksi ke semua Host...")
	fmt.Println()

	okCount, failCount := 0, 0
	for _, h := range cfg.Hosts {
		elapsed, err := withSpinner(fmt.Sprintf("Cek %s (%s)...", h.Alias, h.Host), func() error {
			return testConnection(h)
		})
		if err != nil {
			fmt.Printf("✗ %-20s %s — %v\n", h.Alias, h.Host, err)
			failCount++
		} else {
			fmt.Printf("✓ %-20s %s — %dms\n", h.Alias, h.Host, elapsed.Milliseconds())
			okCount++
		}
		closeControlMaster(h)
	}

	fmt.Println()
	fmt.Printf("Selesai: %d OK, %d gagal dari %d Host.\n", okCount, failCount, len(cfg.Hosts))
}

func cmdExec(args []string, command string) {
	if command == "" {
		fmt.Println("Penggunaan: hop exec <host-alias> [path-alias] -- <command>")
		fmt.Println("Command setelah -- wajib diisi untuk 'hop exec'.")
		return
	}
	if len(args) < 2 {
		fmt.Println("Penggunaan: hop exec <host-alias> [path-alias] -- <command>")
		return
	}

	hostAlias := args[1]
	pathAlias := ""
	if len(args) >= 3 {
		pathAlias = args[2]
	}

	cfg, err := LoadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	idx := findHost(cfg, hostAlias)
	if idx < 0 {
		fmt.Printf("⚠ Host '%s' tidak ditemukan. Gunakan 'hop list' untuk melihat Host yang tersedia.\n", hostAlias)
		os.Exit(1)
	}
	host := cfg.Hosts[idx]

	targetPath := ""
	if len(host.Paths) > 0 {
		if pathAlias == "" {
			targetPath = host.Paths[0].Path
		} else if pIdx := findPathAlias(&host, pathAlias); pIdx >= 0 {
			targetPath = host.Paths[pIdx].Path
		}
	}

	exitCode := ExecRemote(host, targetPath, command)
	os.Exit(exitCode)
}

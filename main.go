package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"text/tabwriter"
)

func main() {
	if err := migrateOldConfig(); err != nil {
		fmt.Fprintf(os.Stderr, "Error migrating config: %v\n", err)
	}
	if err := migrateSchemaV1ToV2(); err != nil {
		fmt.Fprintf(os.Stderr, "Error migrating config schema: %v\n", err)
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
		fmt.Println("No config found, creating default config...")
		if err := InitConfig(); err != nil {
			fmt.Fprintf(os.Stderr, "Error creating config: %v\n", err)
			return
		}
		fmt.Println("Default config created at", configPath)
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
	fmt.Println(`Usage: hop <command> [args]
       hop <host-alias> [path-alias] [-- <command>]

Management:
  list                 Show all hosts and their paths
  add                  Add a new host interactively
  edit    <host>       Edit a host
  remove  <host>       Remove a host

Paths:
  path-list   [<host>]        List paths (all hosts, or specific host)
  path-add    <host>          Add a path to a host
  path-edit   <host> <path>   Edit a path alias
  path-remove <host> <path>   Remove a path from a host

Other:
  doctor               Check connectivity for all configured hosts
  exec    <host> [-- <cmd>]  Execute command non-interactively
  help                 Show this help message`)	
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
		fmt.Println("This field is required.")
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
		fmt.Fprintf(os.Stderr, "Error reading config: %v\n", err)
		return
	}
	if len(cfg.Hosts) == 0 {
		fmt.Println("No hosts configured.")
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
	h.Alias = promptRequired("Alias host")
	h.Host = promptRequired("Host")
	h.User = promptRequired("User")
	portStr := prompt("Port", "22")
	h.Port, _ = strconv.Atoi(portStr)
	if h.Port == 0 {
		h.Port = 22
	}

	h.IdentityFile = prompt("Path File SSH Key (opsional)", "")

	if idx := findHost(cfg, h.Alias); idx >= 0 {
		fmt.Printf("Host '%s' already exists. Use 'hop edit %s' to modify it.\n", h.Alias, h.Alias)
		return
	}

	fmt.Println()
	fmt.Println("Tambah path untuk host ini:")
	for {
		pa := PathAlias{}
		pa.Alias = promptRequired("  Alias path")
		pa.Path = promptRequired("  Path")
		pa.Command = prompt("  Command (opsional, kosongkan jika tidak ada)", "")

		if findPathAlias(&h, pa.Alias) >= 0 {
			fmt.Printf("  Path alias '%s' already exists in this host.\n", pa.Alias)
			continue
		}

		h.Paths = append(h.Paths, pa)

		more := prompt("  Tambah path lain?", "N")
		if strings.ToLower(more) != "y" {
			break
		}
	}

	if len(h.Paths) == 0 {
		fmt.Println("At least one path is required.")
		return
	}

	cfg.Hosts = append(cfg.Hosts, h)
	if err := SaveConfig(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "Error saving config: %v\n", err)
		return
	}
	fmt.Printf("Host '%s' added.\n", h.Alias)
}

func cmdEdit(args []string) {
	if len(args) < 2 {
		fmt.Println("Usage: hop edit <host-alias>")
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
		fmt.Printf("Host '%s' not found.\n", alias)
		return
	}
	h := &cfg.Hosts[idx]
	fmt.Println("Leave blank to keep current value.")
	h.Alias = prompt("Alias host", h.Alias)
	h.Host = prompt("Host", h.Host)
	h.User = prompt("User", h.User)
	h.Port, _ = strconv.Atoi(prompt("Port", strconv.Itoa(h.Port)))
	h.IdentityFile = prompt("Path File SSH Key (opsional)", h.IdentityFile)
	if err := SaveConfig(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "Error saving config: %v\n", err)
		return
	}
	fmt.Printf("Host '%s' updated.\n", h.Alias)
}

func cmdRemove(args []string) {
	if len(args) < 2 {
		fmt.Println("Usage: hop remove <host-alias>")
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
		fmt.Printf("Host '%s' not found.\n", alias)
		return
	}
	fmt.Printf("Remove host '%s'? (y/N): ", alias)
	answer := readLine()
	if strings.ToLower(answer) != "y" {
		fmt.Println("Cancelled.")
		return
	}
	cfg.Hosts = append(cfg.Hosts[:idx], cfg.Hosts[idx+1:]...)
	if err := SaveConfig(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "Error saving config: %v\n", err)
		return
	}
	fmt.Printf("Host '%s' removed.\n", alias)
}

func cmdConnect(hostAlias string, pathAlias string, overrideCmd string) {
	cfg, err := LoadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return
	}
	idx := findHost(cfg, hostAlias)
	if idx < 0 {
		fmt.Printf("Host '%s' not found. Use 'hop list' to see available hosts.\n", hostAlias)
		return
	}
	host := cfg.Hosts[idx]

	if len(host.Paths) == 0 {
		if pathAlias == "" {
			fmt.Printf("Host '%s' belum memiliki path. Silakan tambahkan path terlebih dahulu.\n", hostAlias)
		} else {
			fmt.Printf("Path alias '%s' tidak ditemukan untuk host '%s'. Silakan tambahkan path terlebih dahulu.\n", pathAlias, hostAlias)
		}
		return
	}

	if pathAlias == "" {
		pathAlias = host.Paths[0].Alias
	}

	pathIdx := findPathAlias(&host, pathAlias)
	if pathIdx < 0 {
		fmt.Printf("Path alias '%s' tidak ditemukan untuk host '%s'.\n", pathAlias, hostAlias)
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
		fmt.Println("Usage: hop path-add <host-alias>")
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
		fmt.Printf("Host '%s' not found.\n", alias)
		return
	}
	h := &cfg.Hosts[idx]

	pa := PathAlias{}
	pa.Alias = promptRequired("Alias path")
	pa.Path = promptRequired("Path")
	pa.Command = prompt("Command (opsional, kosongkan jika tidak ada)", "")

	if findPathAlias(h, pa.Alias) >= 0 {
		fmt.Printf("Path alias '%s' already exists in host '%s'.\n", pa.Alias, alias)
		return
	}

	h.Paths = append(h.Paths, pa)
	if err := SaveConfig(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "Error saving config: %v\n", err)
		return
	}
	fmt.Printf("Path '%s' added to host '%s'.\n", pa.Alias, alias)
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
			fmt.Printf("Host '%s' not found.\n", alias)
			return
		}
		h := cfg.Hosts[hIdx]
		if len(h.Paths) == 0 {
			fmt.Printf("No paths configured for host '%s'.\n", alias)
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
		fmt.Println("No paths configured.")
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
		fmt.Println("Usage: hop path-edit <host-alias> <path-alias>")
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
		fmt.Printf("Host '%s' not found.\n", hostAlias)
		return
	}
	h := &cfg.Hosts[hIdx]

	pIdx := findPathAlias(h, pathAlias)
	if pIdx < 0 {
		fmt.Printf("Path alias '%s' not found for host '%s'.\n", pathAlias, hostAlias)
		return
	}

	pa := &h.Paths[pIdx]
	fmt.Println("Leave blank to keep current value.")
	pa.Alias = prompt("Alias path", pa.Alias)
	pa.Path = prompt("Path", pa.Path)
	pa.Command = prompt("Command (opsional)", pa.Command)

	if err := SaveConfig(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "Error saving config: %v\n", err)
		return
	}
	fmt.Printf("Path '%s' updated for host '%s'.\n", pa.Alias, hostAlias)
}

func cmdPathRemove(args []string) {
	if len(args) < 3 {
		fmt.Println("Usage: hop path-remove <host-alias> <path-alias>")
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
		fmt.Printf("Host '%s' not found.\n", hostAlias)
		return
	}
	h := &cfg.Hosts[hIdx]

	pIdx := findPathAlias(h, pathAlias)
	if pIdx < 0 {
		fmt.Printf("Path alias '%s' not found for host '%s'.\n", pathAlias, hostAlias)
		return
	}
	fmt.Printf("Remove path '%s' from host '%s'? (y/N): ", pathAlias, hostAlias)
	answer := readLine()
	if strings.ToLower(answer) != "y" {
		fmt.Println("Cancelled.")
		return
	}

	h.Paths = append(h.Paths[:pIdx], h.Paths[pIdx+1:]...)
	if err := SaveConfig(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "Error saving config: %v\n", err)
		return
	}
	fmt.Printf("Path '%s' removed from host '%s'.\n", pathAlias, hostAlias)
}

func cmdCompletion(args []string) {
	if len(args) >= 2 && args[1] == "bash" {
		fmt.Print(`_hop_completions() {
    local cur prev
    cur="${COMP_WORDS[COMP_CWORD]}"
    prev="${COMP_WORDS[COMP_CWORD-1]}"

    if [ "$COMP_CWORD" -eq 1 ]; then
        COMPREPLY=($(compgen -W "$(hop --complete-hosts) list add edit remove path-list path-add path-edit path-remove init help" -- "$cur"))
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
	fmt.Println("Usage: hop completion bash")
}

func cmdInit() {
	if configExists() {
		fmt.Println("Config already exists at", configPath)
		return
	}
	if err := InitConfig(); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating config: %v\n", err)
		return
	}
	fmt.Println("Default config created at", configPath)
}

func cmdDoctor() {
	cfg, err := LoadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return
	}
	if len(cfg.Hosts) == 0 {
		fmt.Println("No hosts configured.")
		return
	}

	fmt.Println("Memeriksa koneksi ke semua host...")
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
	fmt.Printf("Selesai: %d OK, %d gagal dari %d host.\n", okCount, failCount, len(cfg.Hosts))
}

func cmdExec(args []string, command string) {
	if command == "" {
		fmt.Println("Usage: hop exec <host-alias> [path-alias] -- <command>")
		fmt.Println("Command setelah -- wajib diisi untuk 'hop exec'.")
		return
	}
	if len(args) < 2 {
		fmt.Println("Usage: hop exec <host-alias> [path-alias] -- <command>")
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
		fmt.Printf("Host '%s' not found. Use 'hop list' to see available hosts.\n", hostAlias)
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

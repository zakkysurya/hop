<div align="center">

<img src="docs/images/hop-banner.png" alt="hop — jump from laptop to server" width="600" />



# 🐇💨 hop
**SSH Project Launcher & Directory Jumper**

Lompat langsung ke server dan direktori project favorit Anda — tanpa `history | grep` lagi.

[![Go Version](https://img.shields.io/badge/Go-1.20%2B-00ADD8?style=flat-square&logo=go&logoColor=white)](https://go.dev)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg?style=flat-square)](LICENSE)
[![Platform](https://img.shields.io/badge/platform-Linux%20%7C%20macOS%20%7C%20Windows-lightgrey?style=flat-square)]()
[![Build](https://img.shields.io/badge/build-single%20binary-success?style=flat-square)]()

</div>

---

`hop` adalah CLI ringan berbasis Go untuk mempermudah koneksi SSH dan *directory jumping* — otomatis masuk ke direktori project yang tepat begitu Anda terhubung ke server. Tanpa dependensi eksternal yang berat, murni *shell out* ke binary `ssh` bawaan sistem Anda.

> Dulu: `history | grep <ip>` → `!<no>` → `history | grep <project>` → `!<no>`
> Sekarang: `hop <alias>` ✨

## 📚 Daftar Isi

- [Fitur Utama](#-fitur-utama)
- [Instalasi](#-instalasi)
- [Autentikasi & Auto-Login](#key-autentikasi--auto-login)
- [Konfigurasi](#️-konfigurasi-configyaml)
- [Penggunaan](#-penggunaan)
- [Autocomplete](#️-autocomplete-tab-di-bash)

---

## ✨ Fitur Utama

| Fitur | Deskripsi |
|---|---|
| 🗂️ **Multi-Path per Host** | Satu server bisa punya banyak direktori project, masing-masing dengan alias sendiri |
| 🚀 **Auto Jumper** | Otomatis `cd` ke path tujuan begitu SSH berhasil terhubung |
| 📋 **Tabel Rapi** | Output `hop list` & `hop path-list` mudah dibaca sekilas |
| 🛠️ **Manajemen Interaktif** | `add`, `edit`, `remove`, `path-add`, `path-remove` — semua dipandu prompt |
| 🔄 **Migrasi Otomatis** | Config lama otomatis dicadangkan & dikonversi ke skema terbaru |
| ⌨️ **Autocomplete Pintar** | Tab-completion untuk command, alias host, dan alias path |

---

## 🔑 Autentikasi & Auto-Login

`/hop` mendukung dua metode autentikasi SSH — **SSH key** (recommended) dan **password** (via `sshpass`).

### 🔑 Autentikasi SSH Key (Recommended)

#### Install SSH Key

Generate sebuah key pair (ED25519 adalah pilihan modern):

```bash
# Linux / macOS
ssh-keygen -t ed25519 -C "your_email@domain.com"

# Windows (PowerShell)
ssh-keygen -t ed25519 -C "your_email@domain.com"
```

#### Pasang Public Key ke Server SSH

Copy public key ke `~/.ssh/authorized_keys`:

```bash
# Linux / macOS
ssh-copy-id user@server-ip
ssh-copy-id -i ~/.ssh/id_ed25519.pub user@server-ip

# Windows (PowerShell)
# Ganti USER & IP dulu
ssh-agent ADD_PATH
ssh-add $env:USERPROFILE\.ssh\id_ed25519
ssh-copy-id user@server-ip
```

> **Tips:** `ssh-copy-id` bisa jadi interaktif — pastikan `ssh` berfungsi tanpa password dulu.

#### Configure hop

Edit `~/.config/hop/config.yaml` (or `./hop add`).

```yaml
hosts:
  - alias: prod-server
    host: 192.168.1.100
    user: user
    port: 22
    identity_file: ~/.ssh/id_ed25519
    paths:
      - alias: web
        path: /var/www/html
```

Now `hop prod-server` login otomatis via key, tidak perlu password.

### 🔑 Autentikasi Password (sshpass)

> `sshpass` adalah utilitas open-source yang mengirimkan password via stdin ke `ssh` (hanya untuk testing/domains).
> Meskipun kurang aman dibanding key, berguna untuk server tanpa kunci dan script automatisasi.
> **Install sshpass** di OS anda:

```bash
# Ubuntu/Debian
 sudo apt update && sudo apt install -y sshpass

# macOS
brew install hudochenkov/sshpass/sshpass

# Linux via pacman (Arch etc.)
 sudo pacman -S sshpass

# Verify
sshpass -V
```

#### Configure Hop with Password

Password **TIDAK disimpan plaintext di `config.yaml`**. Saat `hop add`, jawab **y** pada
pertanyaan autentikasi password — password disimpan langsung ke **OS keyring** (`secret-tool`)
dan `config.yaml` tetap bebas credential. `sshpass` hanya dipakai sebagai mekanisme pengiriman
password ke `ssh` (bukan penyimpanan). Lihat [Bagian B](#🟡-bagian-b--password-via-sshpass-auto-login)
untuk detail lengkap.

#### Password Authentication Flow

Setelah `sshpass` di-install, `hop` berjalan otomatis (key atau password, keyring/sistem menentukan):

1. **Jika `identity_file` diisi:** → Coba autentikasi key terlebih dahulu
2. **Jika key gagal + password ada di keyring:** → otomatis fallback ke password (`sshpass`)
3. **Jika hanya password:** → Langsung autentikasi password via `sshpass` (no prompt manual)
4. **Jika keduanya kosong:** → Gunakan SSH agent/default (no required).

#### Windows (Password + sshpass)

> ⚠️ **Catatan:** `sshpass` tidak ada secara native di Windows. Auto-login password di Windows
> hanya bisa lewat **WSL**, **Git Bash**, atau **Cygwin** (tempat `sshpass` terinstall), lalu
> jalankan `hop` di dalam environment tersebut. Untuk pengalaman terbaik, gunakan **SSH Key**
> yang bekerja native di Windows.

#### Verify Password Authentication

```bash
# Coba ping sebelum hop
hop doctor               # Test semua host (password/key)

# Direct login testing
hop prod-server          # Dengan sshpass di-install → Login otomatis

# Manual password prompt (sshpass tidak ada)
hop prod-server          # Akan muncul prompt: password:
```

---

### 🪟 Instruksi Windows

#### Instalasi umum (File exe kompiler)

```powershell
# 1. Buka PowerShell (Windows 10/11 memiliki built-in)
# 2. Verifikasi OpenSSH client aktif (kemungkinan diperlukan - dapatkan dari Windows Features)
# 3. Kompilasi hop.exe
#    (Perlukan go compiler terinstall)
#    go build -o hop.exe .
# 4. Tentukan direktori install (e.g. C:\bin)
#    New-Item -ItemType Directory -Force -Path "C:\bin"
# 5. Salin hop.exe ke C:\bin
#    Copy-Item hop.exe -Destination "C:\bin\hop.exe"
# 6. Tambahkan C:\bin ke PATH (opsi sistem atau user)
#    Windows Settings → System → About → System -> Advanced system settings
#    → Environment Variables → New dalam 'Path'
# 7. Tambahkan variable HOME (harus ada karena hop mencari config via $HOME)
#    Set-Item Env:HOME "C:\Users\$env:USERNAME"
#    (atau ketik ini di PowerShell: [Environment]::SetEnvironmentVariable('HOME', 'C:\Users\$env:USERNAME', 'User'))
# 8. Verifikasi
#    hop.exe --complete-hosts
```

#### Windows Security Note

SSH Server via Windows harus diaktifkan melalui fitur WS-Man
atau gunakan remote desktop (misalnya menggunakan Cloud Shell).

---

### 📖 Contoh Alur Kerja Umum

#### Scenario 1: Server dengan SSH key

```bash
# 1. Peran SSH key di server (sudah dilakukan)
#    ssh user@server-ip "echo 'Hello from server'"

# 2. Hop konfigurasi + key di HOME
~/.config/hop/config.yaml (dengan identity_file)

# 3. Login dari mana saja
hop prod-server
# Output: (mungkin spinner hops) → Login otomatis, cd /var/www/html
```

#### Scenario 2: Server dengan password (sshpass)

```bash
# 1. Install sshpass (Ubuntu/Debian)
 sudo apt install -y sshpass

# 2. Konfigurasi password di config.yaml
hosts:
  - alias: prod-server
    host: server-ip
    user: user
    port: 22
    password: secret123

# 3. Login (auto-login)
hop prod-server
# Output: (login otomatis, no prompt password)
```

#### Scenario 3: Akses Password manual (sshpass tidak ada)

```bash
# 1. Tidak di-install sshpass
hop prod-server
# Output: password prompt
#   Ketikkan password secret123, login
```

---

### 🔍 Troubleshooting

| Masalah | Periksa |
|---|---|
| Auto-login gagal padahal sshpass terinstall | `which sshpass` pada shell yang menjalankan hop |
| Password prompt muncul terus | `sshpass -V` terinstal, pengecekan path config `identity_file` |
| Login kunci gagal | Verifikasi permission key (`chmod 600 ~/.ssh/id_ed25519`) dan `authorized_keys` terperiksa (chmod 644) |
| Path tidak ditemukan | Gunakan `hop list` → jangan lupa config path valid (mungkin harus pakai path relatif) |
| Windows PATH error | Buka “Run as admin” PowerShell, tambahkan environment variable HOME yang benar |

---

**Question lain ?:** Lebih memilih autoprompt password, directory skip, logging, keamanan, instalasi, atau best practice.

<details open>
<summary><b>🐧 Linux / macOS</b></summary>

**Prasyarat:** Go 1.20+ (`go version`) dan binary `ssh` sudah terpasang.

```bash
# 1. Compile
go build -o hop .

# 2. Pasang ke PATH
cp hop ~/.local/bin/
# atau untuk seluruh user di sistem:
sudo cp hop /usr/local/bin/

# 3. Verifikasi
hop help
```

</details>

<details>
<summary><b>🪟 Windows</b></summary>

**Prasyarat:** Go compiler, OpenSSH Client aktif (default di Windows 10/11), PowerShell/Windows Terminal.

```powershell
# 1. Compile
go build -o hop.exe .

# 2. Pasang ke folder khusus
New-Item -ItemType Directory -Force -Path "C:\bin"
Copy-Item hop.exe -Destination "C:\bin\hop.exe"

# 3. Tambahkan C:\bin ke Environment Variable PATH
#    (Start Menu → "Edit the system environment variables")
```

> ⚠️ **Penting:** `hop` mencari config lewat variabel `HOME`, yang tidak selalu ada di Windows secara default. Tambahkan environment variable baru: `HOME` = `C:\Users\NamaUserAnda`.

```powershell
# 4. Verifikasi
hop.exe help
```

</details>

---

## 🔑 Autentikasi & Auto-Login

`hop` mendukung **dua metode autentikasi SSH** yang bisa dipakai secara terpisah maupun fallback:

| Metode | Keamanan | Auto-Login | Catatan |
|---|---|---|---|
| 🔑 **SSH Key** | ⭐⭐⭐ Sangat aman | ✅ Native (semua OS) | **Direkomendasikan** |
| 🔓 **Password** | ⭐ Cukup | ✅ Hanya via `sshpass` | Perlu install `sshpass` |

Dokumentasi dibagi jadi **dua bagian terpisah** di bawah ini:
- [Bagian A — SSH Key (generate + daftarkan ke server)](#bagian-a--ssh-key-recommended)
- [Bagian B — Password via sshpass (auto-login)](#bagian-b--password-via-sshpass-auto-login)

---

## 🟢 Bagian A — SSH Key (Recommended)

### 1. Generate SSH Key

```bash
# Linux / macOS — terminal biasa
ssh-keygen -t ed25519 -C "admin@server"

# Windows — PowerShell atau CMD
ssh-keygen -t ed25519 -C "admin@server"
```

Tekan Enter beberapa kali (kosongkan passphrase agar benar-benar auto-login tanpa prompt).
Hasil:

```
~/.ssh/id_ed25519      # private key (JANGAN dibagikan)
~/.ssh/id_ed25519.pub  # public key (yang didaftarkan ke server)
```

> 💡 **ED25519** lebih modern & aman dari RSA. Untuk kompatibilitas lama bisa pakai
> `ssh-keygen -t rsa -b 4096`.

### 2. Daftarkan Public Key ke Server

**Linux / macOS:**
```bash
# Cara paling mudah:
ssh-copy-id administrator@103.84.195.90

# Atau manual jika ssh-copy-id tidak ada:
cat ~/.ssh/id_ed25519.pub | ssh administrator@103.84.195.90 \
  "mkdir -p ~/.ssh && cat >> ~/.ssh/authorized_keys && chmod 600 ~/.ssh/authorized_keys"
```

**Windows (PowerShell):**
```powershell
# Copy isi public key, lalu tempel ke server:
type $env:USERPROFILE\.ssh\id_ed25519.pub | ssh administrator@103.84.195.90 "cat >> ~/.ssh/authorized_keys"
```

**Verifikasi** (harusnya tanpa password):
```bash
ssh administrator@103.84.195.90 "echo OK"
```

### 3. Konfigurasi `hop` dengan SSH Key

Saat `hop add`, isi **Path File SSH Key** dengan path private key:
```
Path File SSH Key (opsional) []: ~/.ssh/id_ed25519
```

Atau edit `config.yaml`:
```yaml
hosts:
  - alias: prod-brk
    host: 103.84.195.90
    user: administrator
    port: 22
    identity_file: ~/.ssh/id_ed25519
```

Sekarang `hop prod-brk` → **langsung login tanpa password** (native di semua OS, termasuk Windows).

---

## 🟡 Bagian B — Password via sshpass (Auto-Login)

> ⚠️ `sshpass` adalah utilitas pihak ketiga yang mengirim password ke `ssh` secara otomatis.
> **Tidak ada di Windows secara native** — lihat catatan Windows di bawah.

### 1. Install sshpass

```bash
# Ubuntu / Debian
sudo apt update && sudo apt install -y sshpass

# macOS (via Homebrew)
brew install hudochenkov/sshpass/sshpass

# Arch / Manjaro (pacman)
sudo pacman -S sshpass
```

Verifikasi:
```bash
sshpass -V
```

### 2. Konfigurasi `hop` dengan Password

Password **TIDAK lagi disimpan plaintext di `config.yaml`**. Saat `hop add`,
jawab **y** pada pertanyaan autentikasi password, lalu password dimasukkan:

```
Gunakan autentikasi password? (y/N) [N]: y
Password: ********
🔒 Password disimpan di OS keyring.
```

Password disimpan ke **OS keyring** (`gnome-keyring`/`secret-tool`) via
`secret-tool store` — ter-unlock otomatis saat Anda login desktop, sehingga
tidak perlu mengetik password tambahan saat `hop` connect. `config.yaml`
tetap 100% bebas credential plaintext.

> **Catatan:** `sshpass` tetap **wajib** — tapi sebagai *mekanisme pengiriman*
> password ke `ssh` (via environment variable `SSHPASS`, bukan file sementara),
> bukan untuk penyimpanan. Instruksi install di bagian atas.

Field `password:` di `config.yaml` **hanya** menjadi fallback legacy kalau
`secret-tool` tidak tersedia di sistem (pesan peringatan akan muncul).

### 3. Alur Autentikasi (Fallback)

Saat `hop` connect, urutan dicoba:

1. **`identity_file` diisi** → coba SSH key dulu
2. **Key gagal + `password` ada** → otomatis fallback ke password (`sshpass`)
3. **Hanya `password`** → langsung password auth
4. **Keduanya kosong** → SSH agent / default system

Jika `sshpass` **tidak terinstall**, `hop` akan tetap jalan tapi **meminta password manual**
(seperti `ssh` biasa). Pesan informatif akan muncul:

```
⚠ Host 'prod-brk' punya password terkonfigurasi, tapi sshpass tidak ada di PATH.
  Install sshpass untuk auto-login:
    Ubuntu/Debian: sudo apt install -y sshpass
    macOS:        brew install hudochenkov/sshpass/sshpass
  Tekan ENTER untuk lanjut (manual prompt), atau ketik 'i' untuk info.
```

### 3b. Migrasi Otomatis Password Lama ke Keyring

Jika `config.yaml` Anda (dari versi sebelum order-11) masih menyimpan password
plaintext di field `password:`, `hop` akan **otomatis memindahkannya ke OS keyring**
pada eksekusi pertama setelah update — tanpa kehilangan data. Config lama dicadangkan
ke `config.yaml.prekeyring.bak` dan field `password` dikosongkan. Setelah itu,
password tidak lagi muncul di `config.yaml`.

Untuk menghapus password dari keyring suatu host (misal agar kembali pakai prompt
manual dari `ssh`):

```bash
hop secret-remove <host-alias>
```

### 4. Windows & sshpass

`sshpass` **tidak tersedia native di Windows** — auto-login password di Windows
hanya bisa lewat **WSL**, **Git Bash**, atau **Cygwin** (tempat `sshpass` terinstall),
lalu jalankan `hop` di dalam environment tersebut. `hop.exe` native Windows akan
fallback ke prompt password manual. Untuk pengalaman terbaik di Windows, gunakan
**SSH Key** (Bagian A) yang bekerja native tanpa dependensi tambahan.

---

## 🛠️ Konfigurasi (`config.yaml`)

Dibuat otomatis saat `hop` pertama kali dijalankan.

| OS | Lokasi |
|---|---|
| Linux/macOS | `~/.config/hop/config.yaml` |
| Windows | `%HOME%\.config\hop\config.yaml` |

```yaml
hosts:
  - alias: dev-projek
    host: xx.xx.xx.xx
    user: root
    port: 22
    identity_file: ~/.ssh/id_rsa
    # password: secret123   # HANYA fallback legacy (bila secret-tool tak ada); sebaiknya kosong
    paths:
      - alias: projek1
        path: /var/www/html/projek1
      - alias: projek2
        path: /var/www/html/projek2
        command: php artisan serve
```

| Field | Keterangan |
|-------|------------|
| `alias` | Nama unik untuk memanggil host di CLI |
| `host` | IP atau hostname server SSH |
| `user` | Username login SSH |
| `port` | Port SSH (default `22`) |
| `identity_file` | Path ke SSH private key (opsional) |
| `password` | **Hanya fallback legacy** — dipakai kalau `secret-tool` tidak ada di sistem. Preferensi utama: disimpan di OS keyring (lihat `hop add`). Password lama di field ini otomatis dipindah ke keyring pada run pertama. |
| `paths[].alias` | Nama singkat direktori (dipakai saat connect) |
| `paths[].path` | Path tujuan di server |
| `paths[].command` | Perintah default yang dijalankan setelah cd (opsional) |

---

## 📑 Penggunaan

### Koneksi cepat

```bash
hop <host-alias>               # connect, otomatis masuk ke path pertama
hop <host-alias> <path-alias>  # connect ke path spesifik
```

```bash
hop dev-projek           # default path
hop dev-projek projek2   # path spesifik
```

Kalau alias tidak ditemukan, `hop` menampilkan daftar host/path yang valid — tidak perlu tebak-tebakan.

### Perintah default & override

Setiap path bisa memiliki perintah default (field `command` di config) yang otomatis dijalankan setelah masuk ke direktori, sebelum shell interaktif muncul.

```bash
hop dev-projek projek2           # cd + command default (kalau ada)
hop dev-projek projek2 -- htop   # override: jalankan htop, bukan command default
```

Jika path tidak ditemukan di server (fallback ke direktori default), perintah TIDAK dijalankan — Anda langsung masuk ke shell biasa.

### Manajemen host & path

| Command | Fungsi |
|---|---|
| `hop list` | Lihat semua host terdaftar |
| `hop add` | Tambah host baru (interaktif) |
| `hop edit <host>` | Ubah detail host |
| `hop remove <host>` | Hapus host |
| `hop doctor` | Cek koneksi ke semua host |
| `hop exec <host> -- <cmd>` | Jalankan command non-interaktif |
| `hop secret-remove <host>` | Hapus password host dari OS keyring |
| `hop path-list [<host>]` | Lihat semua path (semua host / spesifik) |
| `hop path-add <host>` | Tambah path baru ke host |
| `hop path-remove <host> <path>` | Hapus path dari host |

---

### 🩺 Hop Doctor
Cek koneksi ke semua host terdaftar sekaligus untuk memastikan server aktif.
```bash
hop doctor
# Output:
# ✓ dev-projek 1.2.3.4 — 45ms
# ✗ prod-server 5.6.7.8 — koneksi ditolak
# ...
# Selesai: 1 OK, 1 gagal dari 2 host.
```

### ⚡ Hop Exec (Non-Interaktif)
Jalankan perintah langsung di server tanpa masuk sesi shell interaktif. Berguna untuk automasi.
```bash
hop exec dev-projek -- echo "halo"
hop exec dev-projek projek1 -- ls -l
```

<details>
<summary>Contoh output <code>hop list</code></summary>

```
IP/HOST      ALIAS       USER   PORT   PATHS
-------      -----       ----   ----   -----
xx.xx.xx.xx  dev-projek  root   22     projek1, projek2
```

</details>

<details>
<summary>Contoh output <code>hop path-list</code></summary>

```
IP/HOST      ALIAS         PATH                       PATH ALIAS
-------      -----         ----                       ----------
xx.xx.xx.xx  dev-projek    /var/www/html/projek1      projek1
xx.xx.xx.xx  dev-projek    /var/www/html/projek2      projek2
```

</details>

---

## ⌨️ Autocomplete (Tab) di Bash

Aktifkan sekali, pakai selamanya:

```bash
hop completion bash >> ~/.bashrc
source ~/.bashrc
```

**Cara pakai:**
1. `hop <Tab><Tab>` → daftar semua command & host alias
2. `hop dev-<Tab>` → otomatis lengkap jadi `hop dev-projek `
3. `hop dev-projek <Tab>` → daftar path-alias milik host tersebut

> 💡 Kalau beberapa alias berbagi awalan sama (`projek1`, `projek2`), Tab akan mengisi sebanyak yang unik (`projek`), lalu Anda lanjutkan ketik pembedanya.

---

<div align="center">

Dibuat untuk mempercepat workflow development sehari-hari 🐇💨

</div>

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

## 💻 Instalasi

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
    paths:
      - alias: projek1
        path: /var/www/html/projek1
      - alias: projek2
        path: /var/www/html/projek2
        command: php artisan serve
```

| Field | Keterangan |
|---|---|
| `alias` | Nama unik untuk memanggil host di CLI |
| `host` | IP atau hostname server SSH |
| `user` | Username login SSH |
| `port` | Port SSH (default `22`) |
| `identity_file` | Path ke SSH private key (opsional) |
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

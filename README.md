# 🚀 hop — SSH Project Launcher & Directory Jumper

<p align="center">
  <img src="hop-rabbit.png" alt="hop logo" width="200" />
</p>

`hop` adalah alat CLI (Command Line Interface) ringan yang dirancang untuk mempermudah manajemen dan peluncuran koneksi SSH serta perpindahan direktori (*directory jumping*) langsung setelah masuk ke server. Proyek ini dibangun menggunakan bahasa Go, tanpa dependensi eksternal yang berat, serta menggunakan aplikasi `ssh` bawaan sistem secara langsung (*shell out*).

Alat ini sangat cocok sebagai pengganti cara manual mencari IP server di history terminal Anda (`history | grep <ip>`).

---

## ✨ Fitur Utama

1. **Multi-Path per Host**: Satu host server dapat memiliki beberapa direktori proyek yang beralias tersendiri.
2. **Auto Jumper**: Otomatis berpindah direktori (*cd*) ke path yang dituju saat berhasil terhubung melalui SSH.
3. **Penyajian Data Informatif**: Layout tabel yang rapi untuk perintah `hop list` dan `hop path-list`.
4. **Manajemen Interaktif**: Subcommand intuitif untuk menambah, mengubah, atau menghapus host dan path (`add`, `edit`, `remove`, `path-add`, `path-remove`).
5. **Migrasi Otomatis (v1 → v2)**: Otomatis mendeteksi, mencadangkan, dan mengonversi file konfigurasi dari aplikasi pendahulu (`devjump`) ke skema data baru `hop` tanpa kehilangan data.
6. **Autocompletion Pintar**: Dukungan *autocomplete* (menggunakan tombol `Tab`) untuk Bash yang mencakup perintah, alias host, hingga alias path.

---

## 💻 Panduan Instalasi

### 🐧 1. Cara Instalasi di OS Linux / macOS

#### Prasyarat
- Pastikan Go Compiler (minimal versi 1.20) sudah terinstal: `go version`
- Pastikan biner `ssh` terinstal di sistem Anda.

#### Langkah-langkah
1. Clone repositori ini atau masuk ke direktori proyek `hop`.
2. Lakukan kompilasi kode:
   ```bash
   go build -o hop .
   ```
3. Pindahkan biner hasil kompilasi ke folder biner lokal Anda (pastikan folder ini masuk ke dalam `$PATH` sistem Anda):
   ```bash
   cp hop ~/.local/bin/
   # atau untuk seluruh pengguna di sistem (membutuhkan akses root):
   sudo cp hop /usr/local/bin/
   ```
4. Verifikasi instalasi:
   ```bash
   hop help
   ```

---

### 🪟 2. Cara Instalasi di OS Windows

#### Prasyarat
- Pastikan Go Compiler terinstal di Windows.
- Pastikan fitur **OpenSSH Client** bawaan Windows sudah aktif (aktif secara bawaan di Windows 10/11).
- Terminal yang digunakan direkomendasikan adalah **PowerShell** atau **Windows Terminal**.

#### Langkah-langkah
1. Buka PowerShell atau Command Prompt, arahkan ke direktori proyek `hop`.
2. Compile proyek:
   ```powershell
   go build -o hop.exe .
   ```
3. Buat folder khusus untuk menyimpan biner CLI jika belum ada (misal `C:\bin`) dan salin biner ke folder tersebut:
   ```powershell
   New-Item -ItemType Directory -Force -Path "C:\bin"
   Copy-Item hop.exe -Destination "C:\bin\hop.exe"
   ```
4. Tambahkan folder tersebut ke Environment Variables Windows (`PATH`):
   - Buka Start Menu, cari **"Edit the system environment variables"**.
   - Klik **Environment Variables**.
   - Di bagian *User variables* atau *System variables*, cari variabel **Path**, klik **Edit**, lalu tambahkan `C:\bin`.
   - Simpan semua dialog.
5. **Penting (Konfigurasi Direktori `HOME` di Windows)**:
   Aplikasi `hop` mencari folder konfigurasi berdasarkan variabel lingkungan `HOME`. Secara *default*, Windows menggunakan `USERPROFILE` untuk direktori pengguna. Agar aplikasi berjalan dengan normal di Windows, tambahkan variabel lingkungan baru:
   - Nama variabel: `HOME`
   - Nilai variabel: `C:\Users\NamaUserAnda` (sesuaikan dengan folder user Anda).
6. Buka terminal baru dan verifikasi:
   ```powershell
   hop.exe help
   ```

---

## 🛠️ Konfigurasi file (`config.yaml`)

File konfigurasi akan dibuat secara otomatis saat `hop` dijalankan pertama kali.

* **Lokasi File (Linux/macOS)**: `~/.config/hop/config.yaml`
* **Lokasi File (Windows)**: `%HOME%\.config\hop\config.yaml` (atau `C:\Users\NamaUser\.config\hop\config.yaml`)

### Struktur Skema Konfigurasi (v2)

Berikut adalah contoh isi file konfigurasi `config.yaml`:

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
```

### Penjelasan Field:
- `alias`: Nama unik untuk memanggil host tersebut di CLI.
- `host`: Alamat IP atau hostname domain server SSH.
- `user`: Username yang digunakan untuk login SSH.
- `port`: Port SSH (default: 22).
- `paths`: Daftar direktori proyek di server terkait.
  - `alias`: Nama singkat untuk direktori tersebut (dipakai saat pemanggilan).
  - `path`: Direktori tujuan di server yang akan dituju otomatis setelah login.

---

## 📑 Panduan Penggunaan Perintah (Subcommands)

### 1. `hop list`
Menampilkan ringkasan seluruh host yang terdaftar beserta konfigurasinya secara terstruktur.
* **Perintah**:
  ```bash
  hop list
  ```
* **Contoh Output**:
  ```
  IP/HOST      ALIAS       USER   PORT   PATHS
  -------      -----       ----   ----   -----
  xx.xx.xx.xx  dev-projek  root   22     projek1, projek2
  ```

### 2. Koneksi ke Host (`hop <host-alias> [path-alias]`)
Menghubungkan terminal Anda ke server SSH tujuan dan masuk ke direktori proyek terkait.
* **Koneksi dengan Path Utama (Default)**:
  Jika Anda tidak menyertakan `path-alias`, sistem otomatis masuk ke path pertama yang terdaftar.
  ```bash
  hop dev-projek
  ```
* **Koneksi ke Path Spesifik**:
  ```bash
  hop dev-projek projek2
  ```
* **Penanganan Masalah**:
  Jika host belum memiliki direktori terdaftar, sistem akan memblokir koneksi dengan pesan:
  `Host 'dev-projek' belum memiliki path. Silakan tambahkan path terlebih dahulu.`

### 3. `hop add`
Menambahkan host baru secara interaktif melalui CLI.
* **Perintah**:
  ```bash
  hop add
  ```
* Anda akan dipandu untuk mengisi Alias host, Alamat Host, User, Port, dan minimal satu path beserta aliasnya.

### 4. `hop edit <host-alias>`
Mengubah detail informasi dari host yang sudah ada (seperti IP, port, atau user).
* **Perintah**:
  ```bash
  hop edit dev-projek
  ```

### 5. `hop remove <host-alias>`
Menghapus host beserta seluruh path di dalamnya dari daftar konfigurasi.
* **Perintah**:
  ```bash
  hop remove dev-projek
  ```

### 6. `hop path-list [<host-alias>]`
Menampilkan detail path untuk seluruh host, atau spesifik untuk host yang ditentukan.
* **Perintah**:
  ```bash
  hop path-list
  ```
* **Contoh Output**:
  ```
  IP/HOST      ALIAS         PATH                       PATH ALIAS
  -------      -----         ----                       ----------
  xx.xx.xx.xx  dev-projek    /var/www/html/projek1      projek1
  xx.xx.xx.xx  dev-projek    /var/www/html/projek2      projek2
  ```

### 7. `hop path-add <host-alias>`
Menambahkan alias direktori proyek baru ke host yang sudah ada.
* **Perintah**:
  ```bash
  hop path-add dev-projek
  ```

### 8. `hop path-remove <host-alias> <path-alias>`
Menghapus direktori proyek tertentu dari suatu host.
* **Perintah**:
  ```bash
  hop path-remove dev-projek projek2
  ```

---

## ⌨️ Konfigurasi Autocomplete (Tombol TAB) di Bash

Anda dapat mengaktifkan fitur pelengkapan otomatis (*autocompletion*) saat menekan tombol `Tab` di terminal Bash (Linux/macOS).

### Cara Mengaktifkan secara Permanen
Jalankan perintah berikut untuk menambahkan skrip pelengkap ke profil Bash Anda:
```bash
hop completion bash >> ~/.bashrc
source ~/.bashrc
```

### Cara Kerja Autocomplete (TAB / Double-TAB):
1. **Melengkapi Perintah & Host**: 
   Ketik `hop ` lalu tekan `Tab 2x` untuk melihat daftar subcommand dan host alias yang tersedia.
   *Contoh:* Ketik `hop dev-` lalu tekan `Tab` untuk langsung melengkapinya menjadi `hop dev-projek `.
2. **Melengkapi Path Alias**:
   Setelah host terisi (misalnya `hop dev-projek ` dengan spasi di akhir), tekan `Tab` kembali untuk memicu pelengkapan otomatis path alias milik host tersebut.
3. **Catatan Perilaku Bawaan Bash**:
   Jika seluruh pilihan path alias memiliki awalan kata yang sama (misalnya: `projek1` dan `projek2` yang sama-sama berawalan `projek`), menekan `Tab` akan langsung mengisi awalan terpanjang yang sama (`projek`). Anda hanya perlu melanjutkan mengetik lanjutannya (misal huruf **`2`** untuk `projek2`) lalu tekan `Tab` lagi untuk melengkapinya.

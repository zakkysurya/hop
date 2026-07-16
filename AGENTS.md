# AGENTS.md — hop

## Mode
Ponytail mode aktif: YAGNI, stdlib-first, single binary, tanpa dependency
berat kecuali benar-benar perlu (cobra boleh, database/ORM tidak boleh).

## Alur kerja
1. Baca order terbaru di .context/orders/
2. Implementasi sesuai ACTION di order tersebut, jangan menambah fitur
   di luar scope tanpa order baru
3. Jangan reinvent protokol SSH — selalu shell out ke binary `ssh` sistem
4. Setelah implementasi dan build sukses, copy binary ke ~/.local/bin/ (e.g. `cp ./hop ~/.local/bin/hop`) agar bisa langsung dipakai dari PATH
5. Laporkan ringkas: file apa saja yang dibuat/diubah
6. Setelah mempelajari sebuah order, simpan daftar Todos yang dihasilkan ke file .context/logs/order-N-todos.md (N = nomor order terkait, misal order-12-todos.md), agar riwayat pengerjaan tiap order selalu tercatat dan tidak hilang.
7. Setiap kali ada perubahan fitur/command/skema config, update juga README.md dan docs/index.html di commit yang sama supaya keduanya selalu sinkron dengan kondisi sistem terbaru.
   Termasuk memperbarui info **Update terakhir: DD Mon YYYY** yang ada di footer README.md dan docs/index.html — ganti sesuai tanggal hari itu.

## Struktur project
- main.go — entrypoint & command routing
- config.go — baca/tulis ~/.config/hop/config.yaml
- ssh.go — logic shell-out ke ssh
- secret.go — integrasi OS keyring (secret-tool) untuk password
- logging.go — pencatatan aktivitas untuk `hop logs`

## Larangan
- JANGAN jalankan command git yang destruktif (reset --hard, clean -fd, push --force)
  tanpa konfirmasi eksplisit dari user
- Nilai sensitif di-mask di output (misal cuma tampilkan *** atau jumlah karakter), bukan dicetak utuh

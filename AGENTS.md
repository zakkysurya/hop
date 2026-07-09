# AGENTS.md — devjump

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

## Struktur project
- main.go — entrypoint & command routing
- config.go — baca/tulis ~/.config/devjump/config.yaml
- ssh.go — logic shell-out ke ssh

## Larangan
- JANGAN jalankan command git yang destruktif (reset --hard, clean -fd, push --force)
  tanpa konfirmasi eksplisit dari user

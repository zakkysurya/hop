package main

import "fmt"

const bannerArt = `
██╗  ██╗ ██████╗ ██████╗                   ▄▄█████▄▄
██║  ██║██╔═══██╗██╔══██╗            ▄▄▄▄▄▄▄▄▄  ▄███████▄
███████║██║   ██║██████╔╝       ▀██████████████████████▀▀
██╔══██║██║   ██║██╔═══╝     ▄▄▄▄▄██████▀▀▀▀▀▀████████▄▄▄
██║  ██║╚██████╔╝██║      ▄█▀▄██▀▀▀▀▀▀▀   ───────      ▀▀
╚═╝  ╚═╝ ╚═════╝ ╚═╝      ▄▄█▀   ─────────
`

const description = `HOP — SSH Project Launcher & Directory Jumper

Kelola dan lompat ke direktori server SSH favorit Anda dengan cepat.
Cukup daftarkan host dan path-nya sekali, lalu hubungkan dengan satu perintah:

    hop <host-alias> [path-alias]

HOP men-delegasikan autentikasi SSH sepenuhnya ke OpenSSH client sistem.
Artinya, Anda bisa menggunakan SSH key maupun password — tergantung
konfigurasi server dan ketersediaan key di mesin Anda.

`

func printBanner() {
	fmt.Print(bannerArt)
	fmt.Print(description)
}

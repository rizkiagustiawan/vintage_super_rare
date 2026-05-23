# Carousell "Rare Find" Bot 🕵️‍♂️

Bot otomatis untuk memonitor barang-barang langka (vintage/archive fashion) di Carousell Indonesia menggunakan bahasa pemrograman Go.

## Fitur
- **Bypass Cloudflare**: Menggunakan `chromedp` (headless browser) untuk menghindari deteksi bot.
- **Dukungan Multi-Brand**: Memonitor ratusan brand sekaligus dari file `brand_list.txt`.
- **Notifikasi Telegram**: Memberikan alert instan lengkap dengan harga, penjual, dan link produk.
- **Anti-Spam**: Melakukan scan awal untuk menghindari notifikasi barang lama.

## Prasyarat
- [Go](https://go.dev/dl/) (versi 1.19 atau lebih baru)
- **Google Chrome** atau **Chromium** terinstal di sistem.
- Bot Telegram (dapatkan Token dari [@BotFather](https://t.me/botfather)).

## Instalasi
1. Clone repository ini.
2. Install dependensi:
   ```bash
   go mod tidy
   ```
3. Siapkan file daftar brand:
   Buat file `brand_list.txt` dan masukkan nama brand dipisahkan dengan koma (contoh: `Undercover, Number (N)ine, Yohji Yamamoto`).

## Konfigurasi
Buka `main.go` dan sesuaikan variabel berikut:
- `tgToken`: Token API Bot Telegram Anda.
- `chatID`: ID Chat/User Telegram Anda.
- `time.Sleep`: Sesuaikan interval scan untuk menghindari rate limiting.

## Penggunaan
Jalankan bot dengan perintah:
```bash
go run main.go
```

Bot akan melakukan "Initial Scan" terlebih dahulu untuk mendata barang lama, kemudian akan standby menunggu barang baru muncul.

## Disclaimer
Proyek ini dibuat untuk tujuan pembelajaran. Penggunaan bot untuk scraping secara masif dapat melanggar Ketentuan Layanan (ToS) Carousell. Gunakan dengan bijak dan atur interval waktu yang wajar.

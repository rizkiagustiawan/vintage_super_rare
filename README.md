# Carousell "Perfect Hunter" Bot 💎🕵️‍♂️

Bot otomatis profesional untuk memburu barang-barang "Super Rare" (vintage/archive fashion) di Carousell Indonesia menggunakan Go.

## 🚀 Fitur Unggulan (Pro Version)
- **Persistence (Database JSON)**: Bot mengingat barang yang sudah dilihat dalam file `seen_db.json`. Jika bot restart, tidak perlu scan ulang barang lama.
- **Smart Filtering (Blacklist)**: Otomatis mengabaikan barang dengan judul `repro`, `bootleg`, `fake`, `custom`, dll.
- **VPS Optimized**: Manajemen memori yang efisien (membuka/menutup browser per brand) dan anti-ban delay.
- **Bypass Cloudflare**: Menggunakan headless browser engine terbaru untuk menembus proteksi bot.
- **Telegram Alert Instan**: Notifikasi lengkap dengan Nama Brand, Harga, Penjual, dan Link.

## 🛠️ Prasyarat
- **Go** (v1.19+)
- **Google Chrome** atau **Chromium** (Wajib terinstal di sistem).
- **Telegram Bot Token** & **Chat ID**.

## 📥 Instalasi
1. Clone repository.
2. Install dependensi:
   ```bash
   go mod tidy
   ```
3. Siapkan daftar brand di file `brand_list.txt` (pisahkan dengan koma).

## ⚙️ Konfigurasi (Environment Variables)
Demi keamanan, bot tidak lagi menyimpan token di dalam kode. Set variabel berikut di terminal atau VPS Anda:
```bash
export TELEGRAM_TOKEN="your_bot_token_here"
export TELEGRAM_CHAT_ID="your_chat_id_here"
```

## 🏃 Cara Menjalankan
```bash
go run main.go
```

## 📁 Struktur Proyek
- `main.go`: Mesin utama bot.
- `brand_list.txt`: Daftar brand yang ingin dipantau.
- `seen_db.json`: Database ID barang (otomatis dibuat oleh bot).
- `.gitignore`: Memastikan file database dan binary tidak ter-push ke Git.

## ⚠️ Disclaimer
Gunakan dengan bijak. Atur interval `CycleDelay` dan `MinDelay` di `main.go` agar tidak terlalu agresif dan menyebabkan IP Anda terblokir oleh pihak marketplace.

# GarageFlow — Motorcycle Garage Inventory Management

MVP aplikasi manajemen inventory bengkel motor: pencarian lokasi sparepart, alert stok minimum, restock dengan riwayat pergerakan stok, chart omzet harian dan customer selesai, service board dengan sparepart yang otomatis menjadi penjualan saat job selesai, register/login/logout, role manager/mechanic.

Database adalah **SQLite** (satu file) yang dibuat otomatis saat API start, dan katalog sparepart berasal dari JSON seed `internal/database/parts_seed.json`.

## Stack
- Backend: Go, Gin, GORM, SQLite, JWT, bcrypt
- Frontend: React + Vite, React Router, Axios, Recharts
- API: REST `/api/v1`
- Version control: Git

## Struktur folder
Semua kode Go berada di root project agar sejajar dengan `go.mod` dan `go.sum`:

```text
garage inventory management/
├── main.go              # entrypoint API (package main) — go run .
├── go.mod
├── go.sum
├── internal/            # packages internal (hanya dipakai dari root)
│   ├── config/          # pembacaan env
│   ├── database/        # koneksi, migrasi, seed (go:embed parts_seed.json)
│   ├── handlers/        # handler HTTP
│   ├── middleware/      # JWT + role check
│   └── models/          # model GORM
├── frontend/            # React + Vite
├── garage.db            # SQLite (dibuat otomatis saat API start)
├── .env.example
├── README.md
└── GarageFlow-Full-Project.md
```

Catatan: direktori `internal/` wajib berada di root — jika masih diletakkan di `backend/internal/`, maka `main.go` di root tidak boleh mengimpornya (aturan visibilitas `internal` Go).

## Tampilan & tema
Frontend memakai tema *motorcycle garage* yang dibangun sebagai lapisan CSS override di bagian akhir `frontend/src/styles.css`, sehingga style dasar tetap utuh dan perubahan visual mudah dilacak per blok.

**Identitas visual**
- Palet orange/graphite, tekstur titik halus pada body, aksen hazard stripe di sidebar, modal, kartu, dan kolom service.
- Tipografi: *Chakra Petch* untuk judul/tombol/label kerja, *Share Tech Mono* untuk angka digital, SKU, dan tag teknis, DM Sans/Manrope untuk teks isi.
- Nomor stencil besar **01–04** sebagai watermark tiap halaman dan eyebrow berbentuk stiker miring yang bergeser saat hover.

**Instrumen & data**
- Kartu statistik dashboard berupa panel instrumen gelap: readout mono bercahaya, baut sudut, hazard stripe atas, dan LED berjalan.
- Ticker **HEADS UP** berjalan menampilkan item stok kritis — hover untuk jeda, klik untuk membuka daftar stok menipis.
- Chart omzet berupa *area* bergradien dengan animasi menggambar garis, bar "customer selesai" bergradien hijau, dan tooltip bergaya alat ukur.
- Badge status bergaya stiker yang miring, tabel ledger dengan sapuan kecepatan saat baris disorot.

**Service board**
- Kolom dengan header gelap + hazard stripe, kartu job bergaya *work-order ticket* dengan strip warna status di kiri dan cap stempel miring: `ANTRIAN`, `DIPROSES`, `PART ORDER`, `SELESAI`.

**Gerak & interaksi**
- Ikon tombol bergerak (restock memantul, jual mencetak, refresh berputar, sapuan cahaya pada menu), counter angka animasi, skeleton loading, pulse status, transisi halaman, toast notifikasi, dan tombol utama dengan bayangan tekan 3D.
- Kartu statistik dan ticker dapat diklik untuk berpindah halaman; chip SKU dapat diklik untuk menyalin SKU ke clipboard.
- Animasi otomatis dinonaktifkan bila sistem memakai `prefers-reduced-motion`.
- Layout responsif: pada layar ≤900px form penjualan menumpuk dan tabel inventory/penjualan berubah menjadi kartu agar tombol aksi tetap terlihat; grid statistik menyesuaikan (5 kolom ≥1151px, pola 3+2 pada 801–1150px, 2 kolom dengan omzet melebar ≤800px).

## Prasyarat
- Go 1.23+
- Node.js 20+

## Jalankan lokal

### 1. Backend
Dari root project:

```bash
# Linux/macOS
JWT_SECRET="ganti-dengan-secret-acak-yang-panjang" go run .
```

```powershell
# Windows PowerShell
$env:JWT_SECRET="ganti-dengan-secret-acak-yang-panjang"
go run .
```

Saat start, API otomatis:
1. membuat file `garage.db` (SQLite) di root project,
2. menjalankan `AutoMigrate`,
3. mengisi katalog **78 sparepart** dari `internal/database/parts_seed.json` (11 kategori, 10 item stok kritis) plus akun demo, 3 service job aktif, dan contoh aktivitas 7 hari terakhir (27 penjualan dan 10 service selesai) bila database masih kosong.

Health check: http://localhost:8080/health

### 2. Frontend
Buka terminal baru:
```bash
cd frontend
npm install
npm run dev
```
Buka http://localhost:5173

Jika API bukan di `http://localhost:8080/api/v1`, buat `frontend/.env`:
```env
VITE_API_URL=http://localhost:8080/api/v1
```

Build produksi:
```bash
cd frontend
npm run build     # hasil di frontend/dist (CSS ± 55 kB, JS ± 660 kB belum tergzip)
```

## Akun demo
- Pengelola: `manager@garage.local` / `Garage123!`
- Montir: `mechanic@garage.local` / `Garage123!`

**Ganti/hapus akun demo dan password sebelum deploy.** Registrasi publik sengaja hanya membuat role `mechanic`; jangan membuka endpoint pembuatan manager ke publik. Buat manager pertama melalui seed atau proses admin tepercaya.

## Konfigurasi environment
| Variabel | Default | Keterangan |
|---|---|---|
| `APP_PORT` | `8080` | Port API |
| `GIN_MODE` | `debug` | Mode Gin (`release` untuk produksi) |
| `DB_PATH` | `garage.db` | Lokasi file SQLite |
| `JWT_SECRET` | `dev-only-change-me` | **Wajib diganti** |
| `JWT_EXPIRES_HOURS` | `24` | Umur token |
| `SEED_DEMO` | `true` | Seed akun demo + katalog 78 sparepart |
| `CORS_ORIGINS` | `http://localhost:5173,http://127.0.0.1:5173` | Origin frontend yang diizinkan (pisahkan dengan koma) — ganti dengan domain deploy |

## Seed data sparepart
`internal/database/parts_seed.json` berisi 78 produk (SKU unik) dengan field `sku`, `name`, `category`, `brand`, `location`, `stock`, `min_stock`, `unit_price`:

- Pelumas (8), Pengapian (8), Rem (10), Filter (8), Ban & Roda (8), Rantai & Gir (7), Kelistrikan (7), Mesin (8), Suspensi & Kaki-kaki (6), Body & Aksesoris (5), Cairan (3).
- 10 item punya `stock <= min_stock` sehingga langsung muncul di panel "Perlu restock".
- Seed bersifat idempoten: SKU yang sudah ada dilewati, SKU baru ditambahkan. Jadi menambah produk cukup edit JSON lalu restart API.
- File JSON di-embed ke binary (`go:embed`), jadi tetap terbaca dari direktori kerja mana pun.

## Contoh data 7 hari terakhir
Saat database masih kosong, seed juga membuat aktivitas demo agar dashboard dan halaman penjualan langsung terisi:

- 27 transaksi penjualan: 17 manual dan 10 berasal dari service yang selesai.
- 10 service job berstatus `completed` dalam 7 hari terakhir (1–2 customer per hari), plus 3 job aktif untuk service board.
- Chart omzet dan chart "customer selesai" terisi otomatis; data tidak diduplikasi saat restart.
- Hapus `garage.db`, `garage.db-wal`, dan `garage.db-shm` lalu jalankan ulang API untuk membuat ulang data demo dari nol.

## Fitur
- **Inventory:** cari nama, SKU, kategori, brand, dan lokasi gudang; filter stok menipis; pengelola dapat menambah/mengubah/menghapus data lewat API.
- **Restock:** pengelola menambah stok lewat modal (jumlah, catatan, tanggal masuk); setiap perubahan tercatat di riwayat pergerakan stok.
- **Tanggal transaksi:** penjualan, service, dan restock memakai input tanggal sehingga transaksi bisa dicatat mundur (backdate).
- **Alert stok:** status low-stock jika `stock <= min_stock`.
- **Penjualan:** catat transaksi, kurangi stok secara atomik (`UPDATE ... WHERE stock >= qty`), simpan total penjualan, tandai sumber transaksi `manual` atau `service`.
- **Sparepart service:** setiap service job dapat memiliki daftar sparepart. Saat status diubah menjadi `completed`, seluruh sparepart otomatis dicatat sebagai penjualan `service`, stok dikurangi, dan pergerakan stok dibuat. Mengubah status keluar dari `completed` membalik penjualan dan mengembalikan stok.
- **Dashboard:** total sparepart, low-stock, service aktif, customer selesai hari ini, omzet hari ini, chart omzet 7 hari, dan chart jumlah customer selesai per hari.
- **Service board:** queued, in progress, waiting parts, completed, cancelled.
- **UI interaktif:** tombol **Jual** di tabel membuka modal penjualan cepat dengan preview total, tombol **Restock** membuka modal stok masuk, pembuatan service order punya picker sparepart dengan total sementara, semua aksi sukses/gagal ditampilkan lewat toast serta error inline di dalam modal.
- **Pintasan:** kartu statistik dashboard dan ticker stok kritis dapat diklik untuk menuju halaman terkait, chip SKU dapat diklik untuk disalin.
- **Auth:** register/login/logout, bcrypt, JWT dengan expiry, route middleware. Frontend menyimpan token/user di `localStorage`.

## Endpoint REST
Semua endpoint selain register/login/health membutuhkan `Authorization: Bearer <token>`.

| Method | Endpoint | Akses |
|---|---|---|
| POST | `/api/v1/auth/register` | publik; role selalu mechanic |
| POST | `/api/v1/auth/login` | publik |
| POST | `/api/v1/auth/logout` | login |
| GET | `/api/v1/auth/me` | login |
| GET | `/api/v1/parts?search=oli&low_stock=true` | login |
| GET | `/api/v1/parts/:id` | login |
| POST | `/api/v1/parts` | manager |
| PUT | `/api/v1/parts/:id` | manager |
| DELETE | `/api/v1/parts/:id` | manager |
| POST | `/api/v1/parts/:id/restock` | manager; body `{quantity, note, moved_at}` |
| GET | `/api/v1/stock-movements?part_id=1` | login |
| GET/POST | `/api/v1/sales` | login; `POST` menerima `sold_at` opsional |
| GET/POST | `/api/v1/services` | login; `POST` menerima `service_date` dan `parts[]` |
| PATCH | `/api/v1/services/:id/status` | login; `completed` membuat penjualan otomatis |
| GET | `/api/v1/dashboard` | login |

Contoh login:
```bash
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"manager@garage.local","password":"Garage123!"}'
```

Contoh restock dan service dengan sparepart:
```bash
# Tambah stok 10 unit (manager)
curl -X POST http://localhost:8080/api/v1/parts/1/restock \
  -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" \
  -d '{"quantity":10,"note":"Pembelian supplier","moved_at":"2026-10-27"}'

# Buat service dengan 2 unit sparepart (belum menjadi penjualan)
curl -X POST http://localhost:8080/api/v1/services \
  -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" \
  -d '{"customer_name":"Budi","motorcycle":"Vario 160","service_date":"2026-10-27","parts":[{"part_id":1,"quantity":2}]}'

# Selesaikan service: sparepart otomatis tercatat sebagai penjualan service
curl -X PATCH http://localhost:8080/api/v1/services/14/status \
  -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" \
  -d '{"status":"completed"}'
```

## Git
```bash
git init
git add .
git commit -m "feat: initial GarageFlow inventory management MVP"
git branch -M main
```

## Catatan sebelum production
- SQLite praktis untuk demo/single-user karena tanpa setup, tetapi tulis-serial (satu writer). Untuk trafik tinggi, pertimbangkan migrasi ke database client-server — datanya sudah terpisah rapi di model GORM, tinggal ganti driver.
- Token di localStorage rentan dicuri jika ada XSS. Untuk production pertimbangkan HttpOnly Secure SameSite cookie, CSP, HTTPS, rate limiting, audit log, backup DB, dan refresh/revocation strategy.
- Logout JWT stateless hanya menghapus token di browser; token curian tetap valid sampai expiry. Tambahkan denylist/token version jika perlu revoke di server.
- Atur CORS berdasarkan domain frontend aktual, gunakan secrets manager, batasi akses database, serta buat backup dan migration terkontrol.

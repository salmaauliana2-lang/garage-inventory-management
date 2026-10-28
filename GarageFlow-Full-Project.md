# GarageFlow — Motorcycle Garage Inventory Management

> Dokumentasi dan source code lengkap dalam satu file Markdown.
>
> Stack: Go, Gin, GORM, SQLite (default) / PostgreSQL (opsional), React, Vite, JWT, REST API, Git.

## 1. Tujuan aplikasi

1. Memudahkan pengguna menemukan lokasi sparepart di gudang.
2. Memastikan ketersediaan stok dengan alert jika `stock <= min_stock` (default minimum 3 buah).
3. Mencatat stok masuk (restock) beserta riwayat pergerakan stok.
4. Melihat tren omzet penjualan dan jumlah customer selesai per hari melalui chart.
5. Memantau progress service yang sedang berjalan.
6. Mencatat sparepart yang dipakai setiap service; saat job selesai, sparepart otomatis menjadi transaksi penjualan.
7. Menyediakan register, login, dan logout untuk pengelola dan montir.

## 2. Arsitektur sistem

```mermaid
flowchart LR
    U[Pengguna: Pengelola / Montir] <--> FE[React + Vite<br/>Port 5173]
    FE -- REST API + Bearer JWT --> API[Go + Gin<br/>Port 8080]
    API --> AUTH[JWT Middleware + Role Checks]
    API --> ORM[GORM]
    ORM <--> DB[(SQLite<br/>garage.db)]
    DB --- USERS[(users)]
    DB --- PARTS[(parts)]
    DB --- SALES[(sales)]
    DB --- JOBS[(service_jobs)]
    DB --- JOB_PARTS[(service_job_parts)]
    DB --- MOVEMENTS[(stock_movements)]
```

Database default adalah SQLite: satu file `garage.db`, tanpa Docker, dibuat otomatis saat API start. PostgreSQL tetap didukung — isi `DATABASE_URL` dengan DSN `postgres://...` (misalnya dari `docker compose up -d db`), dan `DB_PATH` diabaikan.

### Alur request

1. Pengguna login melalui React.
2. Backend memverifikasi password menggunakan bcrypt dan mengeluarkan JWT.
3. Frontend menyimpan JWT di `localStorage`.
4. Axios menambahkan `Authorization: Bearer <token>` pada request API.
5. Gin middleware memverifikasi JWT; handler mengakses database melalui GORM.
6. Frontend menampilkan data inventory, chart penjualan, chart customer selesai, dan status service.

## 3. Model database

```mermaid
erDiagram
    USERS ||--o{ SALES : creates
    USERS ||--o{ SERVICE_JOBS : assigned_mechanic
    PARTS ||--o{ SALES : sold_as
    SERVICE_JOBS ||--o{ SERVICE_JOB_PARTS : contains
    PARTS ||--o{ SERVICE_JOB_PARTS : used_in
    PARTS ||--o{ STOCK_MOVEMENTS : tracked_by

    USERS {
        uint id PK
        string name
        string email UK
        string password_hash
        string role
        timestamp created_at
    }
    PARTS {
        uint id PK
        string sku UK
        string name
        string category
        string brand
        string location
        int stock
        int min_stock
        int64 unit_price
    }
    SALES {
        uint id PK
        uint part_id FK
        int quantity
        int64 unit_price
        int64 total
        timestamp sold_at
        uint created_by FK
        string source
        uint service_job_id FK
        string work_order
    }
    SERVICE_JOBS {
        uint id PK
        string work_order UK
        string customer_name
        string motorcycle
        string plate_number
        string complaint
        uint mechanic_id FK
        string status
        timestamp started_at
        timestamp completed_at
    }
    SERVICE_JOB_PARTS {
        uint id PK
        uint service_job_id FK
        uint part_id FK
        int quantity
        int64 unit_price
        int64 subtotal
    }
    STOCK_MOVEMENTS {
        uint id PK
        uint part_id FK
        string type
        int quantity
        string note
        timestamp moved_at
    }
```

GORM menjalankan `AutoMigrate` saat backend mulai. Katalog awal berisi **78 sparepart** dari `internal/database/parts_seed.json` (di-embed dengan `go:embed`), plus akun demo, 3 service job aktif, dan contoh aktivitas 7 hari terakhir. Seed bersifat idempoten: SKU yang sudah ada dilewati, sedangkan aktivitas demo hanya dibuat saat tabel penjualan dan service selesai masih kosong.

## 4. Fitur

- **Inventory:** pencarian nama, SKU, kategori, brand, dan lokasi; filter stok kritis; pengelola dapat menambah, mengubah, dan menghapus sparepart.
- **Restock:** pengelola menambah stok lewat modal berisi jumlah, catatan, dan tanggal masuk. Setiap penambahan dan pengurangan stok tercatat di `stock_movements`.
- **Input tanggal:** penjualan, service, dan restock memakai date picker sehingga transaksi bisa dicatat untuk tanggal lampau.
- **Seed katalog:** 78 sparepart siap pakai (11 kategori), 10 di antaranya berstatus stok kritis.
- **Low-stock alert:** stok kritis bila jumlah stok kurang dari atau sama dengan batas minimum.
- **Penjualan:** transaksi mengurangi stok dan merekam total penjualan; sumber transaksi dibedakan menjadi `manual` dan `service`.
- **Sparepart service:** daftar sparepart per job disimpan di `service_job_parts`. Saat status job diubah menjadi `completed`, semua sparepart otomatis dibuatkan penjualan `service`, stok dikurangi, dan stock movement dicatat. Perubahan status keluar dari `completed` membalik penjualan dan mengembalikan stok.
- **Dashboard:** ringkasan item, low stock, service aktif, customer selesai hari ini, omzet hari ini, chart omzet 7 hari, dan chart customer selesai per hari.
- **Contoh data 7 hari:** seed membuat 27 penjualan (17 manual, 10 service) dan 10 service selesai agar chart langsung terisi.
- **Service board:** queued, in progress, waiting parts, completed, cancelled.
- **Autentikasi:** register/login/logout, bcrypt, JWT expiry, middleware dan role checks.
- **Hak akses:** register publik hanya menghasilkan role `mechanic`; pembuatan/perubahan sparepart dan restock dibatasi ke role `manager`.

## 5. Endpoint REST API

Semua endpoint kecuali `/health`, register, dan login membutuhkan `Authorization: Bearer <token>`.

| Method | Endpoint | Akses |
|---|---|---|
| GET | `/health` | Publik |
| POST | `/api/v1/auth/register` | Publik; role selalu mechanic |
| POST | `/api/v1/auth/login` | Publik |
| POST | `/api/v1/auth/logout` | Login |
| GET | `/api/v1/auth/me` | Login |
| GET | `/api/v1/parts?search=oli` | Login |
| GET | `/api/v1/parts?low_stock=true` | Login |
| GET | `/api/v1/parts/:id` | Login |
| POST | `/api/v1/parts` | Manager |
| PUT | `/api/v1/parts/:id` | Manager |
| DELETE | `/api/v1/parts/:id` | Manager |
| POST | `/api/v1/parts/:id/restock` | Manager |
| GET | `/api/v1/stock-movements` | Login |
| GET | `/api/v1/sales` | Login |
| POST | `/api/v1/sales` | Login; body boleh berisi `sold_at` |
| GET | `/api/v1/services` | Login |
| POST | `/api/v1/services` | Login; body boleh berisi `service_date` dan `parts[]` |
| PATCH | `/api/v1/services/:id/status` | Login; `completed` memicu penjualan otomatis |
| GET | `/api/v1/dashboard` | Login |

Contoh membuat service dengan sparepart:

```bash
curl -X POST http://localhost:8080/api/v1/services \
  -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" \
  -d '{"customer_name":"Budi","motorcycle":"Vario 160","service_date":"2026-10-27","parts":[{"part_id":1,"quantity":2}]}'
```

Saat `PATCH /api/v1/services/:id/status` dipanggil dengan body `{"status":"completed"}`, dua unit sparepart tersebut otomatis tercatat sebagai penjualan bertanda `source: "service"`. Jika job kemudian diubah ke status lain, penjualan dibalik dan stok dikembalikan.

Contoh restock:

```bash
curl -X POST http://localhost:8080/api/v1/parts/1/restock \
  -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" \
  -d '{"quantity":10,"note":"Pembelian supplier","moved_at":"2026-10-27"}'
```

## 6. Cara menjalankan

### Prasyarat

- Go 1.23+
- Node.js 20+
- Docker + Docker Compose **hanya jika** memilih PostgreSQL

### Database

Default **SQLite** — tidak ada langkah setup, `garage.db` dibuat otomatis di direktori kerja saat API start.

Opsional PostgreSQL:

```bash
docker compose up -d db
export DATABASE_URL="postgres://garage:garage_dev_password@localhost:5432/garage_inventory?sslmode=disable"
```

### Jalankan backend (Linux/macOS)

```bash
cp .env.example .env
# Ubah JWT_SECRET di .env. Go tidak otomatis memuat file .env,
# jadi export variabel ke shell sebelum menjalankan server.
set -a
source .env
set +a
go mod download
go run .
```

### Jalankan backend (Windows PowerShell)

```powershell
Copy-Item .env.example .env
$env:APP_PORT="8080"
$env:DB_PATH="garage.db"
$env:JWT_SECRET="ganti-dengan-secret-acak-yang-panjang"
$env:SEED_DEMO="true"
go mod download
go run .
```

Catatan: `DB_PATH` relatif terhadap direktori tempat perintah dijalankan. Untuk PostgreSQL, set `DATABASE_URL` dan biarkan `DB_PATH` kosong.

### Jalankan frontend

Buka terminal baru:

```bash
cd frontend
npm install
npm run dev
```

Buka `http://localhost:5173`. Backend berjalan di `http://localhost:8080`; health check `http://localhost:8080/health`.

### Akun demo

| Role | Email | Password |
|---|---|---|
| Pengelola | `manager@garage.local` | `Garage123!` |
| Montir | `mechanic@garage.local` | `Garage123!` |

Akun demo dibuat saat database kosong dan `SEED_DEMO=true`. Ganti atau hapus kredensial demo sebelum deploy.

## 7. Git / version control

```bash
git init
git add .
git commit -m "feat: initial GarageFlow inventory management MVP"
git branch -M main
```

## 8. Konfigurasi environment

File `.env.example`:

```dotenv
APP_PORT=8080
GIN_MODE=debug
# Default: SQLite, file lokal, tanpa Docker. Dibuat otomatis saat API start.
DB_PATH=garage.db
# Opsional: pakai PostgreSQL. Kalau DATABASE_URL diisi, DB_PATH diabaikan.
# DATABASE_URL=postgres://garage:garage_dev_password@localhost:5432/garage_inventory?sslmode=disable
JWT_SECRET=change-this-to-a-long-random-secret-before-deploying
JWT_EXPIRES_HOURS=24
SEED_DEMO=true
```

## 9. Catatan keamanan dan keterbatasan

- Ganti `JWT_SECRET` dan password demo sebelum penggunaan non-demo.
- SQLite cocok untuk demo/single-user (satu writer). Untuk multi-user dengan trafik tinggi, pindah ke PostgreSQL cukup dengan mengisi `DATABASE_URL`; file `garage.db`, `garage.db-wal`, dan `garage.db-shm` sudah masuk `.gitignore`.
- Token pada `localStorage` rentan dicuri jika aplikasi memiliki XSS. Untuk production, pertimbangkan cookie `HttpOnly`, `Secure`, `SameSite`, CSP, dan HTTPS.
- Logout JWT stateless hanya menghapus token dari browser; token yang dicuri tetap valid sampai expiry. Tambahkan denylist atau token version untuk revocation.
- Batasi CORS ke domain frontend yang benar, batasi akses PostgreSQL, gunakan backup, rate limiting, audit log, dan migration terkontrol.
- Project ini adalah MVP dan belum menjalani pengujian end-to-end di lingkungan pengguna.

---

# 10. Source code lengkap

Setiap file di bawah adalah file asli project. Salin isi code block sesuai path file jika ingin membangun ulang project dari Markdown ini.



## File: `.env.example`

```text
APP_PORT=8080
GIN_MODE=debug
# Default: SQLite, file lokal, tanpa Docker. Dibuat otomatis saat API start.
DB_PATH=garage.db
# Opsional: pakai PostgreSQL. Kalau DATABASE_URL diisi, DB_PATH diabaikan.
# DATABASE_URL=postgres://garage:garage_dev_password@localhost:5432/garage_inventory?sslmode=disable
JWT_SECRET=change-this-to-a-long-random-secret-before-deploying
JWT_EXPIRES_HOURS=24
SEED_DEMO=true
```


## File: `.gitignore`

```text
.env
node_modules/
dist/
coverage.out
*.log
*.db
*.db-shm
*.db-wal
.DS_Store
.idea/
.vscode/
```


## File: `README.md`

```markdown
# GarageFlow — Motorcycle Garage Inventory Management

MVP aplikasi manajemen inventory bengkel motor: pencarian lokasi sparepart, alert stok minimum, restock dengan riwayat pergerakan stok, chart omzet harian dan customer selesai, service board dengan sparepart yang otomatis menjadi penjualan saat job selesai, register/login/logout, role manager/mechanic.

Database default adalah **SQLite** (satu file, tanpa Docker). PostgreSQL tetap didukung lewat `DATABASE_URL` bila dibutuhkan.

## Stack
- Backend: Go, Gin, GORM, SQLite (default) / PostgreSQL (opsional), JWT, bcrypt
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
├── docker-compose.yml   # opsional: PostgreSQL
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
- Docker + Docker Compose **hanya jika** ingin memakai PostgreSQL

## Jalankan lokal

### 1. Backend (tanpa Docker)
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

### 3. Opsional — pakai PostgreSQL
```bash
docker compose up -d db
export DATABASE_URL="postgres://garage:garage_dev_password@localhost:5432/garage_inventory?sslmode=disable"
go run .
```
Jika `DATABASE_URL` kosong atau bukan DSN `postgres://`, backend memakai SQLite di `DB_PATH` (default `garage.db`).

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
| `DATABASE_URL` | _(kosong)_ | Isi DSN `postgres://...` untuk memakai PostgreSQL |
| `JWT_SECRET` | `dev-only-change-me` | **Wajib diganti** |
| `JWT_EXPIRES_HOURS` | `24` | Umur token |
| `SEED_DEMO` | `true` | Seed akun demo + katalog 78 sparepart |

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
- SQLite sangat praktis untuk demo/single-user, tetapi tulis-serial. Untuk multi-user dengan trafik tinggi, pindah ke PostgreSQL cukup dengan mengisi `DATABASE_URL`.
- Token di localStorage rentan dicuri jika ada XSS. Untuk production pertimbangkan HttpOnly Secure SameSite cookie, CSP, HTTPS, rate limiting, audit log, backup DB, dan refresh/revocation strategy.
- Logout JWT stateless hanya menghapus token di browser; token curian tetap valid sampai expiry. Tambahkan denylist/token version jika perlu revoke di server.
- Atur CORS berdasarkan domain frontend aktual, gunakan secrets manager, batasi akses database, serta buat backup dan migration terkontrol.
```


## File: `docker-compose.yml`

```yaml
services:
  db:
    image: postgres:16-alpine
    container_name: garage-postgres
    restart: unless-stopped
    environment:
      POSTGRES_USER: garage
      POSTGRES_PASSWORD: garage_dev_password
      POSTGRES_DB: garage_inventory
    ports:
      - "5432:5432"
    volumes:
      - garage_pgdata:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U garage -d garage_inventory"]
      interval: 5s
      timeout: 3s
      retries: 15

volumes:
  garage_pgdata:
```


## File: `go.mod`

```text
module motorcycle-garage-inventory

go 1.23

require (
	github.com/gin-contrib/cors v1.7.2
	github.com/gin-gonic/gin v1.10.0
	github.com/glebarez/sqlite v1.11.0
	github.com/golang-jwt/jwt/v5 v5.2.1
	golang.org/x/crypto v0.31.0
	gorm.io/driver/postgres v1.5.11
	gorm.io/gorm v1.25.12
)

require (
	github.com/bytedance/sonic v1.11.6 // indirect
	github.com/bytedance/sonic/loader v0.1.1 // indirect
	github.com/cloudwego/base64x v0.1.4 // indirect
	github.com/cloudwego/iasm v0.2.0 // indirect
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/gabriel-vasile/mimetype v1.4.3 // indirect
	github.com/gin-contrib/sse v0.1.0 // indirect
	github.com/glebarez/go-sqlite v1.21.2 // indirect
	github.com/go-playground/locales v0.14.1 // indirect
	github.com/go-playground/universal-translator v0.18.1 // indirect
	github.com/go-playground/validator/v10 v10.20.0 // indirect
	github.com/goccy/go-json v0.10.2 // indirect
	github.com/google/uuid v1.3.0 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20221227161230-091c0ba34f0a // indirect
	github.com/jackc/pgx/v5 v5.5.5 // indirect
	github.com/jackc/puddle/v2 v2.2.1 // indirect
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/jinzhu/now v1.1.5 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/klauspost/cpuid/v2 v2.2.7 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/leodido/go-urn v1.4.0 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/pelletier/go-toml/v2 v2.2.2 // indirect
	github.com/remyoudompheng/bigfft v0.0.0-20230129092748-24d4a6f8daec // indirect
	github.com/twitchyliquid64/golang-asm v0.15.1 // indirect
	github.com/ugorji/go/codec v1.2.12 // indirect
	golang.org/x/arch v0.8.0 // indirect
	golang.org/x/net v0.25.0 // indirect
	golang.org/x/sync v0.10.0 // indirect
	golang.org/x/sys v0.28.0 // indirect
	golang.org/x/text v0.21.0 // indirect
	google.golang.org/protobuf v1.34.1 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	modernc.org/libc v1.22.5 // indirect
	modernc.org/mathutil v1.5.0 // indirect
	modernc.org/memory v1.5.0 // indirect
	modernc.org/sqlite v1.23.1 // indirect
)
```


## File: `go.sum`

```text
github.com/bytedance/sonic v1.11.6 h1:oUp34TzMlL+OY1OUWxHqsdkgC/Zfc85zGqw9siXjrc0=
github.com/bytedance/sonic v1.11.6/go.mod h1:LysEHSvpvDySVdC2f87zGWf6CIKJcAvqab1ZaiQtds4=
github.com/bytedance/sonic/loader v0.1.1 h1:c+e5Pt1k/cy5wMveRDyk2X4B9hF4g7an8N3zCYjJFNM=
github.com/bytedance/sonic/loader v0.1.1/go.mod h1:ncP89zfokxS5LZrJxl5z0UJcsk4M4yY2JpfqGeCtNLU=
github.com/cloudwego/base64x v0.1.4 h1:jwCgWpFanWmN8xoIUHa2rtzmkd5J2plF/dnLS6Xd/0Y=
github.com/cloudwego/base64x v0.1.4/go.mod h1:0zlkT4Wn5C6NdauXdJRhSKRlJvmclQ1hhJgA0rcu/8w=
github.com/cloudwego/iasm v0.2.0 h1:1KNIy1I1H9hNNFEEH3DVnI4UujN+1zjpuk6gwHLTssg=
github.com/cloudwego/iasm v0.2.0/go.mod h1:8rXZaNYT2n95jn+zTI1sDr+IgcD2GVs0nlbbQPiEFhY=
github.com/creack/pty v1.1.9/go.mod h1:oKZEueFk5CKHvIhNR5MUki03XCEU+Q6VDXinZuGJ33E=
github.com/davecgh/go-spew v1.1.0/go.mod h1:J7Y8YcW2NihsgmVo/mv3lAwl/skON4iLHjSsI+c5H38=
github.com/davecgh/go-spew v1.1.1 h1:vj9j/u1bqnvCEfJOwUhtlOARqs3+rkHYY13jYWTU97c=
github.com/davecgh/go-spew v1.1.1/go.mod h1:J7Y8YcW2NihsgmVo/mv3lAwl/skON4iLHjSsI+c5H38=
github.com/dustin/go-humanize v1.0.1 h1:GzkhY7T5VNhEkwH0PVJgjz+fX1rhBrR7pRT3mDkpeCY=
github.com/dustin/go-humanize v1.0.1/go.mod h1:Mu1zIs6XwVuF/gI1OepvI0qD18qycQx+mFykh5fBlto=
github.com/gabriel-vasile/mimetype v1.4.3 h1:in2uUcidCuFcDKtdcBxlR0rJ1+fsokWf+uqxgUFjbI0=
github.com/gabriel-vasile/mimetype v1.4.3/go.mod h1:d8uq/6HKRL6CGdk+aubisF/M5GcPfT7nKyLpA0lbSSk=
github.com/gin-contrib/cors v1.7.2 h1:oLDHxdg8W/XDoN/8zamqk/Drgt4oVZDvaV0YmvVICQw=
github.com/gin-contrib/cors v1.7.2/go.mod h1:SUJVARKgQ40dmrzgXEVxj2m7Ig1v1qIboQkPDTQ9t2E=
github.com/gin-contrib/sse v0.1.0 h1:Y/yl/+YNO8GZSjAhjMsSuLt29uWRFHdHYUb5lYOV9qE=
github.com/gin-contrib/sse v0.1.0/go.mod h1:RHrZQHXnP2xjPF+u1gW/2HnVO7nvIa9PG3Gm+fLHvGI=
github.com/gin-gonic/gin v1.10.0 h1:nTuyha1TYqgedzytsKYqna+DfLos46nTv2ygFy86HFU=
github.com/gin-gonic/gin v1.10.0/go.mod h1:4PMNQiOhvDRa013RKVbsiNwoyezlm2rm0uX/T7kzp5Y=
github.com/glebarez/go-sqlite v1.21.2 h1:3a6LFC4sKahUunAmynQKLZceZCOzUthkRkEAl9gAXWo=
github.com/glebarez/go-sqlite v1.21.2/go.mod h1:sfxdZyhQjTM2Wry3gVYWaW072Ri1WMdWJi0k6+3382k=
github.com/glebarez/sqlite v1.11.0 h1:wSG0irqzP6VurnMEpFGer5Li19RpIRi2qvQz++w0GMw=
github.com/glebarez/sqlite v1.11.0/go.mod h1:h8/o8j5wiAsqSPoWELDUdJXhjAhsVliSn7bWZjOhrgQ=
github.com/go-playground/assert/v2 v2.2.0 h1:JvknZsQTYeFEAhQwI4qEt9cyV5ONwRHC+lYKSsYSR8s=
github.com/go-playground/assert/v2 v2.2.0/go.mod h1:VDjEfimB/XKnb+ZQfWdccd7VUvScMdVu0Titje2rxJ4=
github.com/go-playground/locales v0.14.1 h1:EWaQ/wswjilfKLTECiXz7Rh+3BjFhfDFKv/oXslEjJA=
github.com/go-playground/locales v0.14.1/go.mod h1:hxrqLVvrK65+Rwrd5Fc6F2O76J/NuW9t0sjnWqG1slY=
github.com/go-playground/universal-translator v0.18.1 h1:Bcnm0ZwsGyWbCzImXv+pAJnYK9S473LQFuzCbDbfSFY=
github.com/go-playground/universal-translator v0.18.1/go.mod h1:xekY+UJKNuX9WP91TpwSH2VMlDf28Uj24BCp08ZFTUY=
github.com/go-playground/validator/v10 v10.20.0 h1:K9ISHbSaI0lyB2eWMPJo+kOS/FBExVwjEviJTixqxL8=
github.com/go-playground/validator/v10 v10.20.0/go.mod h1:dbuPbCMFw/DrkbEynArYaCwl3amGuJotoKCe95atGMM=
github.com/goccy/go-json v0.10.2 h1:CrxCmQqYDkv1z7lO7Wbh2HN93uovUHgrECaO5ZrCXAU=
github.com/goccy/go-json v0.10.2/go.mod h1:6MelG93GURQebXPDq3khkgXZkazVtN9CRI+MGFi0w8I=
github.com/golang-jwt/jwt/v5 v5.2.1 h1:OuVbFODueb089Lh128TAcimifWaLhJwVflnrgM17wHk=
github.com/golang-jwt/jwt/v5 v5.2.1/go.mod h1:pqrtFR0X4osieyHYxtmOUWsAWrfe1Q5UVIyoH402zdk=
github.com/google/go-cmp v0.5.5 h1:Khx7svrCpmxxtHBq5j2mp/xVjsi8hQMfNLvJFAlrGgU=
github.com/google/go-cmp v0.5.5/go.mod h1:v8dTdLbMG2kIc/vJvl+f65V22dbkXbowE6jgT/gNBxE=
github.com/google/gofuzz v1.0.0/go.mod h1:dBl0BpW6vV/+mYPU4Po3pmUjxk6FQPldtuIdl/M65Eg=
github.com/google/pprof v0.0.0-20221118152302-e6195bd50e26 h1:Xim43kblpZXfIBQsbuBVKCudVG457BR2GZFIz3uw3hQ=
github.com/google/pprof v0.0.0-20221118152302-e6195bd50e26/go.mod h1:dDKJzRmX4S37WGHujM7tX//fmj1uioxKzKxz3lo4HJo=
github.com/google/uuid v1.3.0 h1:t6JiXgmwXMjEs8VusXIJk2BXHsn+wx8BZdTaoZ5fu7I=
github.com/google/uuid v1.3.0/go.mod h1:TIyPZe4MgqvfeYDBFedMoGGpEw/LqOeaOT+nhxU+yHo=
github.com/jackc/pgpassfile v1.0.0 h1:/6Hmqy13Ss2zCq62VdNG8tM1wchn8zjSGOBJ6icpsIM=
github.com/jackc/pgpassfile v1.0.0/go.mod h1:CEx0iS5ambNFdcRtxPj5JhEz+xB6uRky5eyVu/W2HEg=
github.com/jackc/pgservicefile v0.0.0-20221227161230-091c0ba34f0a h1:bbPeKD0xmW/Y25WS6cokEszi5g+S0QxI/d45PkRi7Nk=
github.com/jackc/pgservicefile v0.0.0-20221227161230-091c0ba34f0a/go.mod h1:5TJZWKEWniPve33vlWYSoGYefn3gLQRzjfDlhSJ9ZKM=
github.com/jackc/pgx/v5 v5.5.5 h1:amBjrZVmksIdNjxGW/IiIMzxMKZFelXbUoPNb+8sjQw=
github.com/jackc/pgx/v5 v5.5.5/go.mod h1:ez9gk+OAat140fv9ErkZDYFWmXLfV+++K0uAOiwgm1A=
github.com/jackc/puddle/v2 v2.2.1 h1:RhxXJtFG022u4ibrCSMSiu5aOq1i77R3OHKNJj77OAk=
github.com/jackc/puddle/v2 v2.2.1/go.mod h1:vriiEXHvEE654aYKXXjOvZM39qJ0q+azkZFrfEOc3H4=
github.com/jinzhu/inflection v1.0.0 h1:K317FqzuhWc8YvSVlFMCCUb36O/S9MCKRDI7QkRKD/E=
github.com/jinzhu/inflection v1.0.0/go.mod h1:h+uFLlag+Qp1Va5pdKtLDYj+kHp5pxUVkryuEj+Srlc=
github.com/jinzhu/now v1.1.5 h1:/o9tlHleP7gOFmsnYNz3RGnqzefHA47wQpKrrdTIwXQ=
github.com/jinzhu/now v1.1.5/go.mod h1:d3SSVoowX0Lcu0IBviAWJpolVfI5UJVZZ7cO71lE/z8=
github.com/json-iterator/go v1.1.12 h1:PV8peI4a0ysnczrg+LtxykD8LfKY9ML6u2jnxaEnrnM=
github.com/json-iterator/go v1.1.12/go.mod h1:e30LSqwooZae/UwlEbR2852Gd8hjQvJoHmT4TnhNGBo=
github.com/klauspost/cpuid/v2 v2.0.9/go.mod h1:FInQzS24/EEf25PyTYn52gqo7WaD8xa0213Md/qVLRg=
github.com/klauspost/cpuid/v2 v2.2.7 h1:ZWSB3igEs+d0qvnxR/ZBzXVmxkgt8DdzP6m9pfuVLDM=
github.com/klauspost/cpuid/v2 v2.2.7/go.mod h1:Lcz8mBdAVJIBVzewtcLocK12l3Y+JytZYpaMropDUws=
github.com/knz/go-libedit v1.10.1/go.mod h1:MZTVkCWyz0oBc7JOWP3wNAzd002ZbM/5hgShxwh4x8M=
github.com/kr/pretty v0.3.0 h1:WgNl7dwNpEZ6jJ9k1snq4pZsg7DOEN8hP9Xw0Tsjwk0=
github.com/kr/pretty v0.3.0/go.mod h1:640gp4NfQd8pI5XOwp5fnNeVWj67G7CFk/SaSQn7NBk=
github.com/kr/text v0.2.0 h1:5Nx0Ya0ZqY2ygV366QzturHI13Jq95ApcVaJBhpS+AY=
github.com/kr/text v0.2.0/go.mod h1:eLer722TekiGuMkidMxC/pM04lWEeraHUUmBw8l2grE=
github.com/leodido/go-urn v1.4.0 h1:WT9HwE9SGECu3lg4d/dIA+jxlljEa1/ffXKmRjqdmIQ=
github.com/leodido/go-urn v1.4.0/go.mod h1:bvxc+MVxLKB4z00jd1z+Dvzr47oO32F/QSNjSBOlFxI=
github.com/mattn/go-isatty v0.0.20 h1:xfD0iDuEKnDkl03q4limB+vH+GxLEtL/jb4xVJSWWEY=
github.com/mattn/go-isatty v0.0.20/go.mod h1:W+V8PltTTMOvKvAeJH7IuucS94S2C6jfK/D7dTCTo3Y=
github.com/modern-go/concurrent v0.0.0-20180228061459-e0a39a4cb421/go.mod h1:6dJC0mAP4ikYIbvyc7fijjWJddQyLn8Ig3JB5CqoB9Q=
github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd h1:TRLaZ9cD/w8PVh93nsPXa1VrQ6jlwL5oN8l14QlcNfg=
github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd/go.mod h1:6dJC0mAP4ikYIbvyc7fijjWJddQyLn8Ig3JB5CqoB9Q=
github.com/modern-go/reflect2 v1.0.2 h1:xBagoLtFs94CBntxluKeaWgTMpvLxC4ur3nMaC9Gz0M=
github.com/modern-go/reflect2 v1.0.2/go.mod h1:yWuevngMOJpCy52FWWMvUC8ws7m/LJsjYzDa0/r8luk=
github.com/pelletier/go-toml/v2 v2.2.2 h1:aYUidT7k73Pcl9nb2gScu7NSrKCSHIDE89b3+6Wq+LM=
github.com/pelletier/go-toml/v2 v2.2.2/go.mod h1:1t835xjRzz80PqgE6HHgN2JOsmgYu/h4qDAS4n929Rs=
github.com/pmezard/go-difflib v1.0.0 h1:4DBwDE0NGyQoBHbLQYPwSUPoCMWR5BEzIk/f1lZbAQM=
github.com/pmezard/go-difflib v1.0.0/go.mod h1:iKH77koFhYxTK1pcRnkKkqfTogsbg7gZNVY4sRDYZ/4=
github.com/remyoudompheng/bigfft v0.0.0-20200410134404-eec4a21b6bb0/go.mod h1:qqbHyh8v60DhA7CoWK5oRCqLrMHRGoxYCSS9EjAz6Eo=
github.com/remyoudompheng/bigfft v0.0.0-20230129092748-24d4a6f8daec h1:W09IVJc94icq4NjY3clb7Lk8O1qJ8BdBEF8z0ibU0rE=
github.com/remyoudompheng/bigfft v0.0.0-20230129092748-24d4a6f8daec/go.mod h1:qqbHyh8v60DhA7CoWK5oRCqLrMHRGoxYCSS9EjAz6Eo=
github.com/rogpeppe/go-internal v1.8.0 h1:FCbCCtXNOY3UtUuHUYaghJg4y7Fd14rXifAYUAtL9R8=
github.com/rogpeppe/go-internal v1.8.0/go.mod h1:WmiCO8CzOY8rg0OYDC4/i/2WRWAB6poM+XZ2dLUbcbE=
github.com/stretchr/objx v0.1.0/go.mod h1:HFkY916IF+rwdDfMAkV7OtwuqBVzrE8GR6GFx+wExME=
github.com/stretchr/objx v0.4.0/go.mod h1:YvHI0jy2hoMjB+UWwv71VJQ9isScKT/TqJzVSSt89Yw=
github.com/stretchr/objx v0.5.0/go.mod h1:Yh+to48EsGEfYuaHDzXPcE3xhTkx73EhmCGUpEOglKo=
github.com/stretchr/objx v0.5.2/go.mod h1:FRsXN1f5AsAjCGJKqEizvkpNtU+EGNCLh3NxZ/8L+MA=
github.com/stretchr/testify v1.3.0/go.mod h1:M5WIy9Dh21IEIfnGCwXGc5bZfKNJtfHm1UVUgZn+9EI=
github.com/stretchr/testify v1.7.0/go.mod h1:6Fq8oRcR53rry900zMqJjRRixrwX3KX962/h/Wwjteg=
github.com/stretchr/testify v1.7.1/go.mod h1:6Fq8oRcR53rry900zMqJjRRixrwX3KX962/h/Wwjteg=
github.com/stretchr/testify v1.8.0/go.mod h1:yNjHg4UonilssWZ8iaSj1OCr/vHnekPRkoO+kdMU+MU=
github.com/stretchr/testify v1.8.1/go.mod h1:w2LPCIKwWwSfY2zedu0+kehJoqGctiVI29o6fzry7u4=
github.com/stretchr/testify v1.8.4/go.mod h1:sz/lmYIOXD/1dqDmKjjqLyZ2RngseejIcXlSw2iwfAo=
github.com/stretchr/testify v1.9.0 h1:HtqpIVDClZ4nwg75+f6Lvsy/wHu+3BoSGCbBAcpTsTg=
github.com/stretchr/testify v1.9.0/go.mod h1:r2ic/lqez/lEtzL7wO/rwa5dbSLXVDPFyf8C91i36aY=
github.com/twitchyliquid64/golang-asm v0.15.1 h1:SU5vSMR7hnwNxj24w34ZyCi/FmDZTkS4MhqMhdFk5YI=
github.com/twitchyliquid64/golang-asm v0.15.1/go.mod h1:a1lVb/DtPvCB8fslRZhAngC2+aY1QWCk3Cedj/Gdt08=
github.com/ugorji/go/codec v1.2.12 h1:9LC83zGrHhuUA9l16C9AHXAqEV/2wBQ4nkvumAE65EE=
github.com/ugorji/go/codec v1.2.12/go.mod h1:UNopzCgEMSXjBc6AOMqYvWC1ktqTAfzJZUZgYf6w6lg=
golang.org/x/arch v0.0.0-20210923205945-b76863e36670/go.mod h1:5om86z9Hs0C8fWVUuoMHwpExlXzs5Tkyp9hOrfG7pp8=
golang.org/x/arch v0.8.0 h1:3wRIsP3pM4yUptoR96otTUOXI367OS0+c9eeRi9doIc=
golang.org/x/arch v0.8.0/go.mod h1:FEVrYAQjsQXMVJ1nsMoVVXPZg6p2JE2mx8psSWTDQys=
golang.org/x/crypto v0.31.0 h1:ihbySMvVjLAeSH1IbfcRTkD/iNscyz8rGzjF/E5hV6U=
golang.org/x/crypto v0.31.0/go.mod h1:kDsLvtWBEx7MV9tJOj9bnXsPbxwJQ6csT/x4KIN4Ssk=
golang.org/x/net v0.25.0 h1:d/OCCoBEUq33pjydKrGQhw7IlUPI2Oylr+8qLx49kac=
golang.org/x/net v0.25.0/go.mod h1:JkAGAh7GEvH74S6FOH42FLoXpXbE/aqXSrIQjXgsiwM=
golang.org/x/sync v0.10.0 h1:3NQrjDixjgGwUOCaF8w2+VYHv0Ve/vGYSbdkTa98gmQ=
golang.org/x/sync v0.10.0/go.mod h1:Czt+wKu1gCyEFDUtn0jG5QVvpJ6rzVqr5aXyt9drQfk=
golang.org/x/sys v0.5.0/go.mod h1:oPkhp1MJrh7nUepCBck5+mAzfO9JrbApNNgaTdGDITg=
golang.org/x/sys v0.6.0/go.mod h1:oPkhp1MJrh7nUepCBck5+mAzfO9JrbApNNgaTdGDITg=
golang.org/x/sys v0.28.0 h1:Fksou7UEQUWlKvIdsqzJmUmCX3cZuD2+P3XyyzwMhlA=
golang.org/x/sys v0.28.0/go.mod h1:/VUhepiaJMQUp4+oa/7Zr1D23ma6VTLIYjOOTFZPUcA=
golang.org/x/text v0.21.0 h1:zyQAAkrwaneQ066sspRyJaG9VNi/YJ1NfzcGB3hZ/qo=
golang.org/x/text v0.21.0/go.mod h1:4IBbMaMmOPCJ8SecivzSH54+73PCFmPWxNTLm+vZkEQ=
golang.org/x/xerrors v0.0.0-20200804184101-5ec99f83aff1 h1:go1bK/D/BFZV2I8cIQd1NKEZ+0owSTG1fDTci4IqFcE=
golang.org/x/xerrors v0.0.0-20200804184101-5ec99f83aff1/go.mod h1:I/5z698sn9Ka8TeJc9MKroUUfqBBauWjQqLJ2OPfmY0=
google.golang.org/protobuf v1.34.1 h1:9ddQBjfCyZPOHPUiPxpYESBLc+T8P3E+Vo4IbKZgFWg=
google.golang.org/protobuf v1.34.1/go.mod h1:c6P6GXX6sHbq/GpV6MGZEdwhWPcYBgnhAHhKbcUYpos=
gopkg.in/check.v1 v0.0.0-20161208181325-20d25e280405/go.mod h1:Co6ibVJAznAaIkqp8huTwlJQCZ016jof/cbN4VW5Yz0=
gopkg.in/check.v1 v1.0.0-20201130134442-10cb98267c6c h1:Hei/4ADfdWqJk1ZMxUNpqntNwaWcugrBjAiHlqqRiVk=
gopkg.in/check.v1 v1.0.0-20201130134442-10cb98267c6c/go.mod h1:JHkPIbrfpd72SG/EVd6muEfDQjcINNoR0C8j2r3qZ4Q=
gopkg.in/yaml.v3 v3.0.0-20200313102051-9f266ea9e77c/go.mod h1:K4uyk7z7BCEPqu6E+C64Yfv1cQ7kz7rIZviUmN+EgEM=
gopkg.in/yaml.v3 v3.0.1 h1:fxVm/GzAzEWqLHuvctI91KS9hhNmmWOoWu0XTYJS7CA=
gopkg.in/yaml.v3 v3.0.1/go.mod h1:K4uyk7z7BCEPqu6E+C64Yfv1cQ7kz7rIZviUmN+EgEM=
gorm.io/driver/postgres v1.5.11 h1:ubBVAfbKEUld/twyKZ0IYn9rSQh448EdelLYk9Mv314=
gorm.io/driver/postgres v1.5.11/go.mod h1:DX3GReXH+3FPWGrrgffdvCk3DQ1dwDPdmbenSkweRGI=
gorm.io/gorm v1.25.12 h1:I0u8i2hWQItBq1WfE0o2+WuL9+8L21K9e2HHSTE/0f8=
gorm.io/gorm v1.25.12/go.mod h1:xh7N7RHfYlNc5EmcI/El95gXusucDrQnHXe0+CgWcLQ=
modernc.org/libc v1.22.5 h1:91BNch/e5B0uPbJFgqbxXuOnxBQjlS//icfQEGmvyjE=
modernc.org/libc v1.22.5/go.mod h1:jj+Z7dTNX8fBScMVNRAYZ/jF91K8fdT2hYMThc3YjBY=
modernc.org/mathutil v1.5.0 h1:rV0Ko/6SfM+8G+yKiyI830l3Wuz1zRutdslNoQ0kfiQ=
modernc.org/mathutil v1.5.0/go.mod h1:mZW8CKdRPY1v87qxC/wUdX5O1qDzXMP5TH3wjfpga6E=
modernc.org/memory v1.5.0 h1:N+/8c5rE6EqugZwHii4IFsaJ7MUhoWX07J5tC/iI5Ds=
modernc.org/memory v1.5.0/go.mod h1:PkUhL0Mugw21sHPeskwZW4D6VscE/GQJOnIpCnW6pSU=
modernc.org/sqlite v1.23.1 h1:nrSBg4aRQQwq59JpvGEQ15tNxoO5pX/kUjcRNwSAGQM=
modernc.org/sqlite v1.23.1/go.mod h1:OrDj17Mggn6MhE+iPbBNf7RGKODDE9NFT0f3EwDzJqk=
nullprogram.com/x/optparse v1.0.0/go.mod h1:KdyPE+Igbe0jQUrVfMqDMeJQIJZEuyV7pjYmp6pbG50=
rsc.io/pdf v0.1.1/go.mod h1:n8OzWcQ6Sp37PL01nO98y4iUCRdTGarVfzxY20ICaU4=
```


## File: `main.go`

```go
package main

import (
	"log"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"motorcycle-garage-inventory/internal/config"
	"motorcycle-garage-inventory/internal/database"
	"motorcycle-garage-inventory/internal/handlers"
	"motorcycle-garage-inventory/internal/middleware"
)

func main() {
	cfg := config.Load()
	db, err := database.Connect(cfg.DatabaseURL, cfg.DBPath)
	if err != nil {
		log.Fatalf("database connection failed: %v", err)
	}
	if err := database.Migrate(db); err != nil {
		log.Fatalf("migration failed: %v", err)
	}
	if cfg.SeedDemo {
		if err := database.SeedDemo(db); err != nil {
			log.Printf("demo seed skipped: %v", err)
		}
	}

	if cfg.GinMode != "" {
		gin.SetMode(cfg.GinMode)
	}
	r := gin.Default()
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:5173", "http://127.0.0.1:5173"},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		AllowCredentials: false, MaxAge: 12 * time.Hour,
	}))
	r.GET("/health", func(c *gin.Context) { c.JSON(200, gin.H{"status": "ok"}) })

	h := handlers.New(db, cfg)
	api := r.Group("/api/v1")
	api.POST("/auth/register", h.Register)
	api.POST("/auth/login", h.Login)
	protected := api.Group("")
	protected.Use(middleware.Auth(cfg.JWTSecret))
	protected.POST("/auth/logout", h.Logout) // JWT is stateless; client discards token.
	protected.GET("/auth/me", h.Me)
	protected.GET("/parts", h.ListParts)
	protected.POST("/parts", middleware.RequireRole("manager"), h.CreatePart)
	protected.GET("/parts/:id", h.GetPart)
	protected.PUT("/parts/:id", middleware.RequireRole("manager"), h.UpdatePart)
	protected.DELETE("/parts/:id", middleware.RequireRole("manager"), h.DeletePart)
	protected.POST("/parts/:id/restock", middleware.RequireRole("manager"), h.RestockPart)
	protected.GET("/stock-movements", h.ListStockMovements)
	protected.GET("/sales", h.ListSales)
	protected.POST("/sales", h.CreateSale)
	protected.GET("/services", h.ListServices)
	protected.POST("/services", h.CreateService)
	protected.PATCH("/services/:id/status", h.UpdateServiceStatus)
	protected.GET("/dashboard", h.Dashboard)

	port := cfg.Port
	if port == "" {
		port = "8080"
	}
	log.Printf("API listening on :%s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatal(err)
	}
}
```


## File: `internal/config/config.go`

```go
package config

import (
	"os"
	"strconv"
)

type Config struct {
	Port, GinMode, DatabaseURL, DBPath, JWTSecret string
	JWTExpiresHours                               int
	SeedDemo                                      bool
}

func Load() Config {
	hours, _ := strconv.Atoi(env("JWT_EXPIRES_HOURS", "24"))
	return Config{
		Port:            env("APP_PORT", "8080"),
		GinMode:         env("GIN_MODE", "debug"),
		DatabaseURL:     env("DATABASE_URL", ""),
		DBPath:          env("DB_PATH", "garage.db"),
		JWTSecret:       env("JWT_SECRET", "dev-only-change-me"),
		JWTExpiresHours: hours,
		SeedDemo:        env("SEED_DEMO", "true") == "true",
	}
}

func env(k, fallback string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return fallback
}
```


## File: `internal/database/database.go`

```go
package database

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/glebarez/sqlite"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"motorcycle-garage-inventory/internal/models"
)

//go:embed parts_seed.json
var partsSeedJSON []byte

// Connect membuka database. Default-nya SQLite (file lokal, tanpa Docker).
// Isi DATABASE_URL dengan DSN postgres://... untuk memakai PostgreSQL.
func Connect(databaseURL, dbPath string) (*gorm.DB, error) {
	usePostgres := strings.HasPrefix(databaseURL, "postgres://") || strings.HasPrefix(databaseURL, "postgresql://")

	if dbPath == "" {
		dbPath = "garage.db"
	}

	var dialector gorm.Dialector
	if usePostgres {
		dialector = postgres.Open(databaseURL)
	} else {
		dialector = sqlite.Open(dbPath)
	}

	db, err := gorm.Open(dialector, &gorm.Config{})
	if err != nil {
		return nil, err
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	if usePostgres {
		sqlDB.SetMaxOpenConns(20)
		sqlDB.SetMaxIdleConns(5)
		sqlDB.SetConnMaxLifetime(time.Hour)
	} else {
		// SQLite hanya mendukung satu writer; WAL menjaga pembacaan tetap lancar.
		sqlDB.SetMaxOpenConns(1)
		if err := db.Exec("PRAGMA journal_mode=WAL").Error; err != nil {
			return nil, err
		}
		if err := db.Exec("PRAGMA busy_timeout=5000").Error; err != nil {
			return nil, err
		}
	}
	return db, nil
}

func Migrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&models.User{},
		&models.Part{},
		&models.ServiceJob{},
		&models.ServiceJobPart{},
		&models.Sale{},
		&models.StockMovement{},
	)
}

func SeedDemo(db *gorm.DB) error {
	if err := seedUsers(db); err != nil {
		return err
	}
	if err := seedParts(db); err != nil {
		return err
	}
	if err := seedJobs(db); err != nil {
		return err
	}
	return seedActivity(db)
}

func seedUsers(db *gorm.DB) error {
	var count int64
	db.Model(&models.User{}).Where("email = ?", "manager@garage.local").Count(&count)
	if count > 0 {
		return nil
	}
	hash, err := bcrypt.GenerateFromPassword([]byte("Garage123!"), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	manager := models.User{Name: "Demo Manager", Email: "manager@garage.local", PasswordHash: string(hash), Role: "manager"}
	mechanic := models.User{Name: "Demo Mechanic", Email: "mechanic@garage.local", PasswordHash: string(hash), Role: "mechanic"}
	if err := db.Create(&manager).Error; err != nil {
		return err
	}
	if err := db.Create(&mechanic).Error; err != nil {
		return err
	}
	return nil
}

// seedParts memuat katalog sparepart dari parts_seed.json. Aman dijalankan
// berulang: SKU yang sudah ada dilewati, SKU baru ditambahkan.
func seedParts(db *gorm.DB) error {
	var parts []models.Part
	if err := json.Unmarshal(partsSeedJSON, &parts); err != nil {
		return fmt.Errorf("decode parts seed: %w", err)
	}
	for i := range parts {
		if parts[i].MinStock == 0 {
			parts[i].MinStock = 3
		}
		var existing int64
		if err := db.Model(&models.Part{}).Where("sku = ?", parts[i].SKU).Count(&existing).Error; err != nil {
			return fmt.Errorf("check part %s: %w", parts[i].SKU, err)
		}
		if existing > 0 {
			continue
		}
		if err := db.Create(&parts[i]).Error; err != nil {
			return fmt.Errorf("seed part %s: %w", parts[i].SKU, err)
		}
	}
	return nil
}

func seedJobs(db *gorm.DB) error {
	var count int64
	db.Model(&models.ServiceJob{}).Count(&count)
	if count > 0 {
		return nil
	}
	var mechanic models.User
	if err := db.Where("email = ?", "mechanic@garage.local").First(&mechanic).Error; err != nil {
		return nil
	}
	now := time.Now()
	jobs := []models.ServiceJob{
		{WorkOrder: "WO-DEMO-001", CustomerName: "Budi Santoso", Motorcycle: "Honda Vario 160", PlateNumber: "AB 1234 CD", Complaint: "Servis berkala dan ganti oli", MechanicID: mechanic.ID, Status: "in_progress"},
		{WorkOrder: "WO-DEMO-002", CustomerName: "Siti Aminah", Motorcycle: "Yamaha NMAX", PlateNumber: "AB 5678 EF", Complaint: "Pemeriksaan rem", MechanicID: mechanic.ID, Status: "queued"},
		{WorkOrder: "WO-DEMO-003", CustomerName: "Rizky Pratama", Motorcycle: "Honda BeAT FI", PlateNumber: "AB 9911 GH", Complaint: "Lampu depan mati dan aki lemah", MechanicID: mechanic.ID, Status: "waiting_parts"},
	}
	for i := range jobs {
		if jobs[i].Status == "in_progress" {
			jobs[i].StartedAt = &now
		}
		if err := db.Create(&jobs[i]).Error; err != nil {
			return fmt.Errorf("seed service: %w", err)
		}
	}
	return nil
}

// seedActivity mengisi contoh transaksi 7 hari terakhir: penjualan manual dan
// service selesai beserta sparepart yang otomatis tercatat sebagai penjualan.
func seedActivity(db *gorm.DB) error {
	var sales, completed int64
	db.Model(&models.Sale{}).Count(&sales)
	db.Model(&models.ServiceJob{}).Where("status = ?", "completed").Count(&completed)
	if sales > 0 || completed > 0 {
		return nil
	}

	var parts []models.Part
	if err := db.Where("stock >= ?", 12).Order("id ASC").Find(&parts).Error; err != nil {
		return err
	}
	if len(parts) < 8 {
		return nil
	}
	var mechanic models.User
	if err := db.Where("email = ?", "mechanic@garage.local").First(&mechanic).Error; err != nil {
		return nil
	}

	customers := []struct{ name, motor, plate, complaint string }{
		{"Agus Setiawan", "Honda Vario 125", "AB 2145 KA", "Ganti oli dan cek rem depan"},
		{"Dewi Lestari", "Yamaha Mio M3", "AB 7781 LB", "Servis berkala 8.000 km"},
		{"Fajar Nugroho", "Honda BeAT FI", "AB 3320 MN", "Rem belakang kurang pakem"},
		{"Rina Marlina", "Yamaha NMAX", "AB 5544 PQ", "Ganti filter udara dan busi"},
		{"Hendra Wijaya", "Honda Scoopy", "AB 8899 RS", "Lampu depan mati"},
		{"Siti Rahayu", "Yamaha Aerox 155", "AB 1122 TU", "Bunyi kasar di area CVT"},
		{"Bayu Saputra", "Honda PCX 160", "AB 6677 VW", "Ganti kampas rem depan"},
	}

	now := time.Now()
	start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local).AddDate(0, 0, -6)

	// Kurangi stok hanya jika masih cukup; kalau tidak, item dilewati.
	takeStock := func(partID uint, qty int) bool {
		res := db.Model(&models.Part{}).Where("id = ? AND stock >= ?", partID, qty).
			UpdateColumn("stock", gorm.Expr("stock - ?", qty))
		return res.Error == nil && res.RowsAffected > 0
	}

	for d := 0; d < 7; d++ {
		day := start.AddDate(0, 0, d)

		// 1-2 customer selesai per hari.
		for j := 0; j < 1+d%2; j++ {
			part := parts[(d*3+j*2)%len(parts)]
			qty := 1
			if !takeStock(part.ID, qty) {
				continue
			}
			started := day.Add(9*time.Hour + time.Duration(j)*90*time.Minute)
			done := started.Add(75 * time.Minute)
			cust := customers[(d+j)%len(customers)]
			job := models.ServiceJob{
				WorkOrder:    fmt.Sprintf("WO-DEMO-%s-%02d", day.Format("20060102"), j+1),
				CustomerName: cust.name,
				Motorcycle:   cust.motor,
				PlateNumber:  cust.plate,
				Complaint:    cust.complaint,
				MechanicID:   mechanic.ID,
				Status:       "completed",
				StartedAt:    &started,
				CompletedAt:  &done,
				CreatedAt:    started,
			}
			if err := db.Create(&job).Error; err != nil {
				return fmt.Errorf("seed service activity: %w", err)
			}
			row := models.ServiceJobPart{ServiceJobID: job.ID, PartID: part.ID, Quantity: qty, UnitPrice: part.UnitPrice, Subtotal: part.UnitPrice * int64(qty)}
			if err := db.Create(&row).Error; err != nil {
				return fmt.Errorf("seed service part: %w", err)
			}
			sale := models.Sale{PartID: part.ID, Quantity: qty, UnitPrice: part.UnitPrice, Total: row.Subtotal, SoldAt: done, CreatedBy: mechanic.ID, Source: "service", ServiceJobID: &job.ID, WorkOrder: job.WorkOrder}
			if err := db.Create(&sale).Error; err != nil {
				return fmt.Errorf("seed service sale: %w", err)
			}
			if err := db.Create(&models.StockMovement{PartID: part.ID, Type: "service", Quantity: -qty, Note: "Service " + job.WorkOrder, CreatedBy: mechanic.ID, MovedAt: done}).Error; err != nil {
				return fmt.Errorf("seed service movement: %w", err)
			}
		}

		// 2-3 penjualan manual per hari.
		for k := 0; k < 2+d%2; k++ {
			part := parts[(d*5+k*4)%len(parts)]
			qty := 1
			if !takeStock(part.ID, qty) {
				continue
			}
			soldAt := day.Add(10*time.Hour + time.Duration(k)*2*time.Hour + 15*time.Minute)
			sale := models.Sale{PartID: part.ID, Quantity: qty, UnitPrice: part.UnitPrice, Total: part.UnitPrice * int64(qty), SoldAt: soldAt, CreatedBy: mechanic.ID, Source: "manual"}
			if err := db.Create(&sale).Error; err != nil {
				return fmt.Errorf("seed sale: %w", err)
			}
			if err := db.Create(&models.StockMovement{PartID: part.ID, Type: "sale", Quantity: -qty, Note: "Penjualan manual", CreatedBy: mechanic.ID, MovedAt: soldAt}).Error; err != nil {
				return fmt.Errorf("seed sale movement: %w", err)
			}
		}
	}
	return nil
}
```


## File: `internal/database/parts_seed.json`

```json
[
  {
    "sku": "OIL-MTX-10W40",
    "name": "Oli Mesin Motul 10W-40 1L",
    "category": "Pelumas",
    "brand": "Motul",
    "location": "Rak A-01",
    "stock": 24,
    "min_stock": 6,
    "unit_price": 95000
  },
  {
    "sku": "OIL-YML-MATIC",
    "name": "Oli Mesin Yamalube Matic 0.8L",
    "category": "Pelumas",
    "brand": "Yamalube",
    "location": "Rak A-02",
    "stock": 30,
    "min_stock": 8,
    "unit_price": 52000
  },
  {
    "sku": "OIL-AHM-MPX2",
    "name": "Oli Mesin AHM MPX2 0.8L",
    "category": "Pelumas",
    "brand": "AHM Oil",
    "location": "Rak A-03",
    "stock": 28,
    "min_stock": 8,
    "unit_price": 48000
  },
  {
    "sku": "OIL-SHELL-AX7",
    "name": "Oli Mesin Shell Advance AX7 1L",
    "category": "Pelumas",
    "brand": "Shell",
    "location": "Rak A-04",
    "stock": 18,
    "min_stock": 6,
    "unit_price": 68000
  },
  {
    "sku": "OIL-YML-GARDAN",
    "name": "Oli Gardan Yamalube 0.1L",
    "category": "Pelumas",
    "brand": "Yamalube",
    "location": "Rak A-05",
    "stock": 40,
    "min_stock": 10,
    "unit_price": 18000
  },
  {
    "sku": "OIL-FDR-SPORT",
    "name": "Oli Mesin FDR Sport 1L",
    "category": "Pelumas",
    "brand": "FDR",
    "location": "Rak A-06",
    "stock": 12,
    "min_stock": 4,
    "unit_price": 72000
  },
  {
    "sku": "OIL-BRT-RACING",
    "name": "Oli Mesin BRT Racing 0.8L",
    "category": "Pelumas",
    "brand": "BRT",
    "location": "Rak A-07",
    "stock": 10,
    "min_stock": 3,
    "unit_price": 78000
  },
  {
    "sku": "OIL-TOP1-SYN",
    "name": "Oli Mesin TOP 1 Synthetic 1L",
    "category": "Pelumas",
    "brand": "TOP 1",
    "location": "Rak A-08",
    "stock": 8,
    "min_stock": 3,
    "unit_price": 125000
  },
  {
    "sku": "SPK-NGK-CR7",
    "name": "Busi NGK CR7HSA",
    "category": "Pengapian",
    "brand": "NGK",
    "location": "Rak B-01",
    "stock": 36,
    "min_stock": 10,
    "unit_price": 28000
  },
  {
    "sku": "SPK-NGK-CPR9",
    "name": "Busi NGK CPR9EA-9",
    "category": "Pengapian",
    "brand": "NGK",
    "location": "Rak B-02",
    "stock": 20,
    "min_stock": 6,
    "unit_price": 42000
  },
  {
    "sku": "SPK-DENSO-U22",
    "name": "Busi Denso U22EPR-U",
    "category": "Pengapian",
    "brand": "Denso",
    "location": "Rak B-03",
    "stock": 24,
    "min_stock": 8,
    "unit_price": 26000
  },
  {
    "sku": "SPK-NGK-IRIDIUM",
    "name": "Busi NGK Iridium CR8EIX",
    "category": "Pengapian",
    "brand": "NGK",
    "location": "Rak B-04",
    "stock": 2,
    "min_stock": 3,
    "unit_price": 135000
  },
  {
    "sku": "IGN-COIL-VIXION",
    "name": "Koil Pengapian Vixion",
    "category": "Pengapian",
    "brand": "Yamaha",
    "location": "Rak B-05",
    "stock": 4,
    "min_stock": 2,
    "unit_price": 285000
  },
  {
    "sku": "IGN-COIL-BEAT",
    "name": "Koil Pengapian BeAT FI",
    "category": "Pengapian",
    "brand": "AHM",
    "location": "Rak B-06",
    "stock": 5,
    "min_stock": 2,
    "unit_price": 195000
  },
  {
    "sku": "IGN-CDI-BEAT",
    "name": "CDI Unit BeAT FI",
    "category": "Pengapian",
    "brand": "AHM",
    "location": "Rak B-07",
    "stock": 3,
    "min_stock": 2,
    "unit_price": 385000
  },
  {
    "sku": "IGN-SPUL-BEAT",
    "name": "Spul Stator BeAT FI",
    "category": "Pengapian",
    "brand": "AHM",
    "location": "Rak B-08",
    "stock": 2,
    "min_stock": 2,
    "unit_price": 520000
  },
  {
    "sku": "BRK-PAD-DPN-01",
    "name": "Kampas Rem Depan Aspira",
    "category": "Rem",
    "brand": "Aspira",
    "location": "Rak C-01",
    "stock": 22,
    "min_stock": 6,
    "unit_price": 65000
  },
  {
    "sku": "BRK-PAD-BLK-01",
    "name": "Kampas Rem Belakang Aspira",
    "category": "Rem",
    "brand": "Aspira",
    "location": "Rak C-02",
    "stock": 18,
    "min_stock": 6,
    "unit_price": 58000
  },
  {
    "sku": "BRK-PAD-FDR-DPN",
    "name": "Kampas Rem Depan FDR",
    "category": "Rem",
    "brand": "FDR",
    "location": "Rak C-03",
    "stock": 10,
    "min_stock": 4,
    "unit_price": 78000
  },
  {
    "sku": "BRK-DISC-DPN-01",
    "name": "Disc Brake Depan Vario 125",
    "category": "Rem",
    "brand": "AHM",
    "location": "Rak C-04",
    "stock": 4,
    "min_stock": 2,
    "unit_price": 245000
  },
  {
    "sku": "BRK-DISC-BLK-01",
    "name": "Disc Brake Belakang NMAX",
    "category": "Rem",
    "brand": "Yamaha",
    "location": "Rak C-05",
    "stock": 2,
    "min_stock": 2,
    "unit_price": 285000
  },
  {
    "sku": "BRK-KALIPER-BEAT",
    "name": "Kaliper Rem Depan BeAT",
    "category": "Rem",
    "brand": "AHM",
    "location": "Rak C-06",
    "stock": 2,
    "min_stock": 1,
    "unit_price": 375000
  },
  {
    "sku": "BRK-MINYAK-DOT3",
    "name": "Minyak Rem DOT 3 300ml",
    "category": "Rem",
    "brand": "Aspira",
    "location": "Rak C-07",
    "stock": 14,
    "min_stock": 4,
    "unit_price": 32000
  },
  {
    "sku": "BRK-SEPATU-DPN",
    "name": "Sepatu Rem Tromol Depan",
    "category": "Rem",
    "brand": "Aspira",
    "location": "Rak C-08",
    "stock": 12,
    "min_stock": 4,
    "unit_price": 48000
  },
  {
    "sku": "BRK-SEPATU-BLK",
    "name": "Sepatu Rem Tromol Belakang",
    "category": "Rem",
    "brand": "Aspira",
    "location": "Rak C-09",
    "stock": 9,
    "min_stock": 4,
    "unit_price": 45000
  },
  {
    "sku": "BRK-SELANG-BRAID",
    "name": "Selang Rem Braided Kawahara",
    "category": "Rem",
    "brand": "Kawahara",
    "location": "Rak C-10",
    "stock": 3,
    "min_stock": 2,
    "unit_price": 210000
  },
  {
    "sku": "FLT-AIR-VARIO",
    "name": "Filter Udara Vario 125/150",
    "category": "Filter",
    "brand": "AHM",
    "location": "Rak D-01",
    "stock": 16,
    "min_stock": 5,
    "unit_price": 42000
  },
  {
    "sku": "FLT-AIR-NMAX",
    "name": "Filter Udara NMAX",
    "category": "Filter",
    "brand": "Yamaha",
    "location": "Rak D-02",
    "stock": 12,
    "min_stock": 4,
    "unit_price": 55000
  },
  {
    "sku": "FLT-OLI-BEAT",
    "name": "Filter Oli BeAT FI",
    "category": "Filter",
    "brand": "AHM",
    "location": "Rak D-03",
    "stock": 25,
    "min_stock": 8,
    "unit_price": 22000
  },
  {
    "sku": "FLT-OLI-VIXION",
    "name": "Filter Oli Vixion",
    "category": "Filter",
    "brand": "Yamaha",
    "location": "Rak D-04",
    "stock": 18,
    "min_stock": 6,
    "unit_price": 27000
  },
  {
    "sku": "FLT-BENSIN-BEAT",
    "name": "Filter Bensin BeAT FI",
    "category": "Filter",
    "brand": "AHM",
    "location": "Rak D-05",
    "stock": 20,
    "min_stock": 6,
    "unit_price": 38000
  },
  {
    "sku": "FLT-AIR-RACING",
    "name": "Filter Udara Racing Kawahara",
    "category": "Filter",
    "brand": "Kawahara",
    "location": "Rak D-06",
    "stock": 5,
    "min_stock": 2,
    "unit_price": 145000
  },
  {
    "sku": "FLT-OLI-UNIV",
    "name": "Filter Oli Universal Aspira",
    "category": "Filter",
    "brand": "Aspira",
    "location": "Rak D-07",
    "stock": 30,
    "min_stock": 10,
    "unit_price": 18000
  },
  {
    "sku": "FLT-BENSIN-NMAX",
    "name": "Filter Bensin NMAX",
    "category": "Filter",
    "brand": "Yamaha",
    "location": "Rak D-08",
    "stock": 10,
    "min_stock": 4,
    "unit_price": 62000
  },
  {
    "sku": "TIR-DPN-IRC-80",
    "name": "Ban Luar Depan IRC 80/90-14",
    "category": "Ban & Roda",
    "brand": "IRC",
    "location": "Rak E-01",
    "stock": 10,
    "min_stock": 4,
    "unit_price": 178000
  },
  {
    "sku": "TIR-BLK-IRC-90",
    "name": "Ban Luar Belakang IRC 90/90-14",
    "category": "Ban & Roda",
    "brand": "IRC",
    "location": "Rak E-02",
    "stock": 8,
    "min_stock": 4,
    "unit_price": 195000
  },
  {
    "sku": "TIR-DPN-FDR-70",
    "name": "Ban Luar Depan FDR 70/90-17",
    "category": "Ban & Roda",
    "brand": "FDR",
    "location": "Rak E-03",
    "stock": 6,
    "min_stock": 3,
    "unit_price": 210000
  },
  {
    "sku": "TIR-BLK-FDR-100",
    "name": "Ban Luar Belakang FDR 100/80-17",
    "category": "Ban & Roda",
    "brand": "FDR",
    "location": "Rak E-04",
    "stock": 2,
    "min_stock": 2,
    "unit_price": 285000
  },
  {
    "sku": "TIR-DLM-IRC-14",
    "name": "Ban Dalam IRC 275-14",
    "category": "Ban & Roda",
    "brand": "IRC",
    "location": "Rak E-05",
    "stock": 15,
    "min_stock": 5,
    "unit_price": 42000
  },
  {
    "sku": "TIR-DLM-FDR-17",
    "name": "Ban Dalam FDR 300-17",
    "category": "Ban & Roda",
    "brand": "FDR",
    "location": "Rak E-06",
    "stock": 12,
    "min_stock": 5,
    "unit_price": 48000
  },
  {
    "sku": "RODA-BEARING-6301",
    "name": "Bearing Roda 6301",
    "category": "Ban & Roda",
    "brand": "NSK",
    "location": "Rak E-07",
    "stock": 20,
    "min_stock": 6,
    "unit_price": 35000
  },
  {
    "sku": "RODA-BEARING-6302",
    "name": "Bearing Roda 6302",
    "category": "Ban & Roda",
    "brand": "Koyo",
    "location": "Rak E-08",
    "stock": 16,
    "min_stock": 6,
    "unit_price": 42000
  },
  {
    "sku": "RAN-RANTAI-428H",
    "name": "Rantai RK 428H-120L",
    "category": "Rantai & Gir",
    "brand": "RK",
    "location": "Rak F-01",
    "stock": 8,
    "min_stock": 3,
    "unit_price": 165000
  },
  {
    "sku": "RAN-RANTAI-DID",
    "name": "Rantai DID 428D-118L",
    "category": "Rantai & Gir",
    "brand": "DID",
    "location": "Rak F-02",
    "stock": 6,
    "min_stock": 3,
    "unit_price": 195000
  },
  {
    "sku": "RAN-GIR-DPN-14",
    "name": "Gir Depan 14T BeAT",
    "category": "Rantai & Gir",
    "brand": "AHM",
    "location": "Rak F-03",
    "stock": 10,
    "min_stock": 4,
    "unit_price": 55000
  },
  {
    "sku": "RAN-GIR-BLK-35",
    "name": "Gir Belakang 35T BeAT",
    "category": "Rantai & Gir",
    "brand": "AHM",
    "location": "Rak F-04",
    "stock": 8,
    "min_stock": 4,
    "unit_price": 95000
  },
  {
    "sku": "RAN-GIR-SET-BEAT",
    "name": "Gir Set BeAT/Scoopy",
    "category": "Rantai & Gir",
    "brand": "AHM",
    "location": "Rak F-05",
    "stock": 5,
    "min_stock": 2,
    "unit_price": 165000
  },
  {
    "sku": "RAN-KARET-GIR",
    "name": "Karet Gir BeAT",
    "category": "Rantai & Gir",
    "brand": "AHM",
    "location": "Rak F-06",
    "stock": 12,
    "min_stock": 4,
    "unit_price": 38000
  },
  {
    "sku": "RAN-GIR-SET-VIXION",
    "name": "Gir Set Vixion",
    "category": "Rantai & Gir",
    "brand": "Yamaha",
    "location": "Rak F-07",
    "stock": 2,
    "min_stock": 2,
    "unit_price": 245000
  },
  {
    "sku": "ELK-AKI-GS-GTZ5S",
    "name": "Aki GS Astra GTZ5S",
    "category": "Kelistrikan",
    "brand": "GS Astra",
    "location": "Rak G-01",
    "stock": 6,
    "min_stock": 3,
    "unit_price": 235000
  },
  {
    "sku": "ELK-AKI-YUASA-YTZ6V",
    "name": "Aki Yuasa YTZ6V",
    "category": "Kelistrikan",
    "brand": "Yuasa",
    "location": "Rak G-02",
    "stock": 2,
    "min_stock": 2,
    "unit_price": 285000
  },
  {
    "sku": "ELK-LAMPU-LED-H4",
    "name": "Lampu LED H4 6000K",
    "category": "Kelistrikan",
    "brand": "Bosch",
    "location": "Rak G-03",
    "stock": 14,
    "min_stock": 4,
    "unit_price": 68000
  },
  {
    "sku": "ELK-LAMPU-SEIN-DPN",
    "name": "Lampu Sein Depan BeAT",
    "category": "Kelistrikan",
    "brand": "AHM",
    "location": "Rak G-04",
    "stock": 10,
    "min_stock": 4,
    "unit_price": 32000
  },
  {
    "sku": "ELK-KLAKSON-12V",
    "name": "Klakson Keong 12V",
    "category": "Kelistrikan",
    "brand": "Denso",
    "location": "Rak G-05",
    "stock": 8,
    "min_stock": 3,
    "unit_price": 85000
  },
  {
    "sku": "ELK-REGULATOR-BEAT",
    "name": "Regulator BeAT FI",
    "category": "Kelistrikan",
    "brand": "AHM",
    "location": "Rak G-06",
    "stock": 1,
    "min_stock": 2,
    "unit_price": 175000
  },
  {
    "sku": "ELK-SEKERING-10A",
    "name": "Sekring 10A (isi 10 pcs)",
    "category": "Kelistrikan",
    "brand": "Aspira",
    "location": "Rak G-07",
    "stock": 25,
    "min_stock": 8,
    "unit_price": 15000
  },
  {
    "sku": "MSM-PISTON-BEAT",
    "name": "Piston Kit BeAT 50mm",
    "category": "Mesin",
    "brand": "AHM",
    "location": "Rak H-01",
    "stock": 1,
    "min_stock": 2,
    "unit_price": 165000
  },
  {
    "sku": "MSM-RING-SEHER-BEAT",
    "name": "Ring Seher BeAT STD",
    "category": "Mesin",
    "brand": "AHM",
    "location": "Rak H-02",
    "stock": 8,
    "min_stock": 3,
    "unit_price": 62000
  },
  {
    "sku": "MSM-PAKU-SIL-BEAT",
    "name": "Paku Sil Silinder BeAT",
    "category": "Mesin",
    "brand": "AHM",
    "location": "Rak H-03",
    "stock": 10,
    "min_stock": 4,
    "unit_price": 28000
  },
  {
    "sku": "MSM-GASKET-BEAT",
    "name": "Gasket Full Set BeAT",
    "category": "Mesin",
    "brand": "AHM",
    "location": "Rak H-04",
    "stock": 6,
    "min_stock": 3,
    "unit_price": 95000
  },
  {
    "sku": "MSM-KLEP-BEAT",
    "name": "Klep In/Ex BeAT",
    "category": "Mesin",
    "brand": "AHM",
    "location": "Rak H-05",
    "stock": 6,
    "min_stock": 2,
    "unit_price": 78000
  },
  {
    "sku": "MSM-KAMPAS-KOPLING",
    "name": "Kampas Kopling Vixion",
    "category": "Mesin",
    "brand": "Yamaha",
    "location": "Rak H-06",
    "stock": 5,
    "min_stock": 2,
    "unit_price": 125000
  },
  {
    "sku": "MSM-BEARING-6305",
    "name": "Bearing Kruk As 6305",
    "category": "Mesin",
    "brand": "NTN",
    "location": "Rak H-07",
    "stock": 6,
    "min_stock": 3,
    "unit_price": 68000
  },
  {
    "sku": "MSM-CAM-CHAIN-92L",
    "name": "Rantai Cam Chain 92L",
    "category": "Mesin",
    "brand": "DID",
    "location": "Rak H-08",
    "stock": 5,
    "min_stock": 2,
    "unit_price": 88000
  },
  {
    "sku": "KAK-SHOCK-DPN-BEAT",
    "name": "Shock Depan BeAT (sepasang)",
    "category": "Suspensi & Kaki-kaki",
    "brand": "AHM",
    "location": "Rak I-01",
    "stock": 3,
    "min_stock": 2,
    "unit_price": 285000
  },
  {
    "sku": "KAK-SHOCK-BLK-VARIO",
    "name": "Shock Belakang Vario",
    "category": "Suspensi & Kaki-kaki",
    "brand": "AHM",
    "location": "Rak I-02",
    "stock": 1,
    "min_stock": 2,
    "unit_price": 325000
  },
  {
    "sku": "KAK-SEAL-SHOCK-27",
    "name": "Seal Shock Depan 27x37",
    "category": "Suspensi & Kaki-kaki",
    "brand": "Aspira",
    "location": "Rak I-03",
    "stock": 12,
    "min_stock": 4,
    "unit_price": 35000
  },
  {
    "sku": "KAK-BUSHING-ARM",
    "name": "Bushing Arm Vario",
    "category": "Suspensi & Kaki-kaki",
    "brand": "AHM",
    "location": "Rak I-04",
    "stock": 10,
    "min_stock": 4,
    "unit_price": 38000
  },
  {
    "sku": "KAK-BALLRACE-KOMPOR",
    "name": "Ball Race Kompor",
    "category": "Suspensi & Kaki-kaki",
    "brand": "NSK",
    "location": "Rak I-05",
    "stock": 7,
    "min_stock": 3,
    "unit_price": 85000
  },
  {
    "sku": "KAK-BAN-DALAM-SHOCK",
    "name": "Karet Shock Belakang",
    "category": "Suspensi & Kaki-kaki",
    "brand": "Aspira",
    "location": "Rak I-06",
    "stock": 9,
    "min_stock": 3,
    "unit_price": 42000
  },
  {
    "sku": "BDY-SPAKBOR-DPN-BEAT",
    "name": "Spakbor Depan BeAT",
    "category": "Body & Aksesoris",
    "brand": "AHM",
    "location": "Rak J-01",
    "stock": 3,
    "min_stock": 2,
    "unit_price": 145000
  },
  {
    "sku": "BDY-VISOR-NMAX",
    "name": "Visor NMAX",
    "category": "Body & Aksesoris",
    "brand": "Yamaha",
    "location": "Rak J-02",
    "stock": 1,
    "min_stock": 1,
    "unit_price": 185000
  },
  {
    "sku": "BDY-GRIP-STANG",
    "name": "Grip Stang Racing Kawahara",
    "category": "Body & Aksesoris",
    "brand": "Kawahara",
    "location": "Rak J-03",
    "stock": 10,
    "min_stock": 4,
    "unit_price": 55000
  },
  {
    "sku": "BDY-SPION-BEAT",
    "name": "Spion BeAT (sepasang)",
    "category": "Body & Aksesoris",
    "brand": "AHM",
    "location": "Rak J-04",
    "stock": 8,
    "min_stock": 3,
    "unit_price": 65000
  },
  {
    "sku": "BDY-KARET-FOOTSTEP",
    "name": "Karet Footstep BeAT",
    "category": "Body & Aksesoris",
    "brand": "AHM",
    "location": "Rak J-05",
    "stock": 14,
    "min_stock": 5,
    "unit_price": 28000
  },
  {
    "sku": "CAI-RADIATOR-1L",
    "name": "Cairan Radiator 1L",
    "category": "Cairan",
    "brand": "Aspira",
    "location": "Rak K-01",
    "stock": 10,
    "min_stock": 4,
    "unit_price": 38000
  },
  {
    "sku": "CAI-LEM-GASKET",
    "name": "Lem Gasket 85g",
    "category": "Cairan",
    "brand": "Aspira",
    "location": "Rak K-02",
    "stock": 12,
    "min_stock": 4,
    "unit_price": 24000
  },
  {
    "sku": "CAI-CARB-CLEANER",
    "name": "Carb Cleaner 500ml",
    "category": "Cairan",
    "brand": "Aspira",
    "location": "Rak K-03",
    "stock": 8,
    "min_stock": 3,
    "unit_price": 42000
  }
]
```


## File: `internal/handlers/handlers.go`

```go
package handlers

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"motorcycle-garage-inventory/internal/config"
	"motorcycle-garage-inventory/internal/models"
)

type Handler struct {
	DB     *gorm.DB
	Config config.Config
}

func New(db *gorm.DB, cfg config.Config) *Handler { return &Handler{DB: db, Config: cfg} }
func fail(c *gin.Context, code int, msg string)   { c.JSON(code, gin.H{"error": msg}) }
func currentUserID(c *gin.Context) uint           { v, _ := c.Get("userID"); id, _ := v.(uint); return id }

// parseDate menerima "" (pakai fallback), YYYY-MM-DD, YYYY-MM-DDTHH:mm, atau RFC3339.
func parseDate(raw string, fallback time.Time) (time.Time, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return fallback, nil
	}
	layouts := []string{time.RFC3339, "2006-01-02T15:04", "2006-01-02 15:04:05", "2006-01-02"}
	for _, layout := range layouts {
		if t, err := time.ParseInLocation(layout, raw, time.Local); err == nil {
			return t, nil
		}
	}
	return fallback, errors.New("format tanggal tidak valid")
}

type authInput struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
	Role     string `json:"role"`
}

func (h *Handler) Register(c *gin.Context) {
	var in authInput
	if c.ShouldBindJSON(&in) != nil {
		fail(c, 400, "invalid request body")
		return
	}
	in.Name = strings.TrimSpace(in.Name)
	in.Email = strings.ToLower(strings.TrimSpace(in.Email))
	if in.Name == "" || in.Email == "" || len(in.Password) < 8 {
		fail(c, 400, "name/email required and password must be at least 8 characters")
		return
	}
	role := "mechanic" // Public registration cannot grant manager privileges.
	hash, err := bcrypt.GenerateFromPassword([]byte(in.Password), bcrypt.DefaultCost)
	if err != nil {
		fail(c, 500, "could not hash password")
		return
	}
	u := models.User{Name: in.Name, Email: in.Email, PasswordHash: string(hash), Role: role}
	if err := h.DB.Create(&u).Error; err != nil {
		fail(c, 409, "email already registered or invalid user")
		return
	}
	c.JSON(201, gin.H{"user": u})
}

func (h *Handler) Login(c *gin.Context) {
	var in authInput
	if c.ShouldBindJSON(&in) != nil {
		fail(c, 400, "invalid request body")
		return
	}
	var u models.User
	if err := h.DB.Where("email = ?", strings.ToLower(strings.TrimSpace(in.Email))).First(&u).Error; err != nil {
		fail(c, 401, "email or password is incorrect")
		return
	}
	if bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(in.Password)) != nil {
		fail(c, 401, "email or password is incorrect")
		return
	}
	hours := h.Config.JWTExpiresHours
	if hours < 1 {
		hours = 24
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{"user_id": u.ID, "role": u.Role, "exp": time.Now().Add(time.Duration(hours) * time.Hour).Unix(), "iat": time.Now().Unix()})
	signed, err := token.SignedString([]byte(h.Config.JWTSecret))
	if err != nil {
		fail(c, 500, "could not create token")
		return
	}
	c.JSON(200, gin.H{"token": signed, "user": u})
}

func (h *Handler) Logout(c *gin.Context) {
	c.JSON(200, gin.H{"message": "logged out; discard the token on the client"})
}

func (h *Handler) Me(c *gin.Context) {
	var u models.User
	if h.DB.First(&u, currentUserID(c)).Error != nil {
		fail(c, 404, "user not found")
		return
	}
	c.JSON(200, gin.H{"user": u})
}

func (h *Handler) ListParts(c *gin.Context) {
	q := h.DB.Model(&models.Part{})
	search := strings.TrimSpace(c.Query("search"))
	if search != "" {
		// LOWER(...) LIKE dipakai agar query jalan di SQLite maupun PostgreSQL.
		like := "%" + strings.ToLower(search) + "%"
		q = q.Where("LOWER(name) LIKE ? OR LOWER(sku) LIKE ? OR LOWER(category) LIKE ? OR LOWER(location) LIKE ? OR LOWER(brand) LIKE ?", like, like, like, like, like)
	}
	if c.Query("low_stock") == "true" {
		q = q.Where("stock <= min_stock")
	}
	var parts []models.Part
	if err := q.Order("name ASC").Find(&parts).Error; err != nil {
		fail(c, 500, "failed to list parts")
		return
	}
	c.JSON(200, gin.H{"data": parts})
}

type partInput struct {
	SKU       string `json:"sku" binding:"required"`
	Name      string `json:"name" binding:"required"`
	Category  string `json:"category"`
	Brand     string `json:"brand"`
	Location  string `json:"location" binding:"required"`
	Stock     int    `json:"stock"`
	MinStock  int    `json:"min_stock"`
	UnitPrice int64  `json:"unit_price"`
}

func (h *Handler) CreatePart(c *gin.Context) {
	var in partInput
	if c.ShouldBindJSON(&in) != nil || in.Stock < 0 || in.MinStock < 0 || in.UnitPrice < 0 {
		fail(c, 400, "invalid part data")
		return
	}
	if in.MinStock == 0 {
		in.MinStock = 3
	}
	p := models.Part{SKU: strings.TrimSpace(in.SKU), Name: strings.TrimSpace(in.Name), Category: in.Category, Brand: in.Brand, Location: in.Location, Stock: in.Stock, MinStock: in.MinStock, UnitPrice: in.UnitPrice}
	if err := h.DB.Create(&p).Error; err != nil {
		fail(c, 409, "SKU already exists or data is invalid")
		return
	}
	c.JSON(201, gin.H{"data": p})
}

func (h *Handler) GetPart(c *gin.Context) {
	var p models.Part
	if h.DB.First(&p, c.Param("id")).Error != nil {
		fail(c, 404, "part not found")
		return
	}
	c.JSON(200, gin.H{"data": p})
}

func (h *Handler) UpdatePart(c *gin.Context) {
	var p models.Part
	if h.DB.First(&p, c.Param("id")).Error != nil {
		fail(c, 404, "part not found")
		return
	}
	var in partInput
	if c.ShouldBindJSON(&in) != nil || in.Stock < 0 || in.MinStock < 0 || in.UnitPrice < 0 {
		fail(c, 400, "invalid part data")
		return
	}
	if in.MinStock == 0 {
		in.MinStock = 3
	}
	p.SKU = in.SKU
	p.Name = in.Name
	p.Category = in.Category
	p.Brand = in.Brand
	p.Location = in.Location
	p.Stock = in.Stock
	p.MinStock = in.MinStock
	p.UnitPrice = in.UnitPrice
	if err := h.DB.Save(&p).Error; err != nil {
		fail(c, 409, "could not update part; check SKU uniqueness")
		return
	}
	c.JSON(200, gin.H{"data": p})
}

func (h *Handler) DeletePart(c *gin.Context) {
	var used int64
	h.DB.Model(&models.Sale{}).Where("part_id = ?", c.Param("id")).Count(&used)
	if used > 0 {
		fail(c, 409, "sparepart sudah dipakai transaksi penjualan; tidak bisa dihapus")
		return
	}
	if err := h.DB.Delete(&models.Part{}, c.Param("id")).Error; err != nil {
		fail(c, 500, "could not delete part")
		return
	}
	c.JSON(200, gin.H{"message": "part deleted"})
}

type restockInput struct {
	Quantity int    `json:"quantity"`
	Note     string `json:"note"`
	MovedAt  string `json:"moved_at"`
}

// RestockPart menambah stok (barang masuk) dan mencatat riwayat pergerakan stok.
func (h *Handler) RestockPart(c *gin.Context) {
	var in restockInput
	if c.ShouldBindJSON(&in) != nil || in.Quantity < 1 {
		fail(c, 400, "quantity minimal 1")
		return
	}
	when, err := parseDate(in.MovedAt, time.Now())
	if err != nil {
		fail(c, 400, "format moved_at tidak valid (pakai YYYY-MM-DD)")
		return
	}

	var part models.Part
	var movement models.StockMovement
	err = h.DB.Transaction(func(tx *gorm.DB) error {
		res := tx.Model(&models.Part{}).Where("id = ?", c.Param("id")).
			UpdateColumn("stock", gorm.Expr("stock + ?", in.Quantity))
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}
		if err := tx.First(&part, c.Param("id")).Error; err != nil {
			return err
		}
		movement = models.StockMovement{PartID: part.ID, Type: "restock", Quantity: in.Quantity, Note: strings.TrimSpace(in.Note), CreatedBy: currentUserID(c), MovedAt: when}
		return tx.Create(&movement).Error
	})
	if err != nil {
		fail(c, 400, "sparepart tidak ditemukan atau restock gagal")
		return
	}
	var updated models.Part
	h.DB.First(&updated, part.ID)
	c.JSON(200, gin.H{"data": updated, "movement": movement})
}

func (h *Handler) ListStockMovements(c *gin.Context) {
	q := h.DB.Preload("Part").Order("moved_at DESC").Limit(200)
	if id := strings.TrimSpace(c.Query("part_id")); id != "" {
		q = q.Where("part_id = ?", id)
	}
	var rows []models.StockMovement
	if err := q.Find(&rows).Error; err != nil {
		fail(c, 500, "failed to list stock movements")
		return
	}
	c.JSON(200, gin.H{"data": rows})
}

type saleInput struct {
	PartID   uint   `json:"part_id"`
	Quantity int    `json:"quantity"`
	SoldAt   string `json:"sold_at"`
}

func (h *Handler) CreateSale(c *gin.Context) {
	var in saleInput
	if c.ShouldBindJSON(&in) != nil || in.PartID == 0 || in.Quantity < 1 {
		fail(c, 400, "part_id and positive quantity are required")
		return
	}
	soldAt, err := parseDate(in.SoldAt, time.Now())
	if err != nil {
		fail(c, 400, "format sold_at tidak valid (pakai YYYY-MM-DD atau YYYY-MM-DDTHH:mm)")
		return
	}

	var sale models.Sale
	err = h.DB.Transaction(func(tx *gorm.DB) error {
		// Update bersyarat: atomik, tidak butuh SELECT ... FOR UPDATE (portabel SQLite/PostgreSQL).
		res := tx.Model(&models.Part{}).
			Where("id = ? AND stock >= ?", in.PartID, in.Quantity).
			UpdateColumn("stock", gorm.Expr("stock - ?", in.Quantity))
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}
		var p models.Part
		if err := tx.First(&p, in.PartID).Error; err != nil {
			return err
		}
		sale = models.Sale{PartID: p.ID, Quantity: in.Quantity, UnitPrice: p.UnitPrice, Total: p.UnitPrice * int64(in.Quantity), SoldAt: soldAt, CreatedBy: currentUserID(c), Source: "manual"}
		if err := tx.Create(&sale).Error; err != nil {
			return err
		}
		return tx.Create(&models.StockMovement{PartID: p.ID, Type: "sale", Quantity: -in.Quantity, Note: "Penjualan manual", CreatedBy: currentUserID(c), MovedAt: soldAt}).Error
	})
	if err != nil {
		fail(c, 400, "sale failed: part may not exist or stock is insufficient")
		return
	}
	h.DB.Preload("Part").First(&sale, sale.ID)
	c.JSON(201, gin.H{"data": sale})
}

func (h *Handler) ListSales(c *gin.Context) {
	var rows []models.Sale
	q := h.DB.Preload("Part").Order("sold_at DESC").Limit(200)
	if err := q.Find(&rows).Error; err != nil {
		fail(c, 500, "failed to list sales")
		return
	}
	c.JSON(200, gin.H{"data": rows})
}

type servicePartInput struct {
	PartID   uint `json:"part_id"`
	Quantity int  `json:"quantity"`
}

type serviceInput struct {
	CustomerName string             `json:"customer_name"`
	Motorcycle   string             `json:"motorcycle"`
	PlateNumber  string             `json:"plate_number"`
	Complaint    string             `json:"complaint"`
	MechanicID   uint               `json:"mechanic_id"`
	ServiceDate  string             `json:"service_date"`
	Parts        []servicePartInput `json:"parts"`
}

func (h *Handler) ListServices(c *gin.Context) {
	var rows []models.ServiceJob
	if err := h.DB.Preload("Mechanic").Preload("Parts.Part").Order("created_at DESC").Find(&rows).Error; err != nil {
		fail(c, 500, "failed to list services")
		return
	}
	c.JSON(200, gin.H{"data": rows})
}

func (h *Handler) CreateService(c *gin.Context) {
	var in serviceInput
	if c.ShouldBindJSON(&in) != nil || strings.TrimSpace(in.CustomerName) == "" || strings.TrimSpace(in.Motorcycle) == "" {
		fail(c, 400, "customer_name and motorcycle are required")
		return
	}
	jobDate, err := parseDate(in.ServiceDate, time.Now())
	if err != nil {
		fail(c, 400, "format service_date tidak valid (pakai YYYY-MM-DD)")
		return
	}
	if in.MechanicID == 0 {
		in.MechanicID = currentUserID(c)
	}
	var mechanic models.User
	if h.DB.First(&mechanic, in.MechanicID).Error != nil {
		fail(c, 400, "mechanic not found")
		return
	}

	// Validasi sparepart yang dibutuhkan sebelum job dibuat.
	type jobLine struct {
		part models.Part
		qty  int
	}
	lines := make([]jobLine, 0, len(in.Parts))
	for _, item := range in.Parts {
		if item.PartID == 0 || item.Quantity < 1 {
			fail(c, 400, "part_id dan quantity minimal 1 wajib diisi")
			return
		}
		var part models.Part
		if err := h.DB.First(&part, item.PartID).Error; err != nil {
			fail(c, 400, "sparepart untuk service tidak ditemukan")
			return
		}
		if part.Stock < item.Quantity {
			fail(c, 400, fmt.Sprintf("stok %s tidak cukup (tersedia %d)", part.Name, part.Stock))
			return
		}
		lines = append(lines, jobLine{part: part, qty: item.Quantity})
	}

	job := models.ServiceJob{
		WorkOrder:    "WO-" + time.Now().Format("20060102-150405") + "-" + strconv.Itoa(int(time.Now().UnixNano()%1000)),
		CustomerName: strings.TrimSpace(in.CustomerName),
		Motorcycle:   strings.TrimSpace(in.Motorcycle),
		PlateNumber:  in.PlateNumber,
		Complaint:    in.Complaint,
		MechanicID:   in.MechanicID,
		Status:       "queued",
		CreatedAt:    jobDate,
	}
	err = h.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&job).Error; err != nil {
			return err
		}
		for _, line := range lines {
			row := models.ServiceJobPart{
				ServiceJobID: job.ID,
				PartID:       line.part.ID,
				Quantity:     line.qty,
				UnitPrice:    line.part.UnitPrice,
				Subtotal:     line.part.UnitPrice * int64(line.qty),
			}
			if err := tx.Create(&row).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		fail(c, 500, "could not create service job")
		return
	}
	h.DB.Preload("Mechanic").Preload("Parts.Part").First(&job, job.ID)
	c.JSON(201, gin.H{"data": job})
}

type statusInput struct {
	Status string `json:"status"`
}

// UpdateServiceStatus mengubah status job. Saat status menjadi completed, seluruh
// sparepart job otomatis dicatat sebagai penjualan dan stok dikurangi. Saat job
// dibuka kembali / dibatalkan, penjualan tersebut dibalik dan stok dikembalikan.
func (h *Handler) UpdateServiceStatus(c *gin.Context) {
	var in statusInput
	if c.ShouldBindJSON(&in) != nil {
		fail(c, 400, "status is required")
		return
	}
	valid := map[string]bool{"queued": true, "in_progress": true, "waiting_parts": true, "completed": true, "cancelled": true}
	if !valid[in.Status] {
		fail(c, 400, "unsupported status")
		return
	}
	var job models.ServiceJob
	if h.DB.Preload("Parts.Part").First(&job, c.Param("id")).Error != nil {
		fail(c, 404, "service job not found")
		return
	}

	now := time.Now()
	err := h.DB.Transaction(func(tx *gorm.DB) error {
		switch in.Status {
		case "completed":
			if job.CompletedAt == nil {
				for _, item := range job.Parts {
					res := tx.Model(&models.Part{}).Where("id = ? AND stock >= ?", item.PartID, item.Quantity).
						UpdateColumn("stock", gorm.Expr("stock - ?", item.Quantity))
					if res.Error != nil {
						return res.Error
					}
					if res.RowsAffected == 0 {
						name := item.Part.Name
						if name == "" {
							name = fmt.Sprintf("part #%d", item.PartID)
						}
						return fmt.Errorf("stok %s tidak cukup, restock dulu sebelum menyelesaikan service", name)
					}
					sale := models.Sale{
						PartID:       item.PartID,
						Quantity:     item.Quantity,
						UnitPrice:    item.UnitPrice,
						Total:        item.Subtotal,
						SoldAt:       now,
						CreatedBy:    currentUserID(c),
						Source:       "service",
						ServiceJobID: &job.ID,
						WorkOrder:    job.WorkOrder,
					}
					if err := tx.Create(&sale).Error; err != nil {
						return err
					}
					if err := tx.Create(&models.StockMovement{PartID: item.PartID, Type: "service", Quantity: -item.Quantity, Note: "Service " + job.WorkOrder, CreatedBy: currentUserID(c), MovedAt: now}).Error; err != nil {
						return err
					}
				}
				completed := now
				job.CompletedAt = &completed
			}
		default:
			// Job keluar dari status completed: balik penjualan + stok.
			if job.CompletedAt != nil {
				var sales []models.Sale
				if err := tx.Where("service_job_id = ?", job.ID).Find(&sales).Error; err != nil {
					return err
				}
				for _, s := range sales {
					if err := tx.Model(&models.Part{}).Where("id = ?", s.PartID).
						UpdateColumn("stock", gorm.Expr("stock + ?", s.Quantity)).Error; err != nil {
						return err
					}
					if err := tx.Create(&models.StockMovement{PartID: s.PartID, Type: "correction", Quantity: s.Quantity, Note: "Pembatalan service " + job.WorkOrder, CreatedBy: currentUserID(c), MovedAt: now}).Error; err != nil {
						return err
					}
					if err := tx.Delete(&models.Sale{}, s.ID).Error; err != nil {
						return err
					}
				}
			}
			job.CompletedAt = nil
			if in.Status == "in_progress" && job.StartedAt == nil {
				started := now
				job.StartedAt = &started
			}
		}

		updates := map[string]any{"status": in.Status, "completed_at": job.CompletedAt, "started_at": job.StartedAt}
		return tx.Model(&models.ServiceJob{}).Where("id = ?", job.ID).Updates(updates).Error
	})
	if err != nil {
		fail(c, 400, err.Error())
		return
	}
	h.DB.Preload("Mechanic").Preload("Parts.Part").First(&job, job.ID)
	c.JSON(200, gin.H{"data": job})
}

func (h *Handler) Dashboard(c *gin.Context) {
	var partCount, lowCount, activeJobs, completedToday int64
	h.DB.Model(&models.Part{}).Count(&partCount)
	h.DB.Model(&models.Part{}).Where("stock <= min_stock").Count(&lowCount)
	h.DB.Model(&models.ServiceJob{}).Where("status IN ?", []string{"queued", "in_progress", "waiting_parts"}).Count(&activeJobs)

	// Chart diagregasi di Go agar tidak bergantung pada fungsi tanggal khas PostgreSQL.
	type DaySale struct {
		Day      string `json:"day"`
		Total    int64  `json:"total"`
		Quantity int64  `json:"quantity"`
	}
	type DayCount struct {
		Day   string `json:"day"`
		Total int64  `json:"total"`
	}
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)
	start := today.AddDate(0, 0, -6)

	chart := make([]DaySale, 0, 7)
	dayIndex := make(map[string]int, 7)
	for i := 0; i < 7; i++ {
		day := start.AddDate(0, 0, i).Format("2006-01-02")
		dayIndex[day] = i
		chart = append(chart, DaySale{Day: day})
	}
	var sales []models.Sale
	h.DB.Model(&models.Sale{}).Where("sold_at >= ?", start).Find(&sales)
	for _, s := range sales {
		if i, ok := dayIndex[s.SoldAt.Format("2006-01-02")]; ok {
			chart[i].Total += s.Total
			chart[i].Quantity += int64(s.Quantity)
		}
	}

	completedChart := make([]DayCount, 0, 7)
	for i := 0; i < 7; i++ {
		completedChart = append(completedChart, DayCount{Day: start.AddDate(0, 0, i).Format("2006-01-02")})
	}
	var jobs []models.ServiceJob
	h.DB.Model(&models.ServiceJob{}).Where("status = ? AND completed_at >= ?", "completed", start).Find(&jobs)
	for _, job := range jobs {
		if job.CompletedAt == nil {
			continue
		}
		if i, ok := dayIndex[job.CompletedAt.Format("2006-01-02")]; ok {
			completedChart[i].Total++
		}
	}
	h.DB.Model(&models.ServiceJob{}).Where("status = ? AND completed_at >= ?", "completed", today).Count(&completedToday)

	var revenue int64
	h.DB.Model(&models.Sale{}).Where("sold_at >= ?", today).Select("COALESCE(SUM(total),0)").Scan(&revenue)
	var lowParts []models.Part
	h.DB.Where("stock <= min_stock").Order("stock ASC").Limit(8).Find(&lowParts)
	c.JSON(http.StatusOK, gin.H{
		"stats": gin.H{
			"parts":           partCount,
			"low_stock":       lowCount,
			"active_services": activeJobs,
			"today_revenue":   revenue,
			"today_completed": completedToday,
		},
		"sales_chart":     chart,
		"completed_chart": completedChart,
		"low_stock_parts": lowParts,
	})
}
```


## File: `internal/middleware/auth.go`

```go
package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

func Auth(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		tokenString := strings.TrimSpace(strings.TrimPrefix(header, "Bearer "))
		if tokenString == "" || tokenString == header {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing bearer token"})
			return
		}
		token, err := jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrSignatureInvalid
			}
			return []byte(secret), nil
		})
		if err != nil || !token.Valid {
			c.AbortWithStatusJSON(401, gin.H{"error": "invalid or expired token"})
			return
		}
		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			c.AbortWithStatusJSON(401, gin.H{"error": "invalid token claims"})
			return
		}
		id, ok := claims["user_id"].(float64)
		if !ok {
			c.AbortWithStatusJSON(401, gin.H{"error": "invalid token subject"})
			return
		}
		role, _ := claims["role"].(string)
		c.Set("userID", uint(id))
		c.Set("role", role)
		c.Next()
	}
}
func RequireRole(role string) gin.HandlerFunc {
	return func(c *gin.Context) {
		actual, _ := c.Get("role")
		if actual != role {
			c.AbortWithStatusJSON(403, gin.H{"error": "insufficient role"})
			return
		}
		c.Next()
	}
}
```


## File: `internal/models/models.go`

```go
package models

import "time"

type User struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	Name         string    `gorm:"size:120;not null" json:"name"`
	Email        string    `gorm:"size:180;uniqueIndex;not null" json:"email"`
	PasswordHash string    `gorm:"not null" json:"-"`
	Role         string    `gorm:"size:20;not null;default:mechanic" json:"role"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}
type Part struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	SKU       string    `gorm:"size:60;uniqueIndex;not null" json:"sku"`
	Name      string    `gorm:"size:180;not null;index" json:"name"`
	Category  string    `gorm:"size:100;index" json:"category"`
	Brand     string    `gorm:"size:100" json:"brand"`
	Location  string    `gorm:"size:120;not null;index" json:"location"`
	Stock     int       `gorm:"not null;default:0" json:"stock"`
	MinStock  int       `gorm:"not null;default:3" json:"min_stock"`
	UnitPrice int64     `gorm:"not null;default:0" json:"unit_price"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
type Sale struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	PartID       uint      `gorm:"not null;index" json:"part_id"`
	Part         Part      `gorm:"foreignKey:PartID" json:"part,omitempty"`
	Quantity     int       `gorm:"not null" json:"quantity"`
	UnitPrice    int64     `gorm:"not null" json:"unit_price"`
	Total        int64     `gorm:"not null" json:"total"`
	SoldAt       time.Time `gorm:"index;not null" json:"sold_at"`
	CreatedBy    uint      `json:"created_by"`
	Source       string    `gorm:"size:20;not null;default:manual" json:"source"` // manual | service
	ServiceJobID *uint     `gorm:"index" json:"service_job_id,omitempty"`
	WorkOrder    string    `gorm:"size:40;index" json:"work_order,omitempty"`
}
type ServiceJob struct {
	ID           uint             `gorm:"primaryKey" json:"id"`
	WorkOrder    string           `gorm:"size:40;uniqueIndex;not null" json:"work_order"`
	CustomerName string           `gorm:"size:140;not null" json:"customer_name"`
	Motorcycle   string           `gorm:"size:140;not null" json:"motorcycle"`
	PlateNumber  string           `gorm:"size:30;index" json:"plate_number"`
	Complaint    string           `gorm:"type:text" json:"complaint"`
	MechanicID   uint             `json:"mechanic_id"`
	Mechanic     User             `gorm:"foreignKey:MechanicID" json:"mechanic,omitempty"`
	Status       string           `gorm:"size:30;not null;default:queued;index" json:"status"`
	StartedAt    *time.Time       `json:"started_at"`
	CompletedAt  *time.Time       `json:"completed_at"`
	CreatedAt    time.Time        `json:"created_at"`
	UpdatedAt    time.Time        `json:"updated_at"`
	Parts        []ServiceJobPart `gorm:"foreignKey:ServiceJobID" json:"parts,omitempty"`
}
type ServiceJobPart struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	ServiceJobID uint      `gorm:"not null;index" json:"service_job_id"`
	PartID       uint      `gorm:"not null;index" json:"part_id"`
	Part         Part      `gorm:"foreignKey:PartID" json:"part,omitempty"`
	Quantity     int       `gorm:"not null" json:"quantity"`
	UnitPrice    int64     `gorm:"not null" json:"unit_price"`
	Subtotal     int64     `gorm:"not null" json:"subtotal"`
	CreatedAt    time.Time `json:"created_at"`
}
type StockMovement struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	PartID    uint      `gorm:"not null;index" json:"part_id"`
	Part      Part      `gorm:"foreignKey:PartID" json:"part,omitempty"`
	Type      string    `gorm:"size:20;not null;index" json:"type"` // restock | sale | service | correction
	Quantity  int       `gorm:"not null" json:"quantity"`           // positif masuk, negatif keluar
	Note      string    `gorm:"size:255" json:"note"`
	CreatedBy uint      `json:"created_by"`
	MovedAt   time.Time `gorm:"index;not null" json:"moved_at"`
}
```


## File: `frontend/index.html`

```html
<!doctype html><html lang="id"><head><meta charset="UTF-8"/><meta name="viewport" content="width=device-width, initial-scale=1.0"/><meta name="theme-color" content="#171e26"/><meta name="description" content="GarageFlow — inventory, penjualan, dan service board bengkel motor."/><title>GarageFlow — Workshop OS</title></head><body><div id="root"></div><script type="module" src="/src/main.jsx"></script></body></html>
```


## File: `frontend/package.json`

```json
{
  "name": "garage-inventory-frontend",
  "private": true,
  "version": "1.0.0",
  "type": "module",
  "scripts": {"dev":"vite --host 0.0.0.0","build":"vite build","preview":"vite preview --host 0.0.0.0"},
  "dependencies": {"axios":"^1.7.9","lucide-react":"^0.468.0","react":"^18.3.1","react-dom":"^18.3.1","recharts":"^2.15.0","react-router-dom":"^6.28.1"},
  "devDependencies": {"@vitejs/plugin-react":"^4.3.4","vite":"^6.0.5"}
}
```


## File: `frontend/package-lock.json`

```json
{
  "name": "garage-inventory-frontend",
  "version": "1.0.0",
  "lockfileVersion": 3,
  "requires": true,
  "packages": {
    "": {
      "name": "garage-inventory-frontend",
      "version": "1.0.0",
      "dependencies": {
        "axios": "^1.7.9",
        "lucide-react": "^0.468.0",
        "react": "^18.3.1",
        "react-dom": "^18.3.1",
        "react-router-dom": "^6.28.1",
        "recharts": "^2.15.0"
      },
      "devDependencies": {
        "@vitejs/plugin-react": "^4.3.4",
        "vite": "^6.0.5"
      }
    },
    "node_modules/@babel/code-frame": {
      "version": "7.29.7",
      "resolved": "https://registry.npmjs.org/@babel/code-frame/-/code-frame-7.29.7.tgz",
      "integrity": "sha512-Aup7aUOfpbAUg2ROOJN6Iw5f9DMBlzu0mIkm/malLQFN/YQgO48wCj0Kxa3sEHJvPVFg7siR+qRInwXd2qhQKw==",
      "dev": true,
      "license": "MIT",
      "dependencies": {
        "@babel/helper-validator-identifier": "^7.29.7",
        "js-tokens": "^4.0.0",
        "picocolors": "^1.1.1"
      },
      "engines": {
        "node": ">=6.9.0"
      }
    },
    "node_modules/@babel/compat-data": {
      "version": "7.29.7",
      "resolved": "https://registry.npmjs.org/@babel/compat-data/-/compat-data-7.29.7.tgz",
      "integrity": "sha512-locTkQyKvwIEgBzVrn8693ebc97F2U8ZHjbXwDXJ5Fn2TCpNwTlKcaKLkdHop5c/icOFE7qt7Q9JC5hnKNa6Gg==",
      "dev": true,
      "license": "MIT",
      "engines": {
        "node": ">=6.9.0"
      }
    },
    "node_modules/@babel/core": {
      "version": "7.29.7",
      "resolved": "https://registry.npmjs.org/@babel/core/-/core-7.29.7.tgz",
      "integrity": "sha512-RgHBCvtjbOK2gXSNBNIkNoEc9qoVEtau3hj8gEqKQuL3HZAibKarWFEI3Lfm6EYKkLalOh8eSrj9b+ch9H/VBA==",
      "dev": true,
      "license": "MIT",
      "dependencies": {
        "@babel/code-frame": "^7.29.7",
        "@babel/generator": "^7.29.7",
        "@babel/helper-compilation-targets": "^7.29.7",
        "@babel/helper-module-transforms": "^7.29.7",
        "@babel/helpers": "^7.29.7",
        "@babel/parser": "^7.29.7",
        "@babel/template": "^7.29.7",
        "@babel/traverse": "^7.29.7",
        "@babel/types": "^7.29.7",
        "@jridgewell/remapping": "^2.3.5",
        "convert-source-map": "^2.0.0",
        "debug": "^4.1.0",
        "gensync": "^1.0.0-beta.2",
        "json5": "^2.2.3",
        "semver": "^6.3.1"
      },
      "engines": {
        "node": ">=6.9.0"
      },
      "funding": {
        "type": "opencollective",
        "url": "https://opencollective.com/babel"
      }
    },
    "node_modules/@babel/generator": {
      "version": "7.29.8",
      "resolved": "https://registry.npmjs.org/@babel/generator/-/generator-7.29.8.tgz",
      "integrity": "sha512-gZbepsdh3WDtgZKWL+vTPh71LSBrm/Y4/QDZBVCcYfmeTEEuoOYwlSy+G1StfJg+/Zy550u/3TATbm7qDbbMtg==",
      "dev": true,
      "license": "MIT",
      "dependencies": {
        "@babel/parser": "^7.29.8",
        "@babel/types": "^7.29.8",
        "@jridgewell/gen-mapping": "^0.3.12",
        "@jridgewell/trace-mapping": "^0.3.28",
        "jsesc": "^3.0.2"
      },
      "engines": {
        "node": ">=6.9.0"
      }
    },
    "node_modules/@babel/helper-compilation-targets": {
      "version": "7.29.7",
      "resolved": "https://registry.npmjs.org/@babel/helper-compilation-targets/-/helper-compilation-targets-7.29.7.tgz",
      "integrity": "sha512-wem6WaBj4NaVYVdNhLPPVacES6ZJ+KBBfSkTMD3YZxbP3rm3Di85tJU5ljaUNhaOynt+Aj0xruhYuzQBt8n71g==",
      "dev": true,
      "license": "MIT",
      "dependencies": {
        "@babel/compat-data": "^7.29.7",
        "@babel/helper-validator-option": "^7.29.7",
        "browserslist": "^4.24.0",
        "lru-cache": "^5.1.1",
        "semver": "^6.3.1"
      },
      "engines": {
        "node": ">=6.9.0"
      }
    },
    "node_modules/@babel/helper-globals": {
      "version": "7.29.7",
      "resolved": "https://registry.npmjs.org/@babel/helper-globals/-/helper-globals-7.29.7.tgz",
      "integrity": "sha512-3nQVUAtvkKH9zahfWgw96Jc/uFOmjACE1kQz82E2lqWmHBgjzbNlsC22nuQTfahmWeQtTq5nQ/4Nnd2A1wj4zA==",
      "dev": true,
      "license": "MIT",
      "engines": {
        "node": ">=6.9.0"
      }
    },
    "node_modules/@babel/helper-module-imports": {
      "version": "7.29.7",
      "resolved": "https://registry.npmjs.org/@babel/helper-module-imports/-/helper-module-imports-7.29.7.tgz",
      "integrity": "sha512-ejHwrQQYcm9xnTivShn2IDOlIzInN34AXskvq9QicvCtEzq1Vzclu/tKF8Jq1Cg8JG2GL6/EmjgsCT7lXepE3g==",
      "dev": true,
      "license": "MIT",
      "dependencies": {
        "@babel/traverse": "^7.29.7",
        "@babel/types": "^7.29.7"
      },
      "engines": {
        "node": ">=6.9.0"
      }
    },
    "node_modules/@babel/helper-module-transforms": {
      "version": "7.29.7",
      "resolved": "https://registry.npmjs.org/@babel/helper-module-transforms/-/helper-module-transforms-7.29.7.tgz",
      "integrity": "sha512-UPUVSyXbOh627KiCIGQSgwWzGeBKLkaJ9PJEdrngIwMSzxLR4jS4+f1f1jb7VzBbg8nFLaYotvVPFCTqdrmTAg==",
      "dev": true,
      "license": "MIT",
      "dependencies": {
        "@babel/helper-module-imports": "^7.29.7",
        "@babel/helper-validator-identifier": "^7.29.7",
        "@babel/traverse": "^7.29.7"
      },
      "engines": {
        "node": ">=6.9.0"
      },
      "peerDependencies": {
        "@babel/core": "^7.0.0"
      }
    },
    "node_modules/@babel/helper-plugin-utils": {
      "version": "7.29.7",
      "resolved": "https://registry.npmjs.org/@babel/helper-plugin-utils/-/helper-plugin-utils-7.29.7.tgz",
      "integrity": "sha512-G7sHYigPY17oO5SYWnfD/0MTBwVR781S/JI643e/JhUYgVgWE/61SoW3NH9KWUKyKq5LVh3npif99Wkt6j86Jw==",
      "dev": true,
      "license": "MIT",
      "engines": {
        "node": ">=6.9.0"
      }
    },
    "node_modules/@babel/helper-string-parser": {
      "version": "7.29.11",
      "resolved": "https://registry.npmjs.org/@babel/helper-string-parser/-/helper-string-parser-7.29.11.tgz",
      "integrity": "sha512-snZbO7MEDOof+hvNa97EjEY7qD7h2kwvQr5AJC3ZhQudKW5dnej6k3kP9NQV88hJwaS8vDqRLFo9rivLEhle2g==",
      "dev": true,
      "license": "MIT",
      "engines": {
        "node": ">=6.9.0"
      }
    },
    "node_modules/@babel/helper-validator-identifier": {
      "version": "7.29.7",
      "resolved": "https://registry.npmjs.org/@babel/helper-validator-identifier/-/helper-validator-identifier-7.29.7.tgz",
      "integrity": "sha512-qehxGkRj55h/ff8EMaJ+cYhyaKlHIxqYDn682wQD7RNp9UujOQsHog2uS0r2vzr4pW+sXf90NeeayjcNaX3fFg==",
      "dev": true,
      "license": "MIT",
      "engines": {
        "node": ">=6.9.0"
      }
    },
    "node_modules/@babel/helper-validator-option": {
      "version": "7.29.7",
      "resolved": "https://registry.npmjs.org/@babel/helper-validator-option/-/helper-validator-option-7.29.7.tgz",
      "integrity": "sha512-N9ZErrD+yW5geCDtBqnOoxmR8+tNKiGuxKlDpuJxfsqpa2dFcexaziGAE/qoHLiDDreVNMupxGmSoNlyvsA3gw==",
      "dev": true,
      "license": "MIT",
      "engines": {
        "node": ">=6.9.0"
      }
    },
    "node_modules/@babel/helpers": {
      "version": "7.29.10",
      "resolved": "https://registry.npmjs.org/@babel/helpers/-/helpers-7.29.10.tgz",
      "integrity": "sha512-EMCx3AGZ3WfAl6ht7AyQoPm2Lh+BGYEMdc4Q6Hf7uSCDOZFuLq44qNwL+iaim8VlyYhRAo4X43+weadK8ZB4Yg==",
      "dev": true,
      "license": "MIT",
      "dependencies": {
        "@babel/template": "^7.29.7",
        "@babel/types": "^7.29.8"
      },
      "engines": {
        "node": ">=6.9.0"
      }
    },
    "node_modules/@babel/parser": {
      "version": "7.29.11",
      "resolved": "https://registry.npmjs.org/@babel/parser/-/parser-7.29.11.tgz",
      "integrity": "sha512-OZ6+oaK7GMyg3IfCpj8JOmn8n+w2rJbDT0LLfVDTLb1ZFeKDh3AiNELgCMUw175IJHSkLl74hG0M0VLsvx+gag==",
      "dev": true,
      "license": "MIT",
      "dependencies": {
        "@babel/types": "^7.29.11"
      },
      "bin": {
        "parser": "bin/babel-parser.js"
      },
      "engines": {
        "node": ">=6.0.0"
      }
    },
    "node_modules/@babel/plugin-transform-react-jsx-self": {
      "version": "7.29.7",
      "resolved": "https://registry.npmjs.org/@babel/plugin-transform-react-jsx-self/-/plugin-transform-react-jsx-self-7.29.7.tgz",
      "integrity": "sha512-TL0hMc9xzy86VD31nUiwzd5otRAcyEPcsegCxolO0PvcXuH1v0kECe/UIznYFihpkvU5wg/jk4v0TTEFfm53fw==",
      "dev": true,
      "license": "MIT",
      "dependencies": {
        "@babel/helper-plugin-utils": "^7.29.7"
      },
      "engines": {
        "node": ">=6.9.0"
      },
      "peerDependencies": {
        "@babel/core": "^7.0.0-0"
      }
    },
    "node_modules/@babel/plugin-transform-react-jsx-source": {
      "version": "7.29.7",
      "resolved": "https://registry.npmjs.org/@babel/plugin-transform-react-jsx-source/-/plugin-transform-react-jsx-source-7.29.7.tgz",
      "integrity": "sha512-06IyK09H3wi4cGbhDBwp5gUGo0IKtnYa8tyTiephirPCK6fbobVGiXMMI5zLQ4aKEYP3wZ3ArU44o+8KMrSG/Q==",
      "dev": true,
      "license": "MIT",
      "dependencies": {
        "@babel/helper-plugin-utils": "^7.29.7"
      },
      "engines": {
        "node": ">=6.9.0"
      },
      "peerDependencies": {
        "@babel/core": "^7.0.0-0"
      }
    },
    "node_modules/@babel/runtime": {
      "version": "7.29.10",
      "resolved": "https://registry.npmjs.org/@babel/runtime/-/runtime-7.29.10.tgz",
      "integrity": "sha512-NpugTzbKgGrNe45Ui32cN8vCFFMxVtP/9rtfwv0nP5FRDAjvYBxJcLVpYA9mr1jxP6cnIT7x9LUzuDTbWYYNRA==",
      "license": "MIT",
      "engines": {
        "node": ">=6.9.0"
      }
    },
    "node_modules/@babel/template": {
      "version": "7.29.7",
      "resolved": "https://registry.npmjs.org/@babel/template/-/template-7.29.7.tgz",
      "integrity": "sha512-puq+Gf35oI24FeN11LkoUQFqv9uwNeWpxXZi/Ji3rRIoKAzKnxRaZ+Gkj0vKS9ZCiTESfng1N9LyOyXvo+m+Gg==",
      "dev": true,
      "license": "MIT",
      "dependencies": {
        "@babel/code-frame": "^7.29.7",
        "@babel/parser": "^7.29.7",
        "@babel/types": "^7.29.7"
      },
      "engines": {
        "node": ">=6.9.0"
      }
    },
    "node_modules/@babel/traverse": {
      "version": "7.29.10",
      "resolved": "https://registry.npmjs.org/@babel/traverse/-/traverse-7.29.10.tgz",
      "integrity": "sha512-B3lYAs2nkKskbgQ1nfy8Ma9ao9Q1rntTAQHrzzyDuj+TSO+X3le6PfWwqDvodsb+OumE10eMFfuMCTQSQ5iuIQ==",
      "dev": true,
      "license": "MIT",
      "dependencies": {
        "@babel/code-frame": "^7.29.7",
        "@babel/generator": "^7.29.8",
        "@babel/helper-globals": "^7.29.7",
        "@babel/parser": "^7.29.9",
        "@babel/template": "^7.29.7",
        "@babel/types": "^7.29.8",
        "debug": "^4.3.1"
      },
      "engines": {
        "node": ">=6.9.0"
      }
    },
    "node_modules/@babel/types": {
      "version": "7.29.11",
      "resolved": "https://registry.npmjs.org/@babel/types/-/types-7.29.11.tgz",
      "integrity": "sha512-WCsemW40Cpc6gouC1SrY6PBaZ+oLRlEVY9XBvZBI7il/HyUrb2nJ505wE1MHEFyDZNnrWDEuy4jL50WVHfTTSQ==",
      "dev": true,
      "license": "MIT",
      "dependencies": {
        "@babel/helper-string-parser": "^7.29.11",
        "@babel/helper-validator-identifier": "^7.29.7"
      },
      "engines": {
        "node": ">=6.9.0"
      }
    },
    "node_modules/@esbuild/aix-ppc64": {
      "version": "0.25.12",
      "resolved": "https://registry.npmjs.org/@esbuild/aix-ppc64/-/aix-ppc64-0.25.12.tgz",
      "integrity": "sha512-Hhmwd6CInZ3dwpuGTF8fJG6yoWmsToE+vYgD4nytZVxcu1ulHpUQRAB1UJ8+N1Am3Mz4+xOByoQoSZf4D+CpkA==",
      "cpu": [
        "ppc64"
      ],
      "dev": true,
      "license": "MIT",
      "optional": true,
      "os": [
        "aix"
      ],
      "engines": {
        "node": ">=18"
      }
    },
    "node_modules/@esbuild/android-arm": {
      "version": "0.25.12",
      "resolved": "https://registry.npmjs.org/@esbuild/android-arm/-/android-arm-0.25.12.tgz",
      "integrity": "sha512-VJ+sKvNA/GE7Ccacc9Cha7bpS8nyzVv0jdVgwNDaR4gDMC/2TTRc33Ip8qrNYUcpkOHUT5OZ0bUcNNVZQ9RLlg==",
      "cpu": [
        "arm"
      ],
      "dev": true,
      "license": "MIT",
      "optional": true,
      "os": [
        "android"
      ],
      "engines": {
        "node": ">=18"
      }
    },
    "node_modules/@esbuild/android-arm64": {
      "version": "0.25.12",
      "resolved": "https://registry.npmjs.org/@esbuild/android-arm64/-/android-arm64-0.25.12.tgz",
      "integrity": "sha512-6AAmLG7zwD1Z159jCKPvAxZd4y/VTO0VkprYy+3N2FtJ8+BQWFXU+OxARIwA46c5tdD9SsKGZ/1ocqBS/gAKHg==",
      "cpu": [
        "arm64"
      ],
      "dev": true,
      "license": "MIT",
      "optional": true,
      "os": [
        "android"
      ],
      "engines": {
        "node": ">=18"
      }
    },
    "node_modules/@esbuild/android-x64": {
      "version": "0.25.12",
      "resolved": "https://registry.npmjs.org/@esbuild/android-x64/-/android-x64-0.25.12.tgz",
      "integrity": "sha512-5jbb+2hhDHx5phYR2By8GTWEzn6I9UqR11Kwf22iKbNpYrsmRB18aX/9ivc5cabcUiAT/wM+YIZ6SG9QO6a8kg==",
      "cpu": [
        "x64"
      ],
      "dev": true,
      "license": "MIT",
      "optional": true,
      "os": [
        "android"
      ],
      "engines": {
        "node": ">=18"
      }
    },
    "node_modules/@esbuild/darwin-arm64": {
      "version": "0.25.12",
      "resolved": "https://registry.npmjs.org/@esbuild/darwin-arm64/-/darwin-arm64-0.25.12.tgz",
      "integrity": "sha512-N3zl+lxHCifgIlcMUP5016ESkeQjLj/959RxxNYIthIg+CQHInujFuXeWbWMgnTo4cp5XVHqFPmpyu9J65C1Yg==",
      "cpu": [
        "arm64"
      ],
      "dev": true,
      "license": "MIT",
      "optional": true,
      "os": [
        "darwin"
      ],
      "engines": {
        "node": ">=18"
      }
    },
    "node_modules/@esbuild/darwin-x64": {
      "version": "0.25.12",
      "resolved": "https://registry.npmjs.org/@esbuild/darwin-x64/-/darwin-x64-0.25.12.tgz",
      "integrity": "sha512-HQ9ka4Kx21qHXwtlTUVbKJOAnmG1ipXhdWTmNXiPzPfWKpXqASVcWdnf2bnL73wgjNrFXAa3yYvBSd9pzfEIpA==",
      "cpu": [
        "x64"
      ],
      "dev": true,
      "license": "MIT",
      "optional": true,
      "os": [
        "darwin"
      ],
      "engines": {
        "node": ">=18"
      }
    },
    "node_modules/@esbuild/freebsd-arm64": {
      "version": "0.25.12",
      "resolved": "https://registry.npmjs.org/@esbuild/freebsd-arm64/-/freebsd-arm64-0.25.12.tgz",
      "integrity": "sha512-gA0Bx759+7Jve03K1S0vkOu5Lg/85dou3EseOGUes8flVOGxbhDDh/iZaoek11Y8mtyKPGF3vP8XhnkDEAmzeg==",
      "cpu": [
        "arm64"
      ],
      "dev": true,
      "license": "MIT",
      "optional": true,
      "os": [
        "freebsd"
      ],
      "engines": {
        "node": ">=18"
      }
    },
    "node_modules/@esbuild/freebsd-x64": {
      "version": "0.25.12",
      "resolved": "https://registry.npmjs.org/@esbuild/freebsd-x64/-/freebsd-x64-0.25.12.tgz",
      "integrity": "sha512-TGbO26Yw2xsHzxtbVFGEXBFH0FRAP7gtcPE7P5yP7wGy7cXK2oO7RyOhL5NLiqTlBh47XhmIUXuGciXEqYFfBQ==",
      "cpu": [
        "x64"
      ],
      "dev": true,
      "license": "MIT",
      "optional": true,
      "os": [
        "freebsd"
      ],
      "engines": {
        "node": ">=18"
      }
    },
    "node_modules/@esbuild/linux-arm": {
      "version": "0.25.12",
      "resolved": "https://registry.npmjs.org/@esbuild/linux-arm/-/linux-arm-0.25.12.tgz",
      "integrity": "sha512-lPDGyC1JPDou8kGcywY0YILzWlhhnRjdof3UlcoqYmS9El818LLfJJc3PXXgZHrHCAKs/Z2SeZtDJr5MrkxtOw==",
      "cpu": [
        "arm"
      ],
      "dev": true,
      "license": "MIT",
      "optional": true,
      "os": [
        "linux"
      ],
      "engines": {
        "node": ">=18"
      }
    },
    "node_modules/@esbuild/linux-arm64": {
      "version": "0.25.12",
      "resolved": "https://registry.npmjs.org/@esbuild/linux-arm64/-/linux-arm64-0.25.12.tgz",
      "integrity": "sha512-8bwX7a8FghIgrupcxb4aUmYDLp8pX06rGh5HqDT7bB+8Rdells6mHvrFHHW2JAOPZUbnjUpKTLg6ECyzvas2AQ==",
      "cpu": [
        "arm64"
      ],
      "dev": true,
      "license": "MIT",
      "optional": true,
      "os": [
        "linux"
      ],
      "engines": {
        "node": ">=18"
      }
    },
    "node_modules/@esbuild/linux-ia32": {
      "version": "0.25.12",
      "resolved": "https://registry.npmjs.org/@esbuild/linux-ia32/-/linux-ia32-0.25.12.tgz",
      "integrity": "sha512-0y9KrdVnbMM2/vG8KfU0byhUN+EFCny9+8g202gYqSSVMonbsCfLjUO+rCci7pM0WBEtz+oK/PIwHkzxkyharA==",
      "cpu": [
        "ia32"
      ],
      "dev": true,
      "license": "MIT",
      "optional": true,
      "os": [
        "linux"
      ],
      "engines": {
        "node": ">=18"
      }
    },
    "node_modules/@esbuild/linux-loong64": {
      "version": "0.25.12",
      "resolved": "https://registry.npmjs.org/@esbuild/linux-loong64/-/linux-loong64-0.25.12.tgz",
      "integrity": "sha512-h///Lr5a9rib/v1GGqXVGzjL4TMvVTv+s1DPoxQdz7l/AYv6LDSxdIwzxkrPW438oUXiDtwM10o9PmwS/6Z0Ng==",
      "cpu": [
        "loong64"
      ],
      "dev": true,
      "license": "MIT",
      "optional": true,
      "os": [
        "linux"
      ],
      "engines": {
        "node": ">=18"
      }
    },
    "node_modules/@esbuild/linux-mips64el": {
      "version": "0.25.12",
      "resolved": "https://registry.npmjs.org/@esbuild/linux-mips64el/-/linux-mips64el-0.25.12.tgz",
      "integrity": "sha512-iyRrM1Pzy9GFMDLsXn1iHUm18nhKnNMWscjmp4+hpafcZjrr2WbT//d20xaGljXDBYHqRcl8HnxbX6uaA/eGVw==",
      "cpu": [
        "mips64el"
      ],
      "dev": true,
      "license": "MIT",
      "optional": true,
      "os": [
        "linux"
      ],
      "engines": {
        "node": ">=18"
      }
    },
    "node_modules/@esbuild/linux-ppc64": {
      "version": "0.25.12",
      "resolved": "https://registry.npmjs.org/@esbuild/linux-ppc64/-/linux-ppc64-0.25.12.tgz",
      "integrity": "sha512-9meM/lRXxMi5PSUqEXRCtVjEZBGwB7P/D4yT8UG/mwIdze2aV4Vo6U5gD3+RsoHXKkHCfSxZKzmDssVlRj1QQA==",
      "cpu": [
        "ppc64"
      ],
      "dev": true,
      "license": "MIT",
      "optional": true,
      "os": [
        "linux"
      ],
      "engines": {
        "node": ">=18"
      }
    },
    "node_modules/@esbuild/linux-riscv64": {
      "version": "0.25.12",
      "resolved": "https://registry.npmjs.org/@esbuild/linux-riscv64/-/linux-riscv64-0.25.12.tgz",
      "integrity": "sha512-Zr7KR4hgKUpWAwb1f3o5ygT04MzqVrGEGXGLnj15YQDJErYu/BGg+wmFlIDOdJp0PmB0lLvxFIOXZgFRrdjR0w==",
      "cpu": [
        "riscv64"
      ],
      "dev": true,
      "license": "MIT",
      "optional": true,
      "os": [
        "linux"
      ],
      "engines": {
        "node": ">=18"
      }
    },
    "node_modules/@esbuild/linux-s390x": {
      "version": "0.25.12",
      "resolved": "https://registry.npmjs.org/@esbuild/linux-s390x/-/linux-s390x-0.25.12.tgz",
      "integrity": "sha512-MsKncOcgTNvdtiISc/jZs/Zf8d0cl/t3gYWX8J9ubBnVOwlk65UIEEvgBORTiljloIWnBzLs4qhzPkJcitIzIg==",
      "cpu": [
        "s390x"
      ],
      "dev": true,
      "license": "MIT",
      "optional": true,
      "os": [
        "linux"
      ],
      "engines": {
        "node": ">=18"
      }
    },
    "node_modules/@esbuild/linux-x64": {
      "version": "0.25.12",
      "resolved": "https://registry.npmjs.org/@esbuild/linux-x64/-/linux-x64-0.25.12.tgz",
      "integrity": "sha512-uqZMTLr/zR/ed4jIGnwSLkaHmPjOjJvnm6TVVitAa08SLS9Z0VM8wIRx7gWbJB5/J54YuIMInDquWyYvQLZkgw==",
      "cpu": [
        "x64"
      ],
      "dev": true,
      "license": "MIT",
      "optional": true,
      "os": [
        "linux"
      ],
      "engines": {
        "node": ">=18"
      }
    },
    "node_modules/@esbuild/netbsd-arm64": {
      "version": "0.25.12",
      "resolved": "https://registry.npmjs.org/@esbuild/netbsd-arm64/-/netbsd-arm64-0.25.12.tgz",
      "integrity": "sha512-xXwcTq4GhRM7J9A8Gv5boanHhRa/Q9KLVmcyXHCTaM4wKfIpWkdXiMog/KsnxzJ0A1+nD+zoecuzqPmCRyBGjg==",
      "cpu": [
        "arm64"
      ],
      "dev": true,
      "license": "MIT",
      "optional": true,
      "os": [
        "netbsd"
      ],
      "engines": {
        "node": ">=18"
      }
    },
    "node_modules/@esbuild/netbsd-x64": {
      "version": "0.25.12",
      "resolved": "https://registry.npmjs.org/@esbuild/netbsd-x64/-/netbsd-x64-0.25.12.tgz",
      "integrity": "sha512-Ld5pTlzPy3YwGec4OuHh1aCVCRvOXdH8DgRjfDy/oumVovmuSzWfnSJg+VtakB9Cm0gxNO9BzWkj6mtO1FMXkQ==",
      "cpu": [
        "x64"
      ],
      "dev": true,
      "license": "MIT",
      "optional": true,
      "os": [
        "netbsd"
      ],
      "engines": {
        "node": ">=18"
      }
    },
    "node_modules/@esbuild/openbsd-arm64": {
      "version": "0.25.12",
      "resolved": "https://registry.npmjs.org/@esbuild/openbsd-arm64/-/openbsd-arm64-0.25.12.tgz",
      "integrity": "sha512-fF96T6KsBo/pkQI950FARU9apGNTSlZGsv1jZBAlcLL1MLjLNIWPBkj5NlSz8aAzYKg+eNqknrUJ24QBybeR5A==",
      "cpu": [
        "arm64"
      ],
      "dev": true,
      "license": "MIT",
      "optional": true,
      "os": [
        "openbsd"
      ],
      "engines": {
        "node": ">=18"
      }
    },
    "node_modules/@esbuild/openbsd-x64": {
      "version": "0.25.12",
      "resolved": "https://registry.npmjs.org/@esbuild/openbsd-x64/-/openbsd-x64-0.25.12.tgz",
      "integrity": "sha512-MZyXUkZHjQxUvzK7rN8DJ3SRmrVrke8ZyRusHlP+kuwqTcfWLyqMOE3sScPPyeIXN/mDJIfGXvcMqCgYKekoQw==",
      "cpu": [
        "x64"
      ],
      "dev": true,
      "license": "MIT",
      "optional": true,
      "os": [
        "openbsd"
      ],
      "engines": {
        "node": ">=18"
      }
    },
    "node_modules/@esbuild/openharmony-arm64": {
      "version": "0.25.12",
      "resolved": "https://registry.npmjs.org/@esbuild/openharmony-arm64/-/openharmony-arm64-0.25.12.tgz",
      "integrity": "sha512-rm0YWsqUSRrjncSXGA7Zv78Nbnw4XL6/dzr20cyrQf7ZmRcsovpcRBdhD43Nuk3y7XIoW2OxMVvwuRvk9XdASg==",
      "cpu": [
        "arm64"
      ],
      "dev": true,
      "license": "MIT",
      "optional": true,
      "os": [
        "openharmony"
      ],
      "engines": {
        "node": ">=18"
      }
    },
    "node_modules/@esbuild/sunos-x64": {
      "version": "0.25.12",
      "resolved": "https://registry.npmjs.org/@esbuild/sunos-x64/-/sunos-x64-0.25.12.tgz",
      "integrity": "sha512-3wGSCDyuTHQUzt0nV7bocDy72r2lI33QL3gkDNGkod22EsYl04sMf0qLb8luNKTOmgF/eDEDP5BFNwoBKH441w==",
      "cpu": [
        "x64"
      ],
      "dev": true,
      "license": "MIT",
      "optional": true,
      "os": [
        "sunos"
      ],
      "engines": {
        "node": ">=18"
      }
    },
    "node_modules/@esbuild/win32-arm64": {
      "version": "0.25.12",
      "resolved": "https://registry.npmjs.org/@esbuild/win32-arm64/-/win32-arm64-0.25.12.tgz",
      "integrity": "sha512-rMmLrur64A7+DKlnSuwqUdRKyd3UE7oPJZmnljqEptesKM8wx9J8gx5u0+9Pq0fQQW8vqeKebwNXdfOyP+8Bsg==",
      "cpu": [
        "arm64"
      ],
      "dev": true,
      "license": "MIT",
      "optional": true,
      "os": [
        "win32"
      ],
      "engines": {
        "node": ">=18"
      }
    },
    "node_modules/@esbuild/win32-ia32": {
      "version": "0.25.12",
      "resolved": "https://registry.npmjs.org/@esbuild/win32-ia32/-/win32-ia32-0.25.12.tgz",
      "integrity": "sha512-HkqnmmBoCbCwxUKKNPBixiWDGCpQGVsrQfJoVGYLPT41XWF8lHuE5N6WhVia2n4o5QK5M4tYr21827fNhi4byQ==",
      "cpu": [
        "ia32"
      ],
      "dev": true,
      "license": "MIT",
      "optional": true,
      "os": [
        "win32"
      ],
      "engines": {
        "node": ">=18"
      }
    },
    "node_modules/@esbuild/win32-x64": {
      "version": "0.25.12",
      "resolved": "https://registry.npmjs.org/@esbuild/win32-x64/-/win32-x64-0.25.12.tgz",
      "integrity": "sha512-alJC0uCZpTFrSL0CCDjcgleBXPnCrEAhTBILpeAp7M/OFgoqtAetfBzX0xM00MUsVVPpVjlPuMbREqnZCXaTnA==",
      "cpu": [
        "x64"
      ],
      "dev": true,
      "license": "MIT",
      "optional": true,
      "os": [
        "win32"
      ],
      "engines": {
        "node": ">=18"
      }
    },
    "node_modules/@jridgewell/gen-mapping": {
      "version": "0.3.13",
      "resolved": "https://registry.npmjs.org/@jridgewell/gen-mapping/-/gen-mapping-0.3.13.tgz",
      "integrity": "sha512-2kkt/7niJ6MgEPxF0bYdQ6etZaA+fQvDcLKckhy1yIQOzaoKjBBjSj63/aLVjYE3qhRt5dvM+uUyfCg6UKCBbA==",
      "dev": true,
      "license": "MIT",
      "dependencies": {
        "@jridgewell/sourcemap-codec": "^1.5.0",
        "@jridgewell/trace-mapping": "^0.3.24"
      }
    },
    "node_modules/@jridgewell/remapping": {
      "version": "2.3.5",
      "resolved": "https://registry.npmjs.org/@jridgewell/remapping/-/remapping-2.3.5.tgz",
      "integrity": "sha512-LI9u/+laYG4Ds1TDKSJW2YPrIlcVYOwi2fUC6xB43lueCjgxV4lffOCZCtYFiH6TNOX+tQKXx97T4IKHbhyHEQ==",
      "dev": true,
      "license": "MIT",
      "dependencies": {
        "@jridgewell/gen-mapping": "^0.3.5",
        "@jridgewell/trace-mapping": "^0.3.24"
      }
    },
    "node_modules/@jridgewell/resolve-uri": {
      "version": "3.1.2",
      "resolved": "https://registry.npmjs.org/@jridgewell/resolve-uri/-/resolve-uri-3.1.2.tgz",
      "integrity": "sha512-bRISgCIjP20/tbWSPWMEi54QVPRZExkuD9lJL+UIxUKtwVJA8wW1Trb1jMs1RFXo1CBTNZ/5hpC9QvmKWdopKw==",
      "dev": true,
      "license": "MIT",
      "engines": {
        "node": ">=6.0.0"
      }
    },
    "node_modules/@jridgewell/sourcemap-codec": {
      "version": "1.6.0",
      "resolved": "https://registry.npmjs.org/@jridgewell/sourcemap-codec/-/sourcemap-codec-1.6.0.tgz",
      "integrity": "sha512-T7jf+5zgsZHwNJ4lvQ7/aezbyk0nNX+zJVWpmHA7VYsEx7a7qr5Rg5IbtJFqkgze5Y2sruq1RUY8Q837Od7iFw==",
      "dev": true,
      "license": "MIT"
    },
    "node_modules/@jridgewell/trace-mapping": {
      "version": "0.3.31",
      "resolved": "https://registry.npmjs.org/@jridgewell/trace-mapping/-/trace-mapping-0.3.31.tgz",
      "integrity": "sha512-zzNR+SdQSDJzc8joaeP8QQoCQr8NuYx2dIIytl1QeBEZHJ9uW6hebsrYgbz8hJwUQao3TWCMtmfV8Nu1twOLAw==",
      "dev": true,
      "license": "MIT",
      "dependencies": {
        "@jridgewell/resolve-uri": "^3.1.0",
        "@jridgewell/sourcemap-codec": "^1.4.14"
      }
    },
    "node_modules/@napi-rs/lzma-linux-x64-gnu": {
      "version": "1.5.1",
      "resolved": "https://registry.npmjs.org/@napi-rs/lzma-linux-x64-gnu/-/lzma-linux-x64-gnu-1.5.1.tgz",
      "integrity": "sha512-oTXEIha4SsuXdTA4Iyskj0kpdx2yVXdhd75c2v3xGrHFfVMsbhTPZU/nMPL4sWKo4pBHm3aucLaqGlF696dTyQ==",
      "cpu": [
        "x64"
      ],
      "dev": true,
      "libc": [
        "glibc"
      ],
      "license": "MIT",
      "optional": true,
      "os": [
        "linux"
      ],
      "engines": {
        "node": "^22.20 || ^24.12 || >=25"
      }
    },
    "node_modules/@remix-run/router": {
      "version": "1.23.4",
      "resolved": "https://registry.npmjs.org/@remix-run/router/-/router-1.23.4.tgz",
      "integrity": "sha512-q7j5geK7xs3UJSdm9/iytUNclBnLmYx1EnSeCFXHPeutdqgIMeFeHtUZgS3EhlKxdBEAu8OwtJCwmLrEzpSs7Q==",
      "license": "MIT",
      "engines": {
        "node": ">=14.0.0"
      }
    },
    "node_modules/@rolldown/pluginutils": {
      "version": "1.0.0-beta.27",
      "resolved": "https://registry.npmjs.org/@rolldown/pluginutils/-/pluginutils-1.0.0-beta.27.tgz",
      "integrity": "sha512-+d0F4MKMCbeVUJwG96uQ4SgAznZNSq93I3V+9NHA4OpvqG8mRCpGdKmK8l/dl02h2CCDHwW2FqilnTyDcAnqjA==",
      "dev": true,
      "license": "MIT"
    },
    "node_modules/@rollup/rollup-android-arm-eabi": {
      "version": "4.65.1",
      "resolved": "https://registry.npmjs.org/@rollup/rollup-android-arm-eabi/-/rollup-android-arm-eabi-4.65.1.tgz",
      "integrity": "sha512-5coxNSjqOI2J3NkVhlp3u+zGcyjnowQFsU9sg7MMz4UPXY4Dv63kFGLt7ConXz9CNq2i0yuOJZAF/2RMrgI2OA==",
      "cpu": [
        "arm"
      ],
      "dev": true,
      "license": "MIT",
      "optional": true,
      "os": [
        "android"
      ]
    },
    "node_modules/@rollup/rollup-android-arm64": {
      "version": "4.65.1",
      "resolved": "https://registry.npmjs.org/@rollup/rollup-android-arm64/-/rollup-android-arm64-4.65.1.tgz",
      "integrity": "sha512-K0HFf6XMB3IYs+jDZ1/RRhZaVOwxNXrl0n4MDEyGDjuK7+B6K/nDJp8DCBitO3S1sK/fzNxAf9QZAny4AIUPtg==",
      "cpu": [
        "arm64"
      ],
      "dev": true,
      "license": "MIT",
      "optional": true,
      "os": [
        "android"
      ]
    },
    "node_modules/@rollup/rollup-darwin-arm64": {
      "version": "4.65.1",
      "resolved": "https://registry.npmjs.org/@rollup/rollup-darwin-arm64/-/rollup-darwin-arm64-4.65.1.tgz",
      "integrity": "sha512-zkZrXYwNlIcSUmcwY2DpsaDHr0XjVUeQHCuSunG3eQD7/5Wn/DvI7OMUE8NRRweK87yG8ebJ1xGOzmwpMsbZQA==",
      "cpu": [
        "arm64"
      ],
      "dev": true,
      "license": "MIT",
      "optional": true,
      "os": [
        "darwin"
      ]
    },
    "node_modules/@rollup/rollup-darwin-x64": {
      "version": "4.65.1",
      "resolved": "https://registry.npmjs.org/@rollup/rollup-darwin-x64/-/rollup-darwin-x64-4.65.1.tgz",
      "integrity": "sha512-Zaed+R3cpkkA3aiNd7NO0GV5KFh/LVf8j6bGcoRdpoXkSm/mWH8M0oX2/3u/nyGcx/bHEovD9CzX+w9/gcpI4A==",
      "cpu": [
        "x64"
      ],
      "dev": true,
      "license": "MIT",
      "optional": true,
      "os": [
        "darwin"
      ]
    },
    "node_modules/@rollup/rollup-freebsd-arm64": {
      "version": "4.65.1",
      "resolved": "https://registry.npmjs.org/@rollup/rollup-freebsd-arm64/-/rollup-freebsd-arm64-4.65.1.tgz",
      "integrity": "sha512-h12p3WJWhyJ9kir0zqAPr0rKNmBzOWIjoaOMT9OULs9KZQZCeOMKd+mGSvERjd9VxJYShd3zIxtrVxkYz6BEVA==",
      "cpu": [
        "arm64"
      ],
      "dev": true,
      "license": "MIT",
      "optional": true,
      "os": [
        "freebsd"
      ]
    },
    "node_modules/@rollup/rollup-freebsd-x64": {
      "version": "4.65.1",
      "resolved": "https://registry.npmjs.org/@rollup/rollup-freebsd-x64/-/rollup-freebsd-x64-4.65.1.tgz",
      "integrity": "sha512-yRBwh/5imKY4DydDulVtrjy8Jf7h7NQJUqZbgmZ3zFsCD9QqlnWvZ7RtEF+ecO1b+IukKjXlu5Bb9fehV3tIbg==",
      "cpu": [
        "x64"
      ],
      "dev": true,
      "license": "MIT",
      "optional": true,
      "os": [
        "freebsd"
      ]
    },
    "node_modules/@rollup/rollup-linux-arm-gnueabihf": {
      "version": "4.65.1",
      "resolved": "https://registry.npmjs.org/@rollup/rollup-linux-arm-gnueabihf/-/rollup-linux-arm-gnueabihf-4.65.1.tgz",
      "integrity": "sha512-1XuPiWYYaLE1UDOqhbKVOfsqxvfmFS/lYPSk92U1hEX1qRfSYH0hNeAlbxFIgjp4d6+YQmL95EPRgdvjbZEddA==",
      "cpu": [
        "arm"
      ],
      "dev": true,
      "libc": [
        "glibc"
      ],
      "license": "MIT",
      "optional": true,
      "os": [
        "linux"
      ]
    },
    "node_modules/@rollup/rollup-linux-arm-musleabihf": {
      "version": "4.65.1",
      "resolved": "https://registry.npmjs.org/@rollup/rollup-linux-arm-musleabihf/-/rollup-linux-arm-musleabihf-4.65.1.tgz",
      "integrity": "sha512-Y3C5e55Yrf6p2r5rBcc4P9hFWVNi6oqWy/53I/AyCJJGlp3KRSpTP8eoUmVntchTbEnhgBgAgLO9flj9AvOdUw==",
      "cpu": [
        "arm"
      ],
      "dev": true,
      "libc": [
        "musl"
      ],
      "license": "MIT",
      "optional": true,
      "os": [
        "linux"
      ]
    },
    "node_modules/@rollup/rollup-linux-arm64-gnu": {
      "version": "4.65.1",
      "resolved": "https://registry.npmjs.org/@rollup/rollup-linux-arm64-gnu/-/rollup-linux-arm64-gnu-4.65.1.tgz",
      "integrity": "sha512-Itj6coSswkYPIcVLczObXmwdplHNQQxyfnK1ppBg4czDaX8RiEzLKqU7EMc64BiBuiDZEUS+M93ktM18rv5llw==",
      "cpu": [
        "arm64"
      ],
      "dev": true,
      "libc": [
        "glibc"
      ],
      "license": "MIT",
      "optional": true,
      "os": [
        "linux"
      ]
    },
    "node_modules/@rollup/rollup-linux-arm64-musl": {
      "version": "4.65.1",
      "resolved": "https://registry.npmjs.org/@rollup/rollup-linux-arm64-musl/-/rollup-linux-arm64-musl-4.65.1.tgz",
      "integrity": "sha512-KeO8jTEqdrnhvcahufbimyMoI44vedRUp1bVz2QUnRX2vDm/hxzCPt5F8fERVD5fsHyTW/vXmCElT054VOSVOA==",
      "cpu": [
        "arm64"
      ],
      "dev": true,
      "libc": [
        "musl"
      ],
      "license": "MIT",
      "optional": true,
      "os": [
        "linux"
      ]
    },
    "node_modules/@rollup/rollup-linux-loong64-gnu": {
      "version": "4.65.1",
      "resolved": "https://registry.npmjs.org/@rollup/rollup-linux-loong64-gnu/-/rollup-linux-loong64-gnu-4.65.1.tgz",
      "integrity": "sha512-dDOz5VZZ+ZGnglpmET6VJefamWXFsgaVtuhSClf2/URZBp048jprD6z1hddsIMN6n1w+oUzGZEBp9u6Mlszsfg==",
      "cpu": [
        "loong64"
      ],
      "dev": true,
      "libc": [
        "glibc"
      ],
      "license": "MIT",
      "optional": true,
      "os": [
        "linux"
      ]
    },
    "node_modules/@rollup/rollup-linux-loong64-musl": {
      "version": "4.65.1",
      "resolved": "https://registry.npmjs.org/@rollup/rollup-linux-loong64-musl/-/rollup-linux-loong64-musl-4.65.1.tgz",
      "integrity": "sha512-JbfRKz8nJCuo8gn1Dqx8lQWEHw4qhZ3uRaHphw5vrSjnSptJTKTRC/bLKHh8QQMsEnhL4uq+bdg3tkiXbJ2q/A==",
      "cpu": [
        "loong64"
      ],
      "dev": true,
      "libc": [
        "musl"
      ],
      "license": "MIT",
      "optional": true,
      "os": [
        "linux"
      ]
    },
    "node_modules/@rollup/rollup-linux-ppc64-gnu": {
      "version": "4.65.1",
      "resolved": "https://registry.npmjs.org/@rollup/rollup-linux-ppc64-gnu/-/rollup-linux-ppc64-gnu-4.65.1.tgz",
      "integrity": "sha512-3oQpe1q0emSeCAssVHMQdOz5fz3PBH5my/Fr0ZekV2UMNL4ZiRa1ciikYlGQXyDkeqseYYv1InwOY8YPqY7RIQ==",
      "cpu": [
        "ppc64"
      ],
      "dev": true,
      "libc": [
        "glibc"
      ],
      "license": "MIT",
      "optional": true,
      "os": [
        "linux"
      ]
    },
    "node_modules/@rollup/rollup-linux-ppc64-musl": {
      "version": "4.65.1",
      "resolved": "https://registry.npmjs.org/@rollup/rollup-linux-ppc64-musl/-/rollup-linux-ppc64-musl-4.65.1.tgz",
      "integrity": "sha512-nNV3HHi23o1LtpGGwH+Bav4hqnc4+klQtC96tSvCfvql8bLlNFTnTFiGl/jai2nve3eEr/vT2ocuyf+h+eMnYQ==",
      "cpu": [
        "ppc64"
      ],
      "dev": true,
      "libc": [
        "musl"
      ],
      "license": "MIT",
      "optional": true,
      "os": [
        "linux"
      ]
    },
    "node_modules/@rollup/rollup-linux-riscv64-gnu": {
      "version": "4.65.1",
      "resolved": "https://registry.npmjs.org/@rollup/rollup-linux-riscv64-gnu/-/rollup-linux-riscv64-gnu-4.65.1.tgz",
      "integrity": "sha512-b5zab0FryBxuocuDeqOsYGAg2EzjKw4mKA6LjQXCyewXjGv/BXqACBsY2cO45xM4hV2weYtdjGThP2VB2ByAYg==",
      "cpu": [
        "riscv64"
      ],
      "dev": true,
      "libc": [
        "glibc"
      ],
      "license": "MIT",
      "optional": true,
      "os": [
        "linux"
      ]
    },
    "node_modules/@rollup/rollup-linux-riscv64-musl": {
      "version": "4.65.1",
      "resolved": "https://registry.npmjs.org/@rollup/rollup-linux-riscv64-musl/-/rollup-linux-riscv64-musl-4.65.1.tgz",
      "integrity": "sha512-Rh08xZakiUxKNgI3hbgH9mivlx478lSqckS9s+Oc3/TX1VrAH3W8z/5wpJXR/v5VjqcJ/jJNVBg64PU1+iQsTQ==",
      "cpu": [
        "riscv64"
      ],
      "dev": true,
      "libc": [
        "musl"
      ],
      "license": "MIT",
      "optional": true,
      "os": [
        "linux"
      ]
    },
    "node_modules/@rollup/rollup-linux-s390x-gnu": {
      "version": "4.65.1",
      "resolved": "https://registry.npmjs.org/@rollup/rollup-linux-s390x-gnu/-/rollup-linux-s390x-gnu-4.65.1.tgz",
      "integrity": "sha512-qDgtS6juN3/k97y4kcD6XLFpXEo4f6CnG8lTOGYwlhrUYTrMyhN0WHdZCPAW5rqS0MqQG7qI7dxnjugt6QB39Q==",
      "cpu": [
        "s390x"
      ],
      "dev": true,
      "libc": [
        "glibc"
      ],
      "license": "MIT",
      "optional": true,
      "os": [
        "linux"
      ]
    },
    "node_modules/@rollup/rollup-linux-x64-gnu": {
      "version": "4.65.1",
      "resolved": "https://registry.npmjs.org/@rollup/rollup-linux-x64-gnu/-/rollup-linux-x64-gnu-4.65.1.tgz",
      "integrity": "sha512-44IKzBPqXXVm0kt1O+F5eW4juR5FyIxGNlFkpP8qwIDzucBsjM1HCbdYUQrQw8rU/g6HRl/tfBm0+Qq4H4Ro3w==",
      "cpu": [
        "x64"
      ],
      "dev": true,
      "libc": [
        "glibc"
      ],
      "license": "MIT",
      "optional": true,
      "os": [
        "linux"
      ]
    },
    "node_modules/@rollup/rollup-linux-x64-musl": {
      "version": "4.65.1",
      "resolved": "https://registry.npmjs.org/@rollup/rollup-linux-x64-musl/-/rollup-linux-x64-musl-4.65.1.tgz",
      "integrity": "sha512-tvYWZ/h2PCHs558Xu7B6LptuRCV8lXV2xI75ONMcldII1m2zlFwuI4BJZi9M6rSMkYqcPLLtPUPrjyqfbzLOdg==",
      "cpu": [
        "x64"
      ],
      "dev": true,
      "libc": [
        "musl"
      ],
      "license": "MIT",
      "optional": true,
      "os": [
        "linux"
      ]
    },
    "node_modules/@rollup/rollup-openbsd-x64": {
      "version": "4.65.1",
      "resolved": "https://registry.npmjs.org/@rollup/rollup-openbsd-x64/-/rollup-openbsd-x64-4.65.1.tgz",
      "integrity": "sha512-gdyeucgZCnei3pXwadKVTEAOgRx0B+c847DUPR/PdOMmp3olhfkYo6r0J4PzRFZB5ZICZQTxntyvgmWH5xT7rg==",
      "cpu": [
        "x64"
      ],
      "dev": true,
      "license": "MIT",
      "optional": true,
      "os": [
        "openbsd"
      ]
    },
    "node_modules/@rollup/rollup-openharmony-arm64": {
      "version": "4.65.1",
      "resolved": "https://registry.npmjs.org/@rollup/rollup-openharmony-arm64/-/rollup-openharmony-arm64-4.65.1.tgz",
      "integrity": "sha512-Wf/NEKdn2SAL4Nvt8HyDtdkseWTwiLbZEOoNYnEaHkbso0B9eHO42vKnZh+OYsn+IMhAMcWcuIwPlBOgOXsUzg==",
      "cpu": [
        "arm64"
      ],
      "dev": true,
      "license": "MIT",
      "optional": true,
      "os": [
        "openharmony"
      ]
    },
    "node_modules/@rollup/rollup-win32-arm64-msvc": {
      "version": "4.65.1",
      "resolved": "https://registry.npmjs.org/@rollup/rollup-win32-arm64-msvc/-/rollup-win32-arm64-msvc-4.65.1.tgz",
      "integrity": "sha512-x9VSsDmsMsEWy4cr/mFeaEsXnVJ6Kc4NCAU/lCnwml3s7LQbZoa9+/qwc9etWHbuuDT5mYxdyyKUwo76njtjZw==",
      "cpu": [
        "arm64"
      ],
      "dev": true,
      "license": "MIT",
      "optional": true,
      "os": [
        "win32"
      ]
    },
    "node_modules/@rollup/rollup-win32-ia32-msvc": {
      "version": "4.65.1",
      "resolved": "https://registry.npmjs.org/@rollup/rollup-win32-ia32-msvc/-/rollup-win32-ia32-msvc-4.65.1.tgz",
      "integrity": "sha512-RIDbiVvvLkN45mOFpbo6eYMgnuiBEB62BGR/0s4QJgGg2AvVxucDDNn67kpAcUoYyfmY8iX8/RifMcmxmmnejA==",
      "cpu": [
        "ia32"
      ],
      "dev": true,
      "license": "MIT",
      "optional": true,
      "os": [
        "win32"
      ]
    },
    "node_modules/@rollup/rollup-win32-x64-gnu": {
      "version": "4.65.1",
      "resolved": "https://registry.npmjs.org/@rollup/rollup-win32-x64-gnu/-/rollup-win32-x64-gnu-4.65.1.tgz",
      "integrity": "sha512-+KN8HIr88vN59JNxlrM2fMimzY43Y0lQB7+0xdVsFbvFCAA74CLKCdfzIHzzvndJryBc94Pm+MvwtukKjaS2ug==",
      "cpu": [
        "x64"
      ],
      "dev": true,
      "license": "MIT",
      "optional": true,
      "os": [
        "win32"
      ]
    },
    "node_modules/@rollup/rollup-win32-x64-msvc": {
      "version": "4.65.1",
      "resolved": "https://registry.npmjs.org/@rollup/rollup-win32-x64-msvc/-/rollup-win32-x64-msvc-4.65.1.tgz",
      "integrity": "sha512-+Y1Sd49cSDq60UOl8/U2HI0ebUvnSl/hDhMHQn+ouw17yb8fYwMWIazzJ4JFpSkjPE7sj5NTNSX9IoyEqcFfyQ==",
      "cpu": [
        "x64"
      ],
      "dev": true,
      "license": "MIT",
      "optional": true,
      "os": [
        "win32"
      ]
    },
    "node_modules/@types/babel__core": {
      "version": "7.20.5",
      "resolved": "https://registry.npmjs.org/@types/babel__core/-/babel__core-7.20.5.tgz",
      "integrity": "sha512-qoQprZvz5wQFJwMDqeseRXWv3rqMvhgpbXFfVyWhbx9X47POIA6i/+dXefEmZKoAgOaTdaIgNSMqMIU61yRyzA==",
      "dev": true,
      "license": "MIT",
      "dependencies": {
        "@babel/parser": "^7.20.7",
        "@babel/types": "^7.20.7",
        "@types/babel__generator": "*",
        "@types/babel__template": "*",
        "@types/babel__traverse": "*"
      }
    },
    "node_modules/@types/babel__generator": {
      "version": "7.27.0",
      "resolved": "https://registry.npmjs.org/@types/babel__generator/-/babel__generator-7.27.0.tgz",
      "integrity": "sha512-ufFd2Xi92OAVPYsy+P4n7/U7e68fex0+Ee8gSG9KX7eo084CWiQ4sdxktvdl0bOPupXtVJPY19zk6EwWqUQ8lg==",
      "dev": true,
      "license": "MIT",
      "dependencies": {
        "@babel/types": "^7.0.0"
      }
    },
    "node_modules/@types/babel__template": {
      "version": "7.4.4",
      "resolved": "https://registry.npmjs.org/@types/babel__template/-/babel__template-7.4.4.tgz",
      "integrity": "sha512-h/NUaSyG5EyxBIp8YRxo4RMe2/qQgvyowRwVMzhYhBCONbW8PUsg4lkFMrhgZhUe5z3L3MiLDuvyJ/CaPa2A8A==",
      "dev": true,
      "license": "MIT",
      "dependencies": {
        "@babel/parser": "^7.1.0",
        "@babel/types": "^7.0.0"
      }
    },
    "node_modules/@types/babel__traverse": {
      "version": "7.28.0",
      "resolved": "https://registry.npmjs.org/@types/babel__traverse/-/babel__traverse-7.28.0.tgz",
      "integrity": "sha512-8PvcXf70gTDZBgt9ptxJ8elBeBjcLOAcOtoO/mPJjtji1+CdGbHgm77om1GrsPxsiE+uXIpNSK64UYaIwQXd4Q==",
      "dev": true,
      "license": "MIT",
      "dependencies": {
        "@babel/types": "^7.28.2"
      }
    },
    "node_modules/@types/d3-array": {
      "version": "3.2.2",
      "resolved": "https://registry.npmjs.org/@types/d3-array/-/d3-array-3.2.2.tgz",
      "integrity": "sha512-hOLWVbm7uRza0BYXpIIW5pxfrKe0W+D5lrFiAEYR+pb6w3N2SwSMaJbXdUfSEv+dT4MfHBLtn5js0LAWaO6otw==",
      "license": "MIT"
    },
    "node_modules/@types/d3-color": {
      "version": "3.1.3",
      "resolved": "https://registry.npmjs.org/@types/d3-color/-/d3-color-3.1.3.tgz",
      "integrity": "sha512-iO90scth9WAbmgv7ogoq57O9YpKmFBbmoEoCHDB2xMBY0+/KVrqAaCDyCE16dUspeOvIxFFRI+0sEtqDqy2b4A==",
      "license": "MIT"
    },
    "node_modules/@types/d3-ease": {
      "version": "3.0.2",
      "resolved": "https://registry.npmjs.org/@types/d3-ease/-/d3-ease-3.0.2.tgz",
      "integrity": "sha512-NcV1JjO5oDzoK26oMzbILE6HW7uVXOHLQvHshBUW4UMdZGfiY6v5BeQwh9a9tCzv+CeefZQHJt5SRgK154RtiA==",
      "license": "MIT"
    },
    "node_modules/@types/d3-interpolate": {
      "version": "3.0.4",
      "resolved": "https://registry.npmjs.org/@types/d3-interpolate/-/d3-interpolate-3.0.4.tgz",
      "integrity": "sha512-mgLPETlrpVV1YRJIglr4Ez47g7Yxjl1lj7YKsiMCb27VJH9W8NVM6Bb9d8kkpG/uAQS5AmbA48q2IAolKKo1MA==",
      "license": "MIT",
      "dependencies": {
        "@types/d3-color": "*"
      }
    },
    "node_modules/@types/d3-path": {
      "version": "3.1.1",
      "resolved": "https://registry.npmjs.org/@types/d3-path/-/d3-path-3.1.1.tgz",
      "integrity": "sha512-VMZBYyQvbGmWyWVea0EHs/BwLgxc+MKi1zLDCONksozI4YJMcTt8ZEuIR4Sb1MMTE8MMW49v0IwI5+b7RmfWlg==",
      "license": "MIT"
    },
    "node_modules/@types/d3-scale": {
      "version": "4.0.9",
      "resolved": "https://registry.npmjs.org/@types/d3-scale/-/d3-scale-4.0.9.tgz",
      "integrity": "sha512-dLmtwB8zkAeO/juAMfnV+sItKjlsw2lKdZVVy6LRr0cBmegxSABiLEpGVmSJJ8O08i4+sGR6qQtb6WtuwJdvVw==",
      "license": "MIT",
      "dependencies": {
        "@types/d3-time": "*"
      }
    },
    "node_modules/@types/d3-shape": {
      "version": "3.2.0",
      "resolved": "https://registry.npmjs.org/@types/d3-shape/-/d3-shape-3.2.0.tgz",
      "integrity": "sha512-kVd74ta9eof3eJOvbNd1vGKS/XERRyQbT26Og63hIsvDO84cjD5gEOhsXf26w3FSoNlPVz84DOFcKv/oou+fMw==",
      "license": "MIT",
      "dependencies": {
        "@types/d3-path": "*"
      }
    },
    "node_modules/@types/d3-time": {
      "version": "3.0.4",
      "resolved": "https://registry.npmjs.org/@types/d3-time/-/d3-time-3.0.4.tgz",
      "integrity": "sha512-yuzZug1nkAAaBlBBikKZTgzCeA+k1uy4ZFwWANOfKw5z5LRhV0gNA7gNkKm7HoK+HRN0wX3EkxGk0fpbWhmB7g==",
      "license": "MIT"
    },
    "node_modules/@types/d3-timer": {
      "version": "3.0.2",
      "resolved": "https://registry.npmjs.org/@types/d3-timer/-/d3-timer-3.0.2.tgz",
      "integrity": "sha512-Ps3T8E8dZDam6fUyNiMkekK3XUsaUEik+idO9/YjPtfj2qruF8tFBXS7XhtE4iIXBLxhmLjP3SXpLhVf21I9Lw==",
      "license": "MIT"
    },
    "node_modules/@types/estree": {
      "version": "1.0.9",
      "resolved": "https://registry.npmjs.org/@types/estree/-/estree-1.0.9.tgz",
      "integrity": "sha512-GhdPgy1el4/ImP05X05Uw4cw2/M93BCUmnEvWZNStlCzEKME4Fkk+YpoA5OiHNQmoS7Cafb8Xa3Pya8m1Qrzeg==",
      "dev": true,
      "license": "MIT"
    },
    "node_modules/@vitejs/plugin-react": {
      "version": "4.7.0",
      "resolved": "https://registry.npmjs.org/@vitejs/plugin-react/-/plugin-react-4.7.0.tgz",
      "integrity": "sha512-gUu9hwfWvvEDBBmgtAowQCojwZmJ5mcLn3aufeCsitijs3+f2NsrPtlAWIR6OPiqljl96GVCUbLe0HyqIpVaoA==",
      "dev": true,
      "license": "MIT",
      "dependencies": {
        "@babel/core": "^7.28.0",
        "@babel/plugin-transform-react-jsx-self": "^7.27.1",
        "@babel/plugin-transform-react-jsx-source": "^7.27.1",
        "@rolldown/pluginutils": "1.0.0-beta.27",
        "@types/babel__core": "^7.20.5",
        "react-refresh": "^0.17.0"
      },
      "engines": {
        "node": "^14.18.0 || >=16.0.0"
      },
      "peerDependencies": {
        "vite": "^4.2.0 || ^5.0.0 || ^6.0.0 || ^7.0.0"
      }
    },
    "node_modules/agent-base": {
      "version": "6.0.2",
      "resolved": "https://registry.npmjs.org/agent-base/-/agent-base-6.0.2.tgz",
      "integrity": "sha512-RZNwNclF7+MS/8bDg70amg32dyeZGZxiDuQmZxKLAlQjr3jGyLx+4Kkk58UO7D2QdgFIQCovuSuZESne6RG6XQ==",
      "license": "MIT",
      "dependencies": {
        "debug": "4"
      },
      "engines": {
        "node": ">= 6.0.0"
      }
    },
    "node_modules/asynckit": {
      "version": "0.4.0",
      "resolved": "https://registry.npmjs.org/asynckit/-/asynckit-0.4.0.tgz",
      "integrity": "sha512-Oei9OH4tRh0YqU3GxhX79dM/mwVgvbZJaSNaRk+bshkj0S5cfHcgYakreBjrHwatXKbz+IoIdYLxrKim2MjW0Q==",
      "license": "MIT"
    },
    "node_modules/axios": {
      "version": "1.20.0",
      "resolved": "https://registry.npmjs.org/axios/-/axios-1.20.0.tgz",
      "integrity": "sha512-r8aOh8j9cGKpgQAqpzrUHnSIc6a59Y3Xf/cv8sy1DrHCkZHzQGEuoq1tARk6qSyDdtQGSDgpb9kFlruzPvrgwg==",
      "license": "MIT",
      "dependencies": {
        "follow-redirects": "^1.16.0",
        "form-data": "^4.0.6",
        "https-proxy-agent": "^5.0.1",
        "proxy-from-env": "^2.1.0"
      }
    },
    "node_modules/baseline-browser-mapping": {
      "version": "2.11.33",
      "resolved": "https://registry.npmjs.org/baseline-browser-mapping/-/baseline-browser-mapping-2.11.33.tgz",
      "integrity": "sha512-yiVKDX+qCMRKBE084lXvq9EcwFUhKZ2nU8TK1EQQuOgPNR/lmbGxXOeDdlGUqC2Zz11Cs9rk1d/8NfgBmlkt0w==",
      "dev": true,
      "license": "Apache-2.0",
      "bin": {
        "baseline-browser-mapping": "dist/cli.cjs"
      },
      "engines": {
        "node": ">=6.0.0"
      }
    },
    "node_modules/browserslist": {
      "version": "4.29.5",
      "resolved": "https://registry.npmjs.org/browserslist/-/browserslist-4.29.5.tgz",
      "integrity": "sha512-khTmKu5NGYTotqcYo8Bew2k4RAE9PuXRSFRXG2lQBrCEQHyeQ/vX6gKMxfokwywBsLiXZMnHsmWMG1pnO8J7uQ==",
      "dev": true,
      "funding": [
        {
          "type": "opencollective",
          "url": "https://opencollective.com/browserslist"
        },
        {
          "type": "tidelift",
          "url": "https://tidelift.com/funding/github/npm/browserslist"
        },
        {
          "type": "github",
          "url": "https://github.com/sponsors/ai"
        }
      ],
      "license": "MIT",
      "dependencies": {
        "baseline-browser-mapping": "^2.11.33",
        "caniuse-lite": "^1.0.30001819",
        "electron-to-chromium": "^1.5.464",
        "node-releases": "^2.0.60",
        "update-browserslist-db": "^1.3.4"
      },
      "bin": {
        "browserslist": "cli.js"
      },
      "engines": {
        "node": "^6 || ^7 || ^8 || ^9 || ^10 || ^11 || ^12 || >=13.7"
      }
    },
    "node_modules/call-bind-apply-helpers": {
      "version": "1.0.2",
      "resolved": "https://registry.npmjs.org/call-bind-apply-helpers/-/call-bind-apply-helpers-1.0.2.tgz",
      "integrity": "sha512-Sp1ablJ0ivDkSzjcaJdxEunN5/XvksFJ2sMBFfq6x0ryhQV/2b/KwFe21cMpmHtPOSij8K99/wSfoEuTObmuMQ==",
      "license": "MIT",
      "dependencies": {
        "es-errors": "^1.3.0",
        "function-bind": "^1.1.2"
      },
      "engines": {
        "node": ">= 0.4"
      }
    },
    "node_modules/caniuse-lite": {
      "version": "1.0.30001819",
      "resolved": "https://registry.npmjs.org/caniuse-lite/-/caniuse-lite-1.0.30001819.tgz",
      "integrity": "sha512-LFUqIIAwpzsIuevnKnsR9qnStBcgzPShTX61lBwN9uFkkf8o8zLiebEjbW8AYtZiF+1YKDWfdt/pWDtMuG8Xwg==",
      "dev": true,
      "funding": [
        {
          "type": "opencollective",
          "url": "https://opencollective.com/browserslist"
        },
        {
          "type": "tidelift",
          "url": "https://tidelift.com/funding/github/npm/caniuse-lite"
        },
        {
          "type": "github",
          "url": "https://github.com/sponsors/ai"
        }
      ],
      "license": "CC-BY-4.0"
    },
    "node_modules/clsx": {
      "version": "2.1.1",
      "resolved": "https://registry.npmjs.org/clsx/-/clsx-2.1.1.tgz",
      "integrity": "sha512-eYm0QWBtUrBWZWG0d386OGAw16Z995PiOVo2B7bjWSbHedGl5e0ZWaq65kOGgUSNesEIDkB9ISbTg/JK9dhCZA==",
      "license": "MIT",
      "engines": {
        "node": ">=6"
      }
    },
    "node_modules/combined-stream": {
      "version": "1.0.8",
      "resolved": "https://registry.npmjs.org/combined-stream/-/combined-stream-1.0.8.tgz",
      "integrity": "sha512-FQN4MRfuJeHf7cBbBMJFXhKSDq+2kAArBlmRBvcvFE5BB1HZKXtSFASDhdlz9zOYwxh8lDdnvmMOe/+5cdoEdg==",
      "license": "MIT",
      "dependencies": {
        "delayed-stream": "~1.0.0"
      },
      "engines": {
        "node": ">= 0.8"
      }
    },
    "node_modules/convert-source-map": {
      "version": "2.0.0",
      "resolved": "https://registry.npmjs.org/convert-source-map/-/convert-source-map-2.0.0.tgz",
      "integrity": "sha512-Kvp459HrV2FEJ1CAsi1Ku+MY3kasH19TFykTz2xWmMeq6bk2NU3XXvfJ+Q61m0xktWwt+1HSYf3JZsTms3aRJg==",
      "dev": true,
      "license": "MIT"
    },
    "node_modules/csstype": {
      "version": "3.2.3",
      "resolved": "https://registry.npmjs.org/csstype/-/csstype-3.2.3.tgz",
      "integrity": "sha512-z1HGKcYy2xA8AGQfwrn0PAy+PB7X/GSj3UVJW9qKyn43xWa+gl5nXmU4qqLMRzWVLFC8KusUX8T/0kCiOYpAIQ==",
      "license": "MIT"
    },
    "node_modules/d3-array": {
      "version": "3.2.4",
      "resolved": "https://registry.npmjs.org/d3-array/-/d3-array-3.2.4.tgz",
      "integrity": "sha512-tdQAmyA18i4J7wprpYq8ClcxZy3SC31QMeByyCFyRt7BVHdREQZ5lpzoe5mFEYZUWe+oq8HBvk9JjpibyEV4Jg==",
      "license": "ISC",
      "dependencies": {
        "internmap": "1 - 2"
      },
      "engines": {
        "node": ">=12"
      }
    },
    "node_modules/d3-color": {
      "version": "3.1.0",
      "resolved": "https://registry.npmjs.org/d3-color/-/d3-color-3.1.0.tgz",
      "integrity": "sha512-zg/chbXyeBtMQ1LbD/WSoW2DpC3I0mpmPdW+ynRTj/x2DAWYrIY7qeZIHidozwV24m4iavr15lNwIwLxRmOxhA==",
      "license": "ISC",
      "engines": {
        "node": ">=12"
      }
    },
    "node_modules/d3-ease": {
      "version": "3.0.1",
      "resolved": "https://registry.npmjs.org/d3-ease/-/d3-ease-3.0.1.tgz",
      "integrity": "sha512-wR/XK3D3XcLIZwpbvQwQ5fK+8Ykds1ip7A2Txe0yxncXSdq1L9skcG7blcedkOX+ZcgxGAmLX1FrRGbADwzi0w==",
      "license": "BSD-3-Clause",
      "engines": {
        "node": ">=12"
      }
    },
    "node_modules/d3-format": {
      "version": "3.1.2",
      "resolved": "https://registry.npmjs.org/d3-format/-/d3-format-3.1.2.tgz",
      "integrity": "sha512-AJDdYOdnyRDV5b6ArilzCPPwc1ejkHcoyFarqlPqT7zRYjhavcT3uSrqcMvsgh2CgoPbK3RCwyHaVyxYcP2Arg==",
      "license": "ISC",
      "engines": {
        "node": ">=12"
      }
    },
    "node_modules/d3-interpolate": {
      "version": "3.0.1",
      "resolved": "https://registry.npmjs.org/d3-interpolate/-/d3-interpolate-3.0.1.tgz",
      "integrity": "sha512-3bYs1rOD33uo8aqJfKP3JWPAibgw8Zm2+L9vBKEHJ2Rg+viTR7o5Mmv5mZcieN+FRYaAOWX5SJATX6k1PWz72g==",
      "license": "ISC",
      "dependencies": {
        "d3-color": "1 - 3"
      },
      "engines": {
        "node": ">=12"
      }
    },
    "node_modules/d3-path": {
      "version": "3.1.0",
      "resolved": "https://registry.npmjs.org/d3-path/-/d3-path-3.1.0.tgz",
      "integrity": "sha512-p3KP5HCf/bvjBSSKuXid6Zqijx7wIfNW+J/maPs+iwR35at5JCbLUT0LzF1cnjbCHWhqzQTIN2Jpe8pRebIEFQ==",
      "license": "ISC",
      "engines": {
        "node": ">=12"
      }
    },
    "node_modules/d3-scale": {
      "version": "4.0.2",
      "resolved": "https://registry.npmjs.org/d3-scale/-/d3-scale-4.0.2.tgz",
      "integrity": "sha512-GZW464g1SH7ag3Y7hXjf8RoUuAFIqklOAq3MRl4OaWabTFJY9PN/E1YklhXLh+OQ3fM9yS2nOkCoS+WLZ6kvxQ==",
      "license": "ISC",
      "dependencies": {
        "d3-array": "2.10.0 - 3",
        "d3-format": "1 - 3",
        "d3-interpolate": "1.2.0 - 3",
        "d3-time": "2.1.1 - 3",
        "d3-time-format": "2 - 4"
      },
      "engines": {
        "node": ">=12"
      }
    },
    "node_modules/d3-shape": {
      "version": "3.2.0",
      "resolved": "https://registry.npmjs.org/d3-shape/-/d3-shape-3.2.0.tgz",
      "integrity": "sha512-SaLBuwGm3MOViRq2ABk3eLoxwZELpH6zhl3FbAoJ7Vm1gofKx6El1Ib5z23NUEhF9AsGl7y+dzLe5Cw2AArGTA==",
      "license": "ISC",
      "dependencies": {
        "d3-path": "^3.1.0"
      },
      "engines": {
        "node": ">=12"
      }
    },
    "node_modules/d3-time": {
      "version": "3.1.0",
      "resolved": "https://registry.npmjs.org/d3-time/-/d3-time-3.1.0.tgz",
      "integrity": "sha512-VqKjzBLejbSMT4IgbmVgDjpkYrNWUYJnbCGo874u7MMKIWsILRX+OpX/gTk8MqjpT1A/c6HY2dCA77ZN0lkQ2Q==",
      "license": "ISC",
      "dependencies": {
        "d3-array": "2 - 3"
      },
      "engines": {
        "node": ">=12"
      }
    },
    "node_modules/d3-time-format": {
      "version": "4.1.0",
      "resolved": "https://registry.npmjs.org/d3-time-format/-/d3-time-format-4.1.0.tgz",
      "integrity": "sha512-dJxPBlzC7NugB2PDLwo9Q8JiTR3M3e4/XANkreKSUxF8vvXKqm1Yfq4Q5dl8budlunRVlUUaDUgFt7eA8D6NLg==",
      "license": "ISC",
      "dependencies": {
        "d3-time": "1 - 3"
      },
      "engines": {
        "node": ">=12"
      }
    },
    "node_modules/d3-timer": {
      "version": "3.0.1",
      "resolved": "https://registry.npmjs.org/d3-timer/-/d3-timer-3.0.1.tgz",
      "integrity": "sha512-ndfJ/JxxMd3nw31uyKoY2naivF+r29V+Lc0svZxe1JvvIRmi8hUsrMvdOwgS1o6uBHmiz91geQ0ylPP0aj1VUA==",
      "license": "ISC",
      "engines": {
        "node": ">=12"
      }
    },
    "node_modules/debug": {
      "version": "4.4.3",
      "resolved": "https://registry.npmjs.org/debug/-/debug-4.4.3.tgz",
      "integrity": "sha512-RGwwWnwQvkVfavKVt22FGLw+xYSdzARwm0ru6DhTVA3umU5hZc28V3kO4stgYryrTlLpuvgI9GiijltAjNbcqA==",
      "license": "MIT",
      "dependencies": {
        "ms": "^2.1.3"
      },
      "engines": {
        "node": ">=6.0"
      },
      "peerDependenciesMeta": {
        "supports-color": {
          "optional": true
        }
      }
    },
    "node_modules/decimal.js-light": {
      "version": "2.5.1",
      "resolved": "https://registry.npmjs.org/decimal.js-light/-/decimal.js-light-2.5.1.tgz",
      "integrity": "sha512-qIMFpTMZmny+MMIitAB6D7iVPEorVw6YQRWkvarTkT4tBeSLLiHzcwj6q0MmYSFCiVpiqPJTJEYIrpcPzVEIvg==",
      "license": "MIT"
    },
    "node_modules/delayed-stream": {
      "version": "1.0.0",
      "resolved": "https://registry.npmjs.org/delayed-stream/-/delayed-stream-1.0.0.tgz",
      "integrity": "sha512-ZySD7Nf91aLB0RxL4KGrKHBXl7Eds1DAmEdcoVawXnLD7SDhpNgtuII2aAkg7a7QS41jxPSZ17p4VdGnMHk3MQ==",
      "license": "MIT",
      "engines": {
        "node": ">=0.4.0"
      }
    },
    "node_modules/dom-helpers": {
      "version": "5.2.1",
      "resolved": "https://registry.npmjs.org/dom-helpers/-/dom-helpers-5.2.1.tgz",
      "integrity": "sha512-nRCa7CK3VTrM2NmGkIy4cbK7IZlgBE/PYMn55rrXefr5xXDP0LdtfPnblFDoVdcAfslJ7or6iqAUnx0CCGIWQA==",
      "license": "MIT",
      "dependencies": {
        "@babel/runtime": "^7.8.7",
        "csstype": "^3.0.2"
      }
    },
    "node_modules/dunder-proto": {
      "version": "1.0.1",
      "resolved": "https://registry.npmjs.org/dunder-proto/-/dunder-proto-1.0.1.tgz",
      "integrity": "sha512-KIN/nDJBQRcXw0MLVhZE9iQHmG68qAVIBg9CqmUYjmQIhgij9U5MFvrqkUL5FbtyyzZuOeOt0zdeRe4UY7ct+A==",
      "license": "MIT",
      "dependencies": {
        "call-bind-apply-helpers": "^1.0.1",
        "es-errors": "^1.3.0",
        "gopd": "^1.2.0"
      },
      "engines": {
        "node": ">= 0.4"
      }
    },
    "node_modules/electron-to-chromium": {
      "version": "1.5.465",
      "resolved": "https://registry.npmjs.org/electron-to-chromium/-/electron-to-chromium-1.5.465.tgz",
      "integrity": "sha512-zg4iUpbeGw7YNZsswc2xKr2uiBex2d8dELAXo92Icw3but1ULbrvK8USe6wVLaWjgJnKBARv0pchx9Owktx6wQ==",
      "dev": true,
      "license": "ISC"
    },
    "node_modules/es-define-property": {
      "version": "1.0.1",
      "resolved": "https://registry.npmjs.org/es-define-property/-/es-define-property-1.0.1.tgz",
      "integrity": "sha512-e3nRfgfUZ4rNGL232gUgX06QNyyez04KdjFrF+LTRoOXmrOgFKDg4BCdsjW8EnT69eqdYGmRpJwiPVYNrCaW3g==",
      "license": "MIT",
      "engines": {
        "node": ">= 0.4"
      }
    },
    "node_modules/es-errors": {
      "version": "1.3.0",
      "resolved": "https://registry.npmjs.org/es-errors/-/es-errors-1.3.0.tgz",
      "integrity": "sha512-Zf5H2Kxt2xjTvbJvP2ZWLEICxA6j+hAmMzIlypy4xcBg1vKVnx89Wy0GbS+kf5cwCVFFzdCFh2XSCFNULS6csw==",
      "license": "MIT",
      "engines": {
        "node": ">= 0.4"
      }
    },
    "node_modules/es-object-atoms": {
      "version": "1.1.2",
      "resolved": "https://registry.npmjs.org/es-object-atoms/-/es-object-atoms-1.1.2.tgz",
      "integrity": "sha512-HWcBoN6NileqtSydK2FqHbS/LoDd2pqrnQHLyJzBj4kOp/ky2MWMN694xOfkK8/SnUsW2DH7EfyVlydKCsm1Zw==",
      "license": "MIT",
      "dependencies": {
        "es-errors": "^1.3.0"
      },
      "engines": {
        "node": ">= 0.4"
      }
    },
    "node_modules/es-set-tostringtag": {
      "version": "2.1.0",
      "resolved": "https://registry.npmjs.org/es-set-tostringtag/-/es-set-tostringtag-2.1.0.tgz",
      "integrity": "sha512-j6vWzfrGVfyXxge+O0x5sh6cvxAog0a/4Rdd2K36zCMV5eJ+/+tOAngRO8cODMNWbVRdVlmGZQL2YS3yR8bIUA==",
      "license": "MIT",
      "dependencies": {
        "es-errors": "^1.3.0",
        "get-intrinsic": "^1.2.6",
        "has-tostringtag": "^1.0.2",
        "hasown": "^2.0.2"
      },
      "engines": {
        "node": ">= 0.4"
      }
    },
    "node_modules/esbuild": {
      "version": "0.25.12",
      "resolved": "https://registry.npmjs.org/esbuild/-/esbuild-0.25.12.tgz",
      "integrity": "sha512-bbPBYYrtZbkt6Os6FiTLCTFxvq4tt3JKall1vRwshA3fdVztsLAatFaZobhkBC8/BrPetoa0oksYoKXoG4ryJg==",
      "dev": true,
      "hasInstallScript": true,
      "license": "MIT",
      "bin": {
        "esbuild": "bin/esbuild"
      },
      "engines": {
        "node": ">=18"
      },
      "optionalDependencies": {
        "@esbuild/aix-ppc64": "0.25.12",
        "@esbuild/android-arm": "0.25.12",
        "@esbuild/android-arm64": "0.25.12",
        "@esbuild/android-x64": "0.25.12",
        "@esbuild/darwin-arm64": "0.25.12",
        "@esbuild/darwin-x64": "0.25.12",
        "@esbuild/freebsd-arm64": "0.25.12",
        "@esbuild/freebsd-x64": "0.25.12",
        "@esbuild/linux-arm": "0.25.12",
        "@esbuild/linux-arm64": "0.25.12",
        "@esbuild/linux-ia32": "0.25.12",
        "@esbuild/linux-loong64": "0.25.12",
        "@esbuild/linux-mips64el": "0.25.12",
        "@esbuild/linux-ppc64": "0.25.12",
        "@esbuild/linux-riscv64": "0.25.12",
        "@esbuild/linux-s390x": "0.25.12",
        "@esbuild/linux-x64": "0.25.12",
        "@esbuild/netbsd-arm64": "0.25.12",
        "@esbuild/netbsd-x64": "0.25.12",
        "@esbuild/openbsd-arm64": "0.25.12",
        "@esbuild/openbsd-x64": "0.25.12",
        "@esbuild/openharmony-arm64": "0.25.12",
        "@esbuild/sunos-x64": "0.25.12",
        "@esbuild/win32-arm64": "0.25.12",
        "@esbuild/win32-ia32": "0.25.12",
        "@esbuild/win32-x64": "0.25.12"
      }
    },
    "node_modules/escalade": {
      "version": "3.2.0",
      "resolved": "https://registry.npmjs.org/escalade/-/escalade-3.2.0.tgz",
      "integrity": "sha512-WUj2qlxaQtO4g6Pq5c29GTcWGDyd8itL8zTlipgECz3JesAiiOKotd8JU6otB3PACgG6xkJUyVhboMS+bje/jA==",
      "dev": true,
      "license": "MIT",
      "engines": {
        "node": ">=6"
      }
    },
    "node_modules/eventemitter3": {
      "version": "4.0.7",
      "resolved": "https://registry.npmjs.org/eventemitter3/-/eventemitter3-4.0.7.tgz",
      "integrity": "sha512-8guHBZCwKnFhYdHr2ysuRWErTwhoN2X8XELRlrRwpmfeY2jjuUN4taQMsULKUVo1K4DvZl+0pgfyoysHxvmvEw==",
      "license": "MIT"
    },
    "node_modules/fast-equals": {
      "version": "5.4.5",
      "resolved": "https://registry.npmjs.org/fast-equals/-/fast-equals-5.4.5.tgz",
      "integrity": "sha512-yKEpcFuRPu/pTQL2VB8YKE2yFldrLlIsoWtJeBnAdqEhvu21OgkFT0C8njf/xYc2LSZGEr3K9JGjthHJxnVLvQ==",
      "license": "MIT",
      "engines": {
        "node": ">=6.0.0"
      }
    },
    "node_modules/fdir": {
      "version": "6.5.0",
      "resolved": "https://registry.npmjs.org/fdir/-/fdir-6.5.0.tgz",
      "integrity": "sha512-tIbYtZbucOs0BRGqPJkshJUYdL+SDH7dVM8gjy+ERp3WAUjLEFJE+02kanyHtwjWOnwrKYBiwAmM0p4kLJAnXg==",
      "dev": true,
      "license": "MIT",
      "engines": {
        "node": ">=12.0.0"
      },
      "peerDependencies": {
        "picomatch": "^3 || ^4"
      },
      "peerDependenciesMeta": {
        "picomatch": {
          "optional": true
        }
      }
    },
    "node_modules/follow-redirects": {
      "version": "1.16.2",
      "resolved": "https://registry.npmjs.org/follow-redirects/-/follow-redirects-1.16.2.tgz",
      "integrity": "sha512-/kVP2U8z596Ia0gwFrwyIeo74ta5aVmSeU1FfAUMAmcEP5EP7zydmag3SLiN64NaRKef3lxsq7i/g7ltJQDGow==",
      "funding": [
        {
          "type": "individual",
          "url": "https://github.com/sponsors/RubenVerborgh"
        }
      ],
      "license": "MIT",
      "engines": {
        "node": ">=4.0"
      },
      "peerDependenciesMeta": {
        "debug": {
          "optional": true
        }
      }
    },
    "node_modules/form-data": {
      "version": "4.0.6",
      "resolved": "https://registry.npmjs.org/form-data/-/form-data-4.0.6.tgz",
      "integrity": "sha512-vKatAh4SlVfgbv+YtmhiRjhEMJsYpsG1Y2rMQtR+SVSbytsSD1YGzDIcrAJmdFec88u/+VoGmxnl+80gL1tRCQ==",
      "license": "MIT",
      "dependencies": {
        "asynckit": "^0.4.0",
        "combined-stream": "^1.0.8",
        "es-set-tostringtag": "^2.1.0",
        "hasown": "^2.0.4",
        "mime-types": "^2.1.35"
      },
      "engines": {
        "node": ">= 6"
      }
    },
    "node_modules/fsevents": {
      "version": "2.3.3",
      "resolved": "https://registry.npmjs.org/fsevents/-/fsevents-2.3.3.tgz",
      "integrity": "sha512-5xoDfX+fL7faATnagmWPpbFtwh/R77WmMMqqHGS65C3vvB0YHrgF+B1YmZ3441tMj5n63k0212XNoJwzlhffQw==",
      "dev": true,
      "hasInstallScript": true,
      "license": "MIT",
      "optional": true,
      "os": [
        "darwin"
      ],
      "engines": {
        "node": "^8.16.0 || ^10.6.0 || >=11.0.0"
      }
    },
    "node_modules/function-bind": {
      "version": "1.1.2",
      "resolved": "https://registry.npmjs.org/function-bind/-/function-bind-1.1.2.tgz",
      "integrity": "sha512-7XHNxH7qX9xG5mIwxkhumTox/MIRNcOgDrxWsMt2pAr23WHp6MrRlN7FBSFpCpr+oVO0F744iUgR82nJMfG2SA==",
      "license": "MIT",
      "funding": {
        "url": "https://github.com/sponsors/ljharb"
      }
    },
    "node_modules/gensync": {
      "version": "1.0.0-beta.2",
      "resolved": "https://registry.npmjs.org/gensync/-/gensync-1.0.0-beta.2.tgz",
      "integrity": "sha512-3hN7NaskYvMDLQY55gnW3NQ+mesEAepTqlg+VEbj7zzqEMBVNhzcGYYeqFo/TlYz6eQiFcp1HcsCZO+nGgS8zg==",
      "dev": true,
      "license": "MIT",
      "engines": {
        "node": ">=6.9.0"
      }
    },
    "node_modules/get-intrinsic": {
      "version": "1.3.0",
      "resolved": "https://registry.npmjs.org/get-intrinsic/-/get-intrinsic-1.3.0.tgz",
      "integrity": "sha512-9fSjSaos/fRIVIp+xSJlE6lfwhES7LNtKaCBIamHsjr2na1BiABJPo0mOjjz8GJDURarmCPGqaiVg5mfjb98CQ==",
      "license": "MIT",
      "dependencies": {
        "call-bind-apply-helpers": "^1.0.2",
        "es-define-property": "^1.0.1",
        "es-errors": "^1.3.0",
        "es-object-atoms": "^1.1.1",
        "function-bind": "^1.1.2",
        "get-proto": "^1.0.1",
        "gopd": "^1.2.0",
        "has-symbols": "^1.1.0",
        "hasown": "^2.0.2",
        "math-intrinsics": "^1.1.0"
      },
      "engines": {
        "node": ">= 0.4"
      },
      "funding": {
        "url": "https://github.com/sponsors/ljharb"
      }
    },
    "node_modules/get-proto": {
      "version": "1.0.1",
      "resolved": "https://registry.npmjs.org/get-proto/-/get-proto-1.0.1.tgz",
      "integrity": "sha512-sTSfBjoXBp89JvIKIefqw7U2CCebsc74kiY6awiGogKtoSGbgjYE/G/+l9sF3MWFPNc9IcoOC4ODfKHfxFmp0g==",
      "license": "MIT",
      "dependencies": {
        "dunder-proto": "^1.0.1",
        "es-object-atoms": "^1.0.0"
      },
      "engines": {
        "node": ">= 0.4"
      }
    },
    "node_modules/gopd": {
      "version": "1.2.0",
      "resolved": "https://registry.npmjs.org/gopd/-/gopd-1.2.0.tgz",
      "integrity": "sha512-ZUKRh6/kUFoAiTAtTYPZJ3hw9wNxx+BIBOijnlG9PnrJsCcSjs1wyyD6vJpaYtgnzDrKYRSqf3OO6Rfa93xsRg==",
      "license": "MIT",
      "engines": {
        "node": ">= 0.4"
      },
      "funding": {
        "url": "https://github.com/sponsors/ljharb"
      }
    },
    "node_modules/has-symbols": {
      "version": "1.1.0",
      "resolved": "https://registry.npmjs.org/has-symbols/-/has-symbols-1.1.0.tgz",
      "integrity": "sha512-1cDNdwJ2Jaohmb3sg4OmKaMBwuC48sYni5HUw2DvsC8LjGTLK9h+eb1X6RyuOHe4hT0ULCW68iomhjUoKUqlPQ==",
      "license": "MIT",
      "engines": {
        "node": ">= 0.4"
      },
      "funding": {
        "url": "https://github.com/sponsors/ljharb"
      }
    },
    "node_modules/has-tostringtag": {
      "version": "1.0.2",
      "resolved": "https://registry.npmjs.org/has-tostringtag/-/has-tostringtag-1.0.2.tgz",
      "integrity": "sha512-NqADB8VjPFLM2V0VvHUewwwsw0ZWBaIdgo+ieHtK3hasLz4qeCRjYcqfB6AQrBggRKppKF8L52/VqdVsO47Dlw==",
      "license": "MIT",
      "dependencies": {
        "has-symbols": "^1.0.3"
      },
      "engines": {
        "node": ">= 0.4"
      },
      "funding": {
        "url": "https://github.com/sponsors/ljharb"
      }
    },
    "node_modules/hasown": {
      "version": "2.0.4",
      "resolved": "https://registry.npmjs.org/hasown/-/hasown-2.0.4.tgz",
      "integrity": "sha512-T2UbfbBEF32wiepXIsMlTW9+dDYC6wMh/t/vYA4tuOMKqWz/n3vr1NFSxQiyP+zk2mXsoMA/i/7qV6LKut1t1A==",
      "license": "MIT",
      "dependencies": {
        "function-bind": "^1.1.2"
      },
      "engines": {
        "node": ">= 0.4"
      }
    },
    "node_modules/https-proxy-agent": {
      "version": "5.0.1",
      "resolved": "https://registry.npmjs.org/https-proxy-agent/-/https-proxy-agent-5.0.1.tgz",
      "integrity": "sha512-dFcAjpTQFgoLMzC2VwU+C/CbS7uRL0lWmxDITmqm7C+7F0Odmj6s9l6alZc6AELXhrnggM2CeWSXHGOdX2YtwA==",
      "license": "MIT",
      "dependencies": {
        "agent-base": "6",
        "debug": "4"
      },
      "engines": {
        "node": ">= 6"
      }
    },
    "node_modules/internmap": {
      "version": "2.0.3",
      "resolved": "https://registry.npmjs.org/internmap/-/internmap-2.0.3.tgz",
      "integrity": "sha512-5Hh7Y1wQbvY5ooGgPbDaL5iYLAPzMTUrjMulskHLH6wnv/A+1q5rgEaiuqEjB+oxGXIVZs1FF+R/KPN3ZSQYYg==",
      "license": "ISC",
      "engines": {
        "node": ">=12"
      }
    },
    "node_modules/js-tokens": {
      "version": "4.0.0",
      "resolved": "https://registry.npmjs.org/js-tokens/-/js-tokens-4.0.0.tgz",
      "integrity": "sha512-RdJUflcE3cUzKiMqQgsCu06FPu9UdIJO0beYbPhHN4k6apgJtifcoCtT9bcxOpYBtpD2kCM6Sbzg4CausW/PKQ==",
      "license": "MIT"
    },
    "node_modules/jsesc": {
      "version": "3.1.0",
      "resolved": "https://registry.npmjs.org/jsesc/-/jsesc-3.1.0.tgz",
      "integrity": "sha512-/sM3dO2FOzXjKQhJuo0Q173wf2KOo8t4I8vHy6lF9poUp7bKT0/NHE8fPX23PwfhnykfqnC2xRxOnVw5XuGIaA==",
      "dev": true,
      "license": "MIT",
      "bin": {
        "jsesc": "bin/jsesc"
      },
      "engines": {
        "node": ">=6"
      }
    },
    "node_modules/json5": {
      "version": "2.2.3",
      "resolved": "https://registry.npmjs.org/json5/-/json5-2.2.3.tgz",
      "integrity": "sha512-XmOWe7eyHYH14cLdVPoyg+GOH3rYX++KpzrylJwSW98t3Nk+U8XOl8FWKOgwtzdb8lXGf6zYwDUzeHMWfxasyg==",
      "dev": true,
      "license": "MIT",
      "bin": {
        "json5": "lib/cli.js"
      },
      "engines": {
        "node": ">=6"
      }
    },
    "node_modules/lodash": {
      "version": "4.18.1",
      "resolved": "https://registry.npmjs.org/lodash/-/lodash-4.18.1.tgz",
      "integrity": "sha512-dMInicTPVE8d1e5otfwmmjlxkZoUpiVLwyeTdUsi/Caj/gfzzblBcCE5sRHV/AsjuCmxWrte2TNGSYuCeCq+0Q==",
      "license": "MIT"
    },
    "node_modules/loose-envify": {
      "version": "1.4.0",
      "resolved": "https://registry.npmjs.org/loose-envify/-/loose-envify-1.4.0.tgz",
      "integrity": "sha512-lyuxPGr/Wfhrlem2CL/UcnUc1zcqKAImBDzukY7Y5F/yQiNdko6+fRLevlw1HgMySw7f611UIY408EtxRSoK3Q==",
      "license": "MIT",
      "dependencies": {
        "js-tokens": "^3.0.0 || ^4.0.0"
      },
      "bin": {
        "loose-envify": "cli.js"
      }
    },
    "node_modules/lru-cache": {
      "version": "5.1.1",
      "resolved": "https://registry.npmjs.org/lru-cache/-/lru-cache-5.1.1.tgz",
      "integrity": "sha512-KpNARQA3Iwv+jTA0utUVVbrh+Jlrr1Fv0e56GGzAFOXN7dk/FviaDW8LHmK52DlcH4WP2n6gI8vN1aesBFgo9w==",
      "dev": true,
      "license": "ISC",
      "dependencies": {
        "yallist": "^3.0.2"
      }
    },
    "node_modules/lucide-react": {
      "version": "0.468.0",
      "resolved": "https://registry.npmjs.org/lucide-react/-/lucide-react-0.468.0.tgz",
      "integrity": "sha512-6koYRhnM2N0GGZIdXzSeiNwguv1gt/FAjZOiPl76roBi3xKEXa4WmfpxgQwTTL4KipXjefrnf3oV4IsYhi4JFA==",
      "license": "ISC",
      "peerDependencies": {
        "react": "^16.5.1 || ^17.0.0 || ^18.0.0 || ^19.0.0-rc"
      }
    },
    "node_modules/math-intrinsics": {
      "version": "1.1.0",
      "resolved": "https://registry.npmjs.org/math-intrinsics/-/math-intrinsics-1.1.0.tgz",
      "integrity": "sha512-/IXtbwEk5HTPyEwyKX6hGkYXxM9nbj64B+ilVJnC/R6B0pH5G4V3b0pVbL7DBj4tkhBAppbQUlf6F6Xl9LHu1g==",
      "license": "MIT",
      "engines": {
        "node": ">= 0.4"
      }
    },
    "node_modules/mime-db": {
      "version": "1.52.0",
      "resolved": "https://registry.npmjs.org/mime-db/-/mime-db-1.52.0.tgz",
      "integrity": "sha512-sPU4uV7dYlvtWJxwwxHD0PuihVNiE7TyAbQ5SWxDCB9mUYvOgroQOwYQQOKPJ8CIbE+1ETVlOoK1UC2nU3gYvg==",
      "license": "MIT",
      "engines": {
        "node": ">= 0.6"
      }
    },
    "node_modules/mime-types": {
      "version": "2.1.35",
      "resolved": "https://registry.npmjs.org/mime-types/-/mime-types-2.1.35.tgz",
      "integrity": "sha512-ZDY+bPm5zTTF+YpCrAU9nK0UgICYPT0QtT1NZWFv4s++TNkcgVaT0g6+4R2uI4MjQjzysHB1zxuWL50hzaeXiw==",
      "license": "MIT",
      "dependencies": {
        "mime-db": "1.52.0"
      },
      "engines": {
        "node": ">= 0.6"
      }
    },
    "node_modules/ms": {
      "version": "2.1.3",
      "resolved": "https://registry.npmjs.org/ms/-/ms-2.1.3.tgz",
      "integrity": "sha512-6FlzubTLZG3J2a/NVCAleEhjzq5oxgHyaCU9yYXvcLsvoVaHJq/s5xXI6/XXP6tz7R9xAOtHnSO/tXtF3WRTlA==",
      "license": "MIT"
    },
    "node_modules/nanoid": {
      "version": "3.3.20",
      "resolved": "https://registry.npmjs.org/nanoid/-/nanoid-3.3.20.tgz",
      "integrity": "sha512-uKdg2G3GNCKQn9byYOpxbGqrT2fGO5KRt5J/8b3pok8rT6qxGWF6hxMyJiEYtAf+FVyYuD9hRaDqX5uPFYJ4ZQ==",
      "dev": true,
      "funding": [
        {
          "type": "github",
          "url": "https://github.com/sponsors/ai"
        }
      ],
      "license": "MIT",
      "bin": {
        "nanoid": "bin/nanoid.cjs"
      },
      "engines": {
        "node": "^10 || ^12 || ^13.7 || ^14 || >=15.0.1"
      }
    },
    "node_modules/node-releases": {
      "version": "2.0.60",
      "resolved": "https://registry.npmjs.org/node-releases/-/node-releases-2.0.60.tgz",
      "integrity": "sha512-OmO8+GM1JUZqXET720VrJCwkjUCXmCMQm0Z3UDDRbfNLHS4NEAvY60j5jiFv0tkKw2PNNlqOLhahVmwJIOLDIQ==",
      "dev": true,
      "license": "MIT",
      "engines": {
        "node": ">=18"
      }
    },
    "node_modules/object-assign": {
      "version": "4.1.1",
      "resolved": "https://registry.npmjs.org/object-assign/-/object-assign-4.1.1.tgz",
      "integrity": "sha512-rJgTQnkUnH1sFw8yT6VSU3zD3sWmu6sZhIseY8VX+GRu3P6F7Fu+JNDoXfklElbLJSnc3FUQHVe4cU5hj+BcUg==",
      "license": "MIT",
      "engines": {
        "node": ">=0.10.0"
      }
    },
    "node_modules/picocolors": {
      "version": "1.1.1",
      "resolved": "https://registry.npmjs.org/picocolors/-/picocolors-1.1.1.tgz",
      "integrity": "sha512-xceH2snhtb5M9liqDsmEw56le376mTZkEX/jEb/RxNFyegNul7eNslCXP9FDj/Lcu0X8KEyMceP2ntpaHrDEVA==",
      "dev": true,
      "license": "ISC"
    },
    "node_modules/picomatch": {
      "version": "4.0.7",
      "resolved": "https://registry.npmjs.org/picomatch/-/picomatch-4.0.7.tgz",
      "integrity": "sha512-qcJu88Q2IWqJsDD529JKMdwGm/dvInW4HvQnRwiH9JtihJvzGOscDtHE3x1pBKeUOTysQ8kVmLnJ2kJu7yhcGA==",
      "dev": true,
      "license": "MIT",
      "engines": {
        "node": ">=12"
      },
      "funding": {
        "url": "https://github.com/sponsors/jonschlinkert"
      }
    },
    "node_modules/postcss": {
      "version": "8.5.32",
      "resolved": "https://registry.npmjs.org/postcss/-/postcss-8.5.32.tgz",
      "integrity": "sha512-LJlKCCAUFKOuZxipUdYw1NzIITEsPcgyOG9LkvRJvuKj/bRzbuRY0ARXvL0pWGpsZvn8jku+LMep2eU0seHp8g==",
      "dev": true,
      "funding": [
        {
          "type": "opencollective",
          "url": "https://opencollective.com/postcss/"
        },
        {
          "type": "tidelift",
          "url": "https://tidelift.com/funding/github/npm/postcss"
        },
        {
          "type": "github",
          "url": "https://github.com/sponsors/ai"
        }
      ],
      "license": "MIT",
      "dependencies": {
        "nanoid": "^3.3.20",
        "picocolors": "^1.1.1",
        "source-map-js": "^1.2.2"
      },
      "engines": {
        "node": "^10 || ^12 || >=14"
      }
    },
    "node_modules/prop-types": {
      "version": "15.8.1",
      "resolved": "https://registry.npmjs.org/prop-types/-/prop-types-15.8.1.tgz",
      "integrity": "sha512-oj87CgZICdulUohogVAR7AjlC0327U4el4L6eAvOqCeudMDVU0NThNaV+b9Df4dXgSP1gXMTnPdhfe/2qDH5cg==",
      "license": "MIT",
      "dependencies": {
        "loose-envify": "^1.4.0",
        "object-assign": "^4.1.1",
        "react-is": "^16.13.1"
      }
    },
    "node_modules/prop-types/node_modules/react-is": {
      "version": "16.13.1",
      "resolved": "https://registry.npmjs.org/react-is/-/react-is-16.13.1.tgz",
      "integrity": "sha512-24e6ynE2H+OKt4kqsOvNd8kBpV65zoxbA4BVsEOB3ARVWQki/DHzaUoC5KuON/BiccDaCCTZBuOcfZs70kR8bQ==",
      "license": "MIT"
    },
    "node_modules/proxy-from-env": {
      "version": "2.1.0",
      "resolved": "https://registry.npmjs.org/proxy-from-env/-/proxy-from-env-2.1.0.tgz",
      "integrity": "sha512-cJ+oHTW1VAEa8cJslgmUZrc+sjRKgAKl3Zyse6+PV38hZe/V6Z14TbCuXcan9F9ghlz4QrFr2c92TNF82UkYHA==",
      "license": "MIT",
      "engines": {
        "node": ">=10"
      }
    },
    "node_modules/react": {
      "version": "18.3.1",
      "resolved": "https://registry.npmjs.org/react/-/react-18.3.1.tgz",
      "integrity": "sha512-wS+hAgJShR0KhEvPJArfuPVN1+Hz1t0Y6n5jLrGQbkb4urgPE/0Rve+1kMB1v/oWgHgm4WIcV+i7F2pTVj+2iQ==",
      "license": "MIT",
      "dependencies": {
        "loose-envify": "^1.1.0"
      },
      "engines": {
        "node": ">=0.10.0"
      }
    },
    "node_modules/react-dom": {
      "version": "18.3.1",
      "resolved": "https://registry.npmjs.org/react-dom/-/react-dom-18.3.1.tgz",
      "integrity": "sha512-5m4nQKp+rZRb09LNH59GM4BxTh9251/ylbKIbpe7TpGxfJ+9kv6BLkLBXIjjspbgbnIBNqlI23tRnTWT0snUIw==",
      "license": "MIT",
      "dependencies": {
        "loose-envify": "^1.1.0",
        "scheduler": "^0.23.2"
      },
      "peerDependencies": {
        "react": "^18.3.1"
      }
    },
    "node_modules/react-is": {
      "version": "18.3.1",
      "resolved": "https://registry.npmjs.org/react-is/-/react-is-18.3.1.tgz",
      "integrity": "sha512-/LLMVyas0ljjAtoYiPqYiL8VWXzUUdThrmU5+n20DZv+a+ClRoevUzw5JxU+Ieh5/c87ytoTBV9G1FiKfNJdmg==",
      "license": "MIT"
    },
    "node_modules/react-refresh": {
      "version": "0.17.0",
      "resolved": "https://registry.npmjs.org/react-refresh/-/react-refresh-0.17.0.tgz",
      "integrity": "sha512-z6F7K9bV85EfseRCp2bzrpyQ0Gkw1uLoCel9XBVWPg/TjRj94SkJzUTGfOa4bs7iJvBWtQG0Wq7wnI0syw3EBQ==",
      "dev": true,
      "license": "MIT",
      "engines": {
        "node": ">=0.10.0"
      }
    },
    "node_modules/react-router": {
      "version": "6.30.6",
      "resolved": "https://registry.npmjs.org/react-router/-/react-router-6.30.6.tgz",
      "integrity": "sha512-5HfK7k5im7LTOB0EqCQmfvy4C13G92Ssj1VTmouTK3AJvyjKTnFuCV0vcMAD/JS+JC4DvDIBRrlAeJIFjh5VWg==",
      "license": "MIT",
      "dependencies": {
        "@remix-run/router": "1.23.4"
      },
      "engines": {
        "node": ">=14.0.0"
      },
      "peerDependencies": {
        "react": ">=16.8"
      }
    },
    "node_modules/react-router-dom": {
      "version": "6.30.6",
      "resolved": "https://registry.npmjs.org/react-router-dom/-/react-router-dom-6.30.6.tgz",
      "integrity": "sha512-0RHKZz7wwffvkU+2MFVT2NnjK44ssLEV+m0CAJaS2Ksmorrwj7WxH00jO0SOCW26/tINUnJHToXblDs33I38YQ==",
      "license": "MIT",
      "dependencies": {
        "@remix-run/router": "1.23.4",
        "react-router": "6.30.6"
      },
      "engines": {
        "node": ">=14.0.0"
      },
      "peerDependencies": {
        "react": ">=16.8",
        "react-dom": ">=16.8"
      }
    },
    "node_modules/react-smooth": {
      "version": "4.0.4",
      "resolved": "https://registry.npmjs.org/react-smooth/-/react-smooth-4.0.4.tgz",
      "integrity": "sha512-gnGKTpYwqL0Iii09gHobNolvX4Kiq4PKx6eWBCYYix+8cdw+cGo3do906l1NBPKkSWx1DghC1dlWG9L2uGd61Q==",
      "license": "MIT",
      "dependencies": {
        "fast-equals": "^5.0.1",
        "prop-types": "^15.8.1",
        "react-transition-group": "^4.4.5"
      },
      "peerDependencies": {
        "react": "^16.8.0 || ^17.0.0 || ^18.0.0 || ^19.0.0",
        "react-dom": "^16.8.0 || ^17.0.0 || ^18.0.0 || ^19.0.0"
      }
    },
    "node_modules/react-transition-group": {
      "version": "4.4.5",
      "resolved": "https://registry.npmjs.org/react-transition-group/-/react-transition-group-4.4.5.tgz",
      "integrity": "sha512-pZcd1MCJoiKiBR2NRxeCRg13uCXbydPnmB4EOeRrY7480qNWO8IIgQG6zlDkm6uRMsURXPuKq0GWtiM59a5Q6g==",
      "license": "BSD-3-Clause",
      "dependencies": {
        "@babel/runtime": "^7.5.5",
        "dom-helpers": "^5.0.1",
        "loose-envify": "^1.4.0",
        "prop-types": "^15.6.2"
      },
      "peerDependencies": {
        "react": ">=16.6.0",
        "react-dom": ">=16.6.0"
      }
    },
    "node_modules/recharts": {
      "version": "2.15.4",
      "resolved": "https://registry.npmjs.org/recharts/-/recharts-2.15.4.tgz",
      "integrity": "sha512-UT/q6fwS3c1dHbXv2uFgYJ9BMFHu3fwnd7AYZaEQhXuYQ4hgsxLvsUXzGdKeZrW5xopzDCvuA2N41WJ88I7zIw==",
      "deprecated": "1.x and 2.x branches are no longer active. Bump to Recharts v3 to receive latest features and bugfixes. See https://github.com/recharts/recharts/wiki/3.0-migration-guide",
      "license": "MIT",
      "dependencies": {
        "clsx": "^2.0.0",
        "eventemitter3": "^4.0.1",
        "lodash": "^4.17.21",
        "react-is": "^18.3.1",
        "react-smooth": "^4.0.4",
        "recharts-scale": "^0.4.4",
        "tiny-invariant": "^1.3.1",
        "victory-vendor": "^36.6.8"
      },
      "engines": {
        "node": ">=14"
      },
      "peerDependencies": {
        "react": "^16.0.0 || ^17.0.0 || ^18.0.0 || ^19.0.0",
        "react-dom": "^16.0.0 || ^17.0.0 || ^18.0.0 || ^19.0.0"
      }
    },
    "node_modules/recharts-scale": {
      "version": "0.4.5",
      "resolved": "https://registry.npmjs.org/recharts-scale/-/recharts-scale-0.4.5.tgz",
      "integrity": "sha512-kivNFO+0OcUNu7jQquLXAxz1FIwZj8nrj+YkOKc5694NbjCvcT6aSZiIzNzd2Kul4o4rTto8QVR9lMNtxD4G1w==",
      "license": "MIT",
      "dependencies": {
        "decimal.js-light": "^2.4.1"
      }
    },
    "node_modules/rollup": {
      "version": "4.65.1",
      "resolved": "https://registry.npmjs.org/rollup/-/rollup-4.65.1.tgz",
      "integrity": "sha512-HzD4IdphaPMvuAacnL/vtoq5B8FwqewKvxTtqYibWvP6VDyZ+UsTtlw5F8HIOIIpLQAkN2NOP/0sM3puo9xN/A==",
      "dev": true,
      "license": "MIT",
      "dependencies": {
        "@types/estree": "1.0.9"
      },
      "bin": {
        "rollup": "dist/bin/rollup"
      },
      "engines": {
        "node": ">=18.0.0",
        "npm": ">=8.0.0"
      },
      "optionalDependencies": {
        "@napi-rs/lzma-linux-x64-gnu": "1.5.1",
        "@rollup/rollup-android-arm-eabi": "4.65.1",
        "@rollup/rollup-android-arm64": "4.65.1",
        "@rollup/rollup-darwin-arm64": "4.65.1",
        "@rollup/rollup-darwin-x64": "4.65.1",
        "@rollup/rollup-freebsd-arm64": "4.65.1",
        "@rollup/rollup-freebsd-x64": "4.65.1",
        "@rollup/rollup-linux-arm-gnueabihf": "4.65.1",
        "@rollup/rollup-linux-arm-musleabihf": "4.65.1",
        "@rollup/rollup-linux-arm64-gnu": "4.65.1",
        "@rollup/rollup-linux-arm64-musl": "4.65.1",
        "@rollup/rollup-linux-loong64-gnu": "4.65.1",
        "@rollup/rollup-linux-loong64-musl": "4.65.1",
        "@rollup/rollup-linux-ppc64-gnu": "4.65.1",
        "@rollup/rollup-linux-ppc64-musl": "4.65.1",
        "@rollup/rollup-linux-riscv64-gnu": "4.65.1",
        "@rollup/rollup-linux-riscv64-musl": "4.65.1",
        "@rollup/rollup-linux-s390x-gnu": "4.65.1",
        "@rollup/rollup-linux-x64-gnu": "4.65.1",
        "@rollup/rollup-linux-x64-musl": "4.65.1",
        "@rollup/rollup-openbsd-x64": "4.65.1",
        "@rollup/rollup-openharmony-arm64": "4.65.1",
        "@rollup/rollup-win32-arm64-msvc": "4.65.1",
        "@rollup/rollup-win32-ia32-msvc": "4.65.1",
        "@rollup/rollup-win32-x64-gnu": "4.65.1",
        "@rollup/rollup-win32-x64-msvc": "4.65.1",
        "fsevents": "~2.3.2"
      }
    },
    "node_modules/scheduler": {
      "version": "0.23.2",
      "resolved": "https://registry.npmjs.org/scheduler/-/scheduler-0.23.2.tgz",
      "integrity": "sha512-UOShsPwz7NrMUqhR6t0hWjFduvOzbtv7toDH1/hIrfRNIDBnnBWd0CwJTGvTpngVlmwGCdP9/Zl/tVrDqcuYzQ==",
      "license": "MIT",
      "dependencies": {
        "loose-envify": "^1.1.0"
      }
    },
    "node_modules/semver": {
      "version": "6.3.1",
      "resolved": "https://registry.npmjs.org/semver/-/semver-6.3.1.tgz",
      "integrity": "sha512-BR7VvDCVHO+q2xBEWskxS6DJE1qRnb7DxzUrogb71CWoSficBxYsiAGd+Kl0mmq/MprG9yArRkyrQxTO6XjMzA==",
      "dev": true,
      "license": "ISC",
      "bin": {
        "semver": "bin/semver.js"
      }
    },
    "node_modules/source-map-js": {
      "version": "1.2.2",
      "resolved": "https://registry.npmjs.org/source-map-js/-/source-map-js-1.2.2.tgz",
      "integrity": "sha512-KGj/8Y43x35aZVDtt+J4mK1hoLGHULMYfSkODJNQjNDC3oW1PqPoxMwo0pLUsWM/UEGzON/NxeHywEfNXNP3Vw==",
      "dev": true,
      "license": "BSD-3-Clause",
      "engines": {
        "node": ">=0.10.0"
      }
    },
    "node_modules/tiny-invariant": {
      "version": "1.3.3",
      "resolved": "https://registry.npmjs.org/tiny-invariant/-/tiny-invariant-1.3.3.tgz",
      "integrity": "sha512-+FbBPE1o9QAYvviau/qC5SE3caw21q3xkvWKBtja5vgqOWIHHJ3ioaq1VPfn/Szqctz2bU/oYeKd9/z5BL+PVg==",
      "license": "MIT"
    },
    "node_modules/tinyglobby": {
      "version": "0.2.17",
      "resolved": "https://registry.npmjs.org/tinyglobby/-/tinyglobby-0.2.17.tgz",
      "integrity": "sha512-wXR/dYpcqKmfWpEdZjiKJOwCNFndD0DMnrW/cYjVGttEkBfVgcLFHoNrlj47mjOVic9yyNu65alsgF4NQyTa2g==",
      "dev": true,
      "license": "MIT",
      "dependencies": {
        "fdir": "^6.5.0",
        "picomatch": "^4.0.4"
      },
      "engines": {
        "node": ">=12.0.0"
      },
      "funding": {
        "url": "https://github.com/sponsors/SuperchupuDev"
      }
    },
    "node_modules/update-browserslist-db": {
      "version": "1.3.4",
      "resolved": "https://registry.npmjs.org/update-browserslist-db/-/update-browserslist-db-1.3.4.tgz",
      "integrity": "sha512-t8wonIOdQyOjtPd2VktlBu6QjTVTks6v8bSi7kd0r2Vll8TK1JYDZtRRdbmCHOAMphkZgrzK+L+vXK4pyZ8t4A==",
      "dev": true,
      "funding": [
        {
          "type": "opencollective",
          "url": "https://opencollective.com/browserslist"
        },
        {
          "type": "tidelift",
          "url": "https://tidelift.com/funding/github/npm/browserslist"
        },
        {
          "type": "github",
          "url": "https://github.com/sponsors/ai"
        }
      ],
      "license": "MIT",
      "dependencies": {
        "escalade": "^3.2.0",
        "picocolors": "^1.1.1"
      },
      "bin": {
        "update-browserslist-db": "cli.js"
      },
      "peerDependencies": {
        "browserslist": ">= 4.21.0"
      }
    },
    "node_modules/victory-vendor": {
      "version": "36.9.2",
      "resolved": "https://registry.npmjs.org/victory-vendor/-/victory-vendor-36.9.2.tgz",
      "integrity": "sha512-PnpQQMuxlwYdocC8fIJqVXvkeViHYzotI+NJrCuav0ZYFoq912ZHBk3mCeuj+5/VpodOjPe1z0Fk2ihgzlXqjQ==",
      "license": "MIT AND ISC",
      "dependencies": {
        "@types/d3-array": "^3.0.3",
        "@types/d3-ease": "^3.0.0",
        "@types/d3-interpolate": "^3.0.1",
        "@types/d3-scale": "^4.0.2",
        "@types/d3-shape": "^3.1.0",
        "@types/d3-time": "^3.0.0",
        "@types/d3-timer": "^3.0.0",
        "d3-array": "^3.1.6",
        "d3-ease": "^3.0.1",
        "d3-interpolate": "^3.0.1",
        "d3-scale": "^4.0.2",
        "d3-shape": "^3.1.0",
        "d3-time": "^3.0.0",
        "d3-timer": "^3.0.1"
      }
    },
    "node_modules/vite": {
      "version": "6.4.5",
      "resolved": "https://registry.npmjs.org/vite/-/vite-6.4.5.tgz",
      "integrity": "sha512-4enwjFfi4IwtEh0w+jPzpLVOFQiPHmmzPxcHXQOyEur47o6u0jE573DmGpjA+7irXoMLYRZD5ZaFUEneS3abPg==",
      "dev": true,
      "license": "MIT",
      "dependencies": {
        "esbuild": "^0.25.0",
        "fdir": "^6.4.4",
        "picomatch": "^4.0.2",
        "postcss": "^8.5.3",
        "rollup": "^4.34.9",
        "tinyglobby": "^0.2.13"
      },
      "bin": {
        "vite": "bin/vite.js"
      },
      "engines": {
        "node": "^18.0.0 || ^20.0.0 || >=22.0.0"
      },
      "funding": {
        "url": "https://github.com/vitejs/vite?sponsor=1"
      },
      "optionalDependencies": {
        "fsevents": "~2.3.3"
      },
      "peerDependencies": {
        "@types/node": "^18.0.0 || ^20.0.0 || >=22.0.0",
        "jiti": ">=1.21.0",
        "less": "*",
        "lightningcss": "^1.21.0",
        "sass": "*",
        "sass-embedded": "*",
        "stylus": "*",
        "sugarss": "*",
        "terser": "^5.16.0",
        "tsx": "^4.8.1",
        "yaml": "^2.4.2"
      },
      "peerDependenciesMeta": {
        "@types/node": {
          "optional": true
        },
        "jiti": {
          "optional": true
        },
        "less": {
          "optional": true
        },
        "lightningcss": {
          "optional": true
        },
        "sass": {
          "optional": true
        },
        "sass-embedded": {
          "optional": true
        },
        "stylus": {
          "optional": true
        },
        "sugarss": {
          "optional": true
        },
        "terser": {
          "optional": true
        },
        "tsx": {
          "optional": true
        },
        "yaml": {
          "optional": true
        }
      }
    },
    "node_modules/yallist": {
      "version": "3.1.1",
      "resolved": "https://registry.npmjs.org/yallist/-/yallist-3.1.1.tgz",
      "integrity": "sha512-a4UGQaWPH59mOXUYnAG2ewncQS4i4F43Tv3JoAM+s2VDAmS9NsK8GpDMLrCHPksFT7h3K6TOoUNn2pb7RoXx4g==",
      "dev": true,
      "license": "ISC"
    }
  }
}
```


## File: `frontend/vite.config.js`

```javascript
import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
export default defineConfig({ plugins:[react()], server:{port:5173} })
```


## File: `frontend/src/App.jsx`

```jsx
import React, {createContext, useContext, useEffect, useState} from 'react'
import {Routes,Route,Navigate,NavLink,useNavigate,useLocation} from 'react-router-dom'
import {LayoutDashboard,Boxes,Wrench,ReceiptText,LogOut,Search,Plus,AlertTriangle,CheckCircle2,Clock3,Menu,PackagePlus,CalendarDays,Cog,RotateCw} from 'lucide-react'
import {ResponsiveContainer,AreaChart,Area,BarChart,Bar,XAxis,YAxis,Tooltip,CartesianGrid} from 'recharts'
import api from './api'

const AuthContext=createContext(null)
const money=n=>new Intl.NumberFormat('id-ID',{style:'currency',currency:'IDR',maximumFractionDigits:0}).format(n||0)
const todayISO=()=>{const d=new Date();return new Date(d.getTime()-d.getTimezoneOffset()*60000).toISOString().slice(0,10)}
const movementTypes={restock:'Barang masuk',sale:'Penjualan',service:'Service',correction:'Koreksi'}

const ToastContext=createContext(()=>{})
const toastIcons={success:<CheckCircle2 size={16}/>,error:<AlertTriangle size={16}/>,info:<Clock3 size={16}/>}
function ToastProvider({children}) {
 const [items,setItems]=useState([])
 const push=(message,type='success')=>{
  const id=`${Date.now()}-${Math.random()}`
  setItems(list=>[...list,{id,message,type}])
  setTimeout(()=>setItems(list=>list.filter(t=>t.id!==id)),3400)
 }
 return <ToastContext.Provider value={push}>
  {children}
  <div className="toast-stack" aria-live="polite">{items.map(t=><div key={t.id} className={`toast ${t.type}`}>{toastIcons[t.type]||toastIcons.info}<span>{t.message}</span></div>)}</div>
 </ToastContext.Provider>
}
const useToast=()=>useContext(ToastContext)

function useCountUp(value) {
 const [current,setCurrent]=useState(0)
 useEffect(()=>{
  if(typeof value!=='number'||!isFinite(value)){setCurrent(value);return}
  let frame
  const start=performance.now()
  const tick=now=>{
   const p=Math.min(1,(now-start)/700)
   setCurrent(Math.round(value*(1-Math.pow(1-p,3))))
   if(p<1)frame=requestAnimationFrame(tick)
  }
  frame=requestAnimationFrame(tick)
  return()=>cancelAnimationFrame(frame)
 },[value])
 return current
}
function CountUp({value,format}) {
 const current=useCountUp(typeof value==='number'?value:0)
 if(typeof value!=='number')return <>{value??'—'}</>
 return <>{format?format(current):current}</>
}

function AuthProvider({children}) {
 const [user,setUser]=useState(()=>{try{return JSON.parse(localStorage.getItem('garage_user'))}catch{return null}})
 const login=async(email,password)=>{const {data}=await api.post('/auth/login',{email,password});localStorage.setItem('garage_token',data.token);localStorage.setItem('garage_user',JSON.stringify(data.user));setUser(data.user)}
 const register=async(name,email,password)=>{await api.post('/auth/register',{name,email,password})}
 const logout=async()=>{try{await api.post('/auth/logout')}catch{}localStorage.removeItem('garage_token');localStorage.removeItem('garage_user');setUser(null)}
 return <AuthContext.Provider value={{user,login,register,logout}}>{children}</AuthContext.Provider>
}
const useAuth=()=>useContext(AuthContext)

function Login(){
 const {login,register}=useAuth()
 const [mode,setMode]=useState('login')
 const [name,setName]=useState('')
 const [email,setEmail]=useState('manager@garage.local')
 const [password,setPassword]=useState('Garage123!')
 const [error,setError]=useState('')
 const [busy,setBusy]=useState(false)
 const nav=useNavigate()
 async function submit(e){
  e.preventDefault();setError('');setBusy(true)
  try{
   if(mode==='register'){await register(name,email,password);await login(email,password)}else await login(email,password)
   nav('/')
  }catch(err){setError(err.response?.data?.error||'Tidak dapat terhubung ke server')}finally{setBusy(false)}
 }
 return <div className="login-page">
  <div className="login-art">
   <Cog className="login-gear g1" size={280} strokeWidth={1}/>
   <Cog className="login-gear g2" size={150} strokeWidth={1}/>
   <div className="login-tach" aria-hidden="true"/>
   <i className="login-needle" aria-hidden="true"/>
   <div className="brand-mark">G<span>F</span></div>
   <p className="eyebrow">WORKSHOP OPERATIONS</p>
   <h1>Semua sparepart.<br/>Satu kendali.</h1>
   <p className="muted-light">Inventory, penjualan, dan progress service dalam satu workspace.</p>
   <div className="login-metric"><span>GARAGEFLOW</span><span>BUILT FOR YOUR WORKSHOP ↗</span></div>
  </div>
  <form className="login-card" onSubmit={submit}>
   <div className="brand-mark small">G<span>F</span></div>
   <p className="eyebrow">{mode==='login'?'SELAMAT DATANG KEMBALI':'BUAT AKUN BARU'}</p>
   <h2>{mode==='login'?'Masuk ke workspace':'Daftar akun'}</h2>
   <p className="subtle">Kelola operasional bengkel tanpa kehilangan jejak.</p>
   {mode==='register'&&<label>Nama lengkap<input required value={name} onChange={e=>setName(e.target.value)} placeholder="Nama Anda"/></label>}
   <label>Email<input type="email" required value={email} onChange={e=>setEmail(e.target.value)} placeholder="nama@bengkel.com"/></label>
   <label>Password<input type="password" minLength="8" required value={password} onChange={e=>setPassword(e.target.value)}/></label>
   {error&&<div className="error">{error}</div>}
   <button className="primary full" disabled={busy}>{busy?'Memproses…':mode==='login'?'Masuk →':'Buat akun →'}</button>
   <p className="switch">{mode==='login'?'Belum punya akun?':'Sudah punya akun?'} <button type="button" className="text-button" onClick={()=>{setMode(mode==='login'?'register':'login');setError('')}}>{mode==='login'?'Daftar':'Masuk'}</button></p>
   {mode==='login'&&<div className="demo-hint">Demo: manager@garage.local / Garage123!</div>}
  </form>
 </div>
}

function Shell({children}){
 const {user,logout}=useAuth()
 const loc=useLocation()
 const [mobile,setMobile]=useState(false)
 return <div className="app-shell">
  <aside className={`sidebar ${mobile?'open':''}`}>
   <Cog className="side-gear" size={340} strokeWidth={0.7} aria-hidden="true"/>
   <div className="brand"><div className="brand-mark">G<span>F</span></div><div><b>garageflow</b><small>WORKSHOP OS</small></div></div>
   <div className="nav-caption">WORKSPACE</div>
   <nav>
    <NavLink to="/" end><LayoutDashboard size={18}/> Overview</NavLink>
    <NavLink to="/parts"><Boxes size={18}/> Inventory</NavLink>
    <NavLink to="/services"><Wrench size={18}/> Service board</NavLink>
    <NavLink to="/sales"><ReceiptText size={18}/> Penjualan</NavLink>
   </nav>
   <div className="sidebar-bottom">
    <div className="online-dot"/> Sistem operasional <span className="online-label">ONLINE</span>
    <div className="profile"><div className="avatar">{user?.name?.slice(0,1)||'U'}</div><div className="profile-copy"><b>{user?.name}</b><small>{user?.role==='manager'?'Pengelola':'Montir'}</small></div><button title="Logout" className="icon-btn" onClick={logout}><LogOut size={17}/></button></div>
   </div>
  </aside>
  <main className="main">
   <header className="topbar">
    <button className="icon-btn mobile-menu" onClick={()=>setMobile(!mobile)}><Menu/></button>
    <div className="breadcrumb">Workspace <span>/</span> <b>Garage management</b></div>
    <div className="top-right">
     <span className="engine-chip"><Cog size={13}/> GARAGE OPEN</span>
     <span className="today">{new Date().toLocaleDateString('id-ID',{weekday:'short',day:'2-digit',month:'short',year:'numeric'})}</span>
     <div className="avatar">{user?.name?.slice(0,1)||'U'}</div>
    </div>
   </header>
   <div className="content" key={loc.pathname}>{children}</div>
  </main>
 </div>
}

function PageTitle({eyebrow,title,desc,action,no}){
 return <div className="page-title">
  <span className="page-no" aria-hidden="true">{no}</span>
  <div><p className="eyebrow">{eyebrow}</p><h1>{title}</h1><p className="subtle">{desc}</p></div>
  {action}
 </div>
}

function RefreshButton({busy,onClick}){
 return <button className="ghost-btn refresh" onClick={onClick} disabled={busy} title="Muat ulang data"><RotateCw size={15} className={busy?'spin-icon':''}/> Refresh</button>
}

function Dashboard(){
 const [data,setData]=useState(null)
 const [err,setErr]=useState('')
 const [busy,setBusy]=useState(false)
 const load=()=>{setBusy(true);api.get('/dashboard').then(r=>{setData(r.data);setErr('')}).catch(e=>setErr(e.response?.data?.error||'Gagal memuat dashboard')).finally(()=>setBusy(false))}
 useEffect(()=>{load()},[])
 const stats=data?.stats||{}
 const nav=useNavigate()
 const dayName=new Date().toLocaleDateString('id-ID',{weekday:'long'}).toUpperCase()
 const low=(data?.low_stock_parts||[]).map(p=>`${p.name} · sisa ${p.stock} pcs`)
 const tickerItems=low.length?low:['Semua stok aman','Tidak ada item kritis hari ini','Katalog siap dijual']
 return <>
  <PageTitle eyebrow={`${dayName} · WORKSHOP PULSE`} title="Overview" no="01" desc="Ringkasan kondisi bengkel hari ini." action={<div className="title-actions"><span className="status-pill"><span/> Live workspace</span><RefreshButton busy={busy} onClick={load}/></div>}/>
  {err&&<div className="error">{err}</div>}
  <div className="ticker" title="Klik untuk membuka inventory stok menipis" role="button" tabIndex={0} onClick={()=>nav('/parts?low_stock=true')} onKeyDown={e=>{if(e.key==='Enter')nav('/parts?low_stock=true')}}>
   <span className="ticker-label"><AlertTriangle size={13}/> HEADS UP</span>
   <div className="ticker-window"><div className="ticker-track">
    {[0,1].map(d=><div className="ticker-group" key={d} aria-hidden={d===1}>
     {tickerItems.map((t,i)=><span key={i}><Boxes size={13}/>{t}</span>)}
    </div>)}
   </div></div>
  </div>
  <div className="stat-grid">
   <Stat label="Total sparepart" to="/parts" value={data?stats.parts:undefined} detail="Item terdaftar" icon={<Boxes/>}/>
   <Stat label="Perlu restock" to="/parts?low_stock=true" value={data?stats.low_stock:undefined} detail="Stok ≤ batas minimum" icon={<AlertTriangle/>} warn/>
   <Stat label="Service berjalan" to="/services" value={data?stats.active_services:undefined} detail="Job belum selesai" icon={<Wrench/>}/>
   <Stat label="Selesai hari ini" to="/services" value={data?stats.today_completed:undefined} detail="Customer selesai" icon={<CheckCircle2/>}/>
   <Stat label="Omzet hari ini" to="/sales" value={data?stats.today_revenue:undefined} detail="Dari transaksi tercatat" icon={<ReceiptText/>} format={money}/>
  </div>
  <div className="dashboard-grid">
   <section className="panel chart-panel">
    <div className="panel-head"><div><h3>Trend penjualan</h3><p>Omzet harian · 7 hari terakhir</p></div><span className="tag">7 HARI</span></div>
    <div className="chart">{data?<ResponsiveContainer width="100%" height="100%">
     <AreaChart data={data.sales_chart||[]} margin={{top:10,right:12,left:0,bottom:0}}>
      <defs>
       <linearGradient id="gfRevenue" x1="0" y1="0" x2="0" y2="1">
        <stop offset="0%" stopColor="#e07b39" stopOpacity={0.6}/>
        <stop offset="100%" stopColor="#e07b39" stopOpacity={0.03}/>
       </linearGradient>
      </defs>
      <CartesianGrid strokeDasharray="3 3" vertical={false} stroke="#e9edf2"/>
      <XAxis dataKey="day" tickFormatter={v=>v.slice(5)} axisLine={false} tickLine={false} tick={{fill:'#87909d',fontSize:12}}/>
      <YAxis axisLine={false} tickLine={false} tick={{fill:'#87909d',fontSize:11}} tickFormatter={v=>v>=1000000?`${(v/1000000).toFixed(1)}jt`:v>=1000?`${Math.round(v/1000)}k`:v}/>
      <Tooltip formatter={v=>money(v)} labelFormatter={v=>`Tanggal ${v}`}/>
      <Area type="monotone" dataKey="total" name="Omzet" stroke="#e07b39" strokeWidth={3} fill="url(#gfRevenue)" dot={{r:3,fill:'#e07b39',strokeWidth:0}} activeDot={{r:5}}/>
     </AreaChart>
    </ResponsiveContainer>:<div className="chart-skeleton skeleton"/>}</div>
    <div className="chart-foot"><span><i className="legend-dot"/> Omzet penjualan</span><span>Data transaksi tersimpan</span></div>
   </section>
   <section className="panel">
    <div className="panel-head"><div><h3>Perhatian stok</h3><p>Item yang perlu dicek</p></div><NavLink className="link" to="/parts?low_stock=true">Lihat semua ↗</NavLink></div>
    <div className="stock-list">
     {(data?.low_stock_parts||[]).map(p=><div className="stock-row" key={p.id}>
      <div className="part-icon"><Boxes size={17}/></div>
      <div className="stock-copy"><b>{p.name}</b><small>{p.sku} · {p.location}</small></div>
      <span className={`stock-count ${p.stock===0?'zero':''}`}>{p.stock} <small>pcs</small></span>
     </div>)}
     {!data&&[1,2,3,4].map(i=><div className="stock-row" key={`s${i}`}><div className="part-icon skeleton"/><div className="stock-copy"><span className="skeleton skeleton-line"/></div><span className="skeleton skeleton-pill"/></div>)}
     {data&&!data.low_stock_parts?.length&&<div className="empty"><CheckCircle2/> Semua stok aman</div>}
    </div>
   </section>
  </div>
  <section className="panel chart-panel completed-panel">
   <div className="panel-head"><div><h3>Customer selesai</h3><p>Jumlah service selesai per hari · 7 hari terakhir</p></div><span className="tag">7 HARI</span></div>
   <div className="chart bar-chart">{data?<ResponsiveContainer width="100%" height="100%">
    <BarChart data={data.completed_chart||[]} margin={{top:10,right:12,left:0,bottom:0}}>
     <defs>
      <linearGradient id="gfCompleted" x1="0" y1="0" x2="0" y2="1">
       <stop offset="0%" stopColor="#5fc497"/>
       <stop offset="100%" stopColor="#2f7d5d"/>
      </linearGradient>
     </defs>
     <CartesianGrid strokeDasharray="3 3" vertical={false} stroke="#e9edf2"/>
     <XAxis dataKey="day" tickFormatter={v=>v.slice(5)} axisLine={false} tickLine={false} tick={{fill:'#87909d',fontSize:12}}/>
     <YAxis allowDecimals={false} axisLine={false} tickLine={false} tick={{fill:'#87909d',fontSize:11}}/>
     <Tooltip formatter={v=>[`${v} customer`,'Selesai']} labelFormatter={v=>`Tanggal ${v}`}/>
     <Bar dataKey="total" name="Selesai" fill="#56a985" radius={[5,5,0,0]} maxBarSize={54}/>
    </BarChart>
   </ResponsiveContainer>:<div className="chart-skeleton skeleton"/>}</div>
   <div className="chart-foot"><span><i className="legend-dot green"/> Customer selesai</span><span>Hari ini: {stats.today_completed??0} customer</span></div>
  </section>
  <section className="panel service-preview">
   <div className="panel-head"><div><h3>Alur service</h3><p>Monitor pekerjaan yang sedang berlangsung</p></div><NavLink className="link" to="/services">Buka service board ↗</NavLink></div>
   <div className="workflow">
    <div><span className="workflow-number">01</span><b>Queued</b><small>Menunggu dikerjakan</small></div>
    <div className="workflow-line"/>
    <div><span className="workflow-number orange">02</span><b>In progress</b><small>Sedang dikerjakan</small></div>
    <div className="workflow-line"/>
    <div><span className="workflow-number green">03</span><b>Completed</b><small>Siap diserahkan</small></div>
   </div>
  </section>
 </>
}

function Stat({label,value,detail,icon,warn,format,to}){
 const nav=useNavigate()
 const click=to?{role:'button',tabIndex:0,onClick:()=>nav(to),onKeyDown:e=>{if(e.key==='Enter'||e.key===' '){e.preventDefault();nav(to)}}}:{}
 return <div className={`stat-card ${warn?'is-warning':''} ${to?'clickable':''}`} {...click}>
  <div className="stat-top"><span>{label}</span><span className={`stat-icon ${warn?'warning':''}`}>{icon}</span></div>
  <strong>{value===undefined||value===null?<span className="skeleton skeleton-line"/>:<CountUp value={value} format={format}/>}</strong>
  <small>{detail}</small>
  <span className="led-meter" aria-hidden="true"><i/><i/><i/><i/><i/></span>
 </div>
}

function Parts(){
 const {user}=useAuth()
 const toast=useToast()
 const [items,setItems]=useState([])
 const [movements,setMovements]=useState([])
 const [search,setSearch]=useState('')
 const [lowOnly,setLowOnly]=useState(new URLSearchParams(location.search).get('low_stock')==='true')
 const [show,setShow]=useState(false)
 const [restock,setRestock]=useState(null)
 const [restockForm,setRestockForm]=useState({quantity:1,note:'',moved_at:todayISO()})
 const [selling,setSelling]=useState(null)
 const [sellForm,setSellForm]=useState({quantity:1,sold_at:todayISO()})
 const [busy,setBusy]=useState(false)
 const [error,setError]=useState('')
 const [form,setForm]=useState({sku:'',name:'',category:'',brand:'',location:'',stock:0,min_stock:3,unit_price:0})
 const load=()=>{
  setBusy(true)
  Promise.all([
   api.get('/parts',{params:{search,low_stock:lowOnly?'true':undefined}}).then(r=>setItems(r.data.data)).catch(()=>setError('Gagal memuat inventory')),
   api.get('/stock-movements').then(r=>setMovements((r.data.data||[]).slice(0,8))).catch(()=>{})
  ]).finally(()=>setBusy(false))
 }
 useEffect(()=>{load()},[search,lowOnly])
 const canManage=user?.role==='manager'
 async function save(e){
  e.preventDefault()
  try{
   const {data}=await api.post('/parts',form)
   setShow(false);setForm({sku:'',name:'',category:'',brand:'',location:'',stock:0,min_stock:3,unit_price:0});load()
   toast(`${data.data.name} ditambahkan ke inventory`,'success')
  }catch(e){setError(e.response?.data?.error||'Gagal menyimpan sparepart')}
 }
 function openSell(p){setSelling(p);setSellForm({quantity:1,sold_at:todayISO()});setError('')}
 async function saveSell(e){
  e.preventDefault()
  try{
   await api.post('/sales',{part_id:selling.id,quantity:Number(sellForm.quantity),sold_at:sellForm.sold_at===todayISO()?'':sellForm.sold_at})
   setSelling(null);load();toast(`Penjualan ${selling.name} tersimpan`,'success')
  }catch(e){setError(e.response?.data?.error||'Transaksi gagal')}
 }
 function openRestock(p){setRestock(p);setRestockForm({quantity:1,note:'',moved_at:todayISO()});setError('')}
 async function saveRestock(e){
  e.preventDefault()
  try{
   await api.post(`/parts/${restock.id}/restock`,{...restockForm,quantity:Number(restockForm.quantity)})
   setRestock(null);load();toast(`Stok ${restock.name} bertambah ${restockForm.quantity} pcs`,'success')
  }catch(e){setError(e.response?.data?.error||'Gagal menambah stok')}
 }
 async function copySku(sku){
  let ok=false
  try{await navigator.clipboard.writeText(sku);ok=true}catch{}
  if(!ok){
   try{
    const ta=document.createElement('textarea');ta.value=sku;ta.setAttribute('readonly','');ta.style.position='fixed';ta.style.opacity='0'
    document.body.appendChild(ta);ta.select();ok=document.execCommand('copy');ta.remove()
   }catch{ok=false}
  }
  ok?toast(`SKU ${sku} disalin ke clipboard`,'info'):toast('Gagal menyalin SKU','error')
 }
 return <>
  <PageTitle eyebrow="STOCK CONTROL" title="Inventory sparepart" no="02" desc="Cari komponen, catat penjualan, dan tambah stok masuk." action={<div className="title-actions"><RefreshButton busy={busy} onClick={load}/>{canManage&&<button className="primary btn-new" onClick={()=>{setError('');setShow(true)}}><Plus size={17}/> Tambah sparepart</button>}</div>}/>
  {error&&<div className="error">{error}</div>}
  <section className="panel inventory-panel">
   <div className="toolbar">
    <div className="searchbox"><Search size={17}/><input value={search} onChange={e=>setSearch(e.target.value)} placeholder="Cari sparepart, SKU, lokasi…"/></div>
    <label className="check-filter"><input type="checkbox" checked={lowOnly} onChange={e=>setLowOnly(e.target.checked)}/> Stok menipis saja</label>
    <span className="result-count">{items.length} item</span>
   </div>
   <div className="table-wrap"><table className="parts-table">
    <thead><tr><th>SPAREPART</th><th>SKU</th><th>LOKASI GUDANG</th><th>STOK</th><th>HARGA</th><th>STATUS</th><th></th></tr></thead>
    <tbody>{items.map(p=><tr key={p.id}>
     <td><b>{p.name}</b><small className="cell-sub">{p.brand||p.category||'—'}</small></td>
     <td><button type="button" className="sku" title="Klik untuk salin SKU" onClick={()=>copySku(p.sku)}>{p.sku}</button></td>
     <td><span className="location-pill">⌖ {p.location}</span></td>
     <td><b>{p.stock}</b> <small>pcs</small><small className="cell-sub">Min. {p.min_stock}</small></td>
     <td>{money(p.unit_price)}</td>
     <td><span className={`badge ${p.stock<=p.min_stock?'low':'good'}`}>{p.stock===0?'Habis':p.stock<=p.min_stock?'Restock':'Tersedia'}</span></td>
     <td><div className="row-actions">
      <button className="small-action sell" onClick={()=>openSell(p)}><ReceiptText size={12}/> Jual</button>
      {canManage&&<button className="small-action restock" onClick={()=>openRestock(p)}><PackagePlus size={12}/> Restock</button>}
     </div></td>
    </tr>)}</tbody>
   </table>{!items.length&&<div className="empty">Tidak ada sparepart yang cocok.</div>}</div>
  </section>
  <section className="panel inventory-panel stock-history">
   <div className="panel-head"><div><h3>Pergerakan stok terakhir</h3><p>Barang masuk, penjualan, dan pemakaian service</p></div><span className="tag">{movements.length} TERBARU</span></div>
   <div className="table-wrap"><table>
    <thead><tr><th>TANGGAL</th><th>SPAREPART</th><th>TIPE</th><th>JUMLAH</th><th>CATATAN</th></tr></thead>
    <tbody>{movements.map(m=><tr key={m.id}>
     <td>{new Date(m.moved_at).toLocaleDateString('id-ID',{day:'2-digit',month:'short',year:'numeric'})}</td>
     <td><b>{m.part?.name||`Part #${m.part_id}`}</b></td>
     <td><span className={`type-pill ${m.quantity>0?'in':'out'}`}>{movementTypes[m.type]||m.type}</span></td>
     <td className={m.quantity>0?'qty-in':'qty-out'}>{m.quantity>0?'+':''}{m.quantity} pcs</td>
     <td>{m.note||'—'}</td>
    </tr>)}</tbody>
   </table>{!movements.length&&<div className="empty">Belum ada pergerakan stok.</div>}</div>
  </section>
  {show&&<div className="modal-backdrop"><form className="modal" onSubmit={save}>
   <div className="panel-head"><div><h3>Tambah sparepart</h3><p>Lokasi harus spesifik agar mudah ditemukan.</p></div><button type="button" className="icon-btn" onClick={()=>setShow(false)}>×</button></div>
   <div className="form-grid">
    {[['sku','SKU'],['name','Nama sparepart'],['category','Kategori'],['brand','Brand'],['location','Lokasi rak'],['stock','Stok awal'],['min_stock','Stok minimum'],['unit_price','Harga satuan (Rp)']].map(([key,label])=><label key={key}>{label}<input required={!['category','brand'].includes(key)} type={['stock','min_stock','unit_price'].includes(key)?'number':'text'} min={['stock','min_stock','unit_price'].includes(key)?'0':undefined} value={form[key]} onChange={e=>setForm({...form,[key]:e.target.type==='number'?Number(e.target.value):e.target.value})}/></label>)}
   </div>
   {error&&<div className="error">{error}</div>}
   <div className="modal-actions"><button type="button" className="secondary" onClick={()=>setShow(false)}>Batal</button><button className="primary btn-new">Simpan sparepart</button></div>
  </form></div>}
  {selling&&<div className="modal-backdrop"><form className="modal" onSubmit={saveSell}>
   <div className="panel-head"><div><h3>Catat penjualan</h3><p>{selling.name} · stok tersedia {selling.stock} pcs</p></div><button type="button" className="icon-btn" onClick={()=>setSelling(null)}>×</button></div>
   <div className="sell-summary"><span>Harga satuan</span><b>{money(selling.unit_price)}</b></div>
   <div className="form-grid">
    <label>Jumlah<input type="number" min="1" max={selling.stock} required value={sellForm.quantity} onChange={e=>setSellForm({...sellForm,quantity:e.target.value})}/></label>
    <label>Tanggal penjualan<input type="date" required value={sellForm.sold_at} onChange={e=>setSellForm({...sellForm,sold_at:e.target.value})}/></label>
   </div>
   <div className="picked-total sell-total"><span>Total</span><b>{money(selling.unit_price*Number(sellForm.quantity||0))}</b></div>
   {error&&<div className="error">{error}</div>}
   <div className="modal-actions"><button type="button" className="secondary" onClick={()=>setSelling(null)}>Batal</button><button className="primary btn-sell"><ReceiptText size={15}/> Simpan penjualan</button></div>
  </form></div>}
  {restock&&<div className="modal-backdrop"><form className="modal" onSubmit={saveRestock}>
   <div className="panel-head"><div><h3>Tambah stok</h3><p>{restock.name} · stok saat ini {restock.stock} pcs</p></div><button type="button" className="icon-btn" onClick={()=>setRestock(null)}>×</button></div>
   <div className="restock-badge"><PackagePlus size={18}/><div><b>{restock.sku}</b><small>{restock.location}</small></div><span>+ stok masuk</span></div>
   <div className="form-grid">
    <label>Jumlah masuk<input type="number" min="1" required value={restockForm.quantity} onChange={e=>setRestockForm({...restockForm,quantity:e.target.value})}/></label>
    <label>Tanggal masuk<input type="date" required value={restockForm.moved_at} onChange={e=>setRestockForm({...restockForm,moved_at:e.target.value})}/></label>
    <label className="full">Catatan<input value={restockForm.note} onChange={e=>setRestockForm({...restockForm,note:e.target.value})} placeholder="Contoh: pembelian dari supplier"/></label>
   </div>
   {error&&<div className="error">{error}</div>}
   <div className="modal-actions"><button type="button" className="secondary" onClick={()=>setRestock(null)}>Batal</button><button className="primary btn-restock"><PackagePlus size={15}/> Simpan stok</button></div>
  </form></div>}
 </>
}

const statusLabels={queued:'Menunggu',in_progress:'Dikerjakan',waiting_parts:'Menunggu sparepart',completed:'Selesai',cancelled:'Dibatalkan'}
const emptyService={customer_name:'',motorcycle:'',plate_number:'',complaint:'',mechanic_id:0,service_date:todayISO()}

function Services(){
 const toast=useToast()
 const [items,setItems]=useState([])
 const [catalog,setCatalog]=useState([])
 const [show,setShow]=useState(false)
 const [busy,setBusy]=useState(false)
 const [err,setErr]=useState('')
 const [form,setForm]=useState(emptyService)
 const [picked,setPicked]=useState([])
 const [pick,setPick]=useState({part_id:'',quantity:1})
 const load=()=>{
  setBusy(true)
  Promise.all([
   api.get('/services').then(r=>setItems(r.data.data)).catch(()=>setErr('Gagal memuat service')),
   api.get('/parts').then(r=>setCatalog(r.data.data)).catch(()=>{})
  ]).finally(()=>setBusy(false))
 }
 useEffect(()=>{load()},[])
 function openForm(){setForm(emptyService);setPicked([]);setPick({part_id:'',quantity:1});setErr('');setShow(true)}
 function addLine(){
  const id=Number(pick.part_id)
  const qty=Number(pick.quantity)
  const part=catalog.find(x=>x.id===id)
  if(!part||qty<1)return
  const existing=picked.find(x=>x.part_id===id)
  const totalQty=(existing?.quantity||0)+qty
  if(totalQty>part.stock){setErr(`Stok ${part.name} hanya ${part.stock} pcs`);return}
  setErr('')
  if(existing)setPicked(picked.map(x=>x.part_id===id?{...x,quantity:totalQty,subtotal:part.unit_price*totalQty}:x))
  else setPicked([...picked,{part_id:id,quantity:qty,name:part.name,unit_price:part.unit_price,subtotal:part.unit_price*qty}])
  setPick({part_id:'',quantity:1})
 }
 async function create(e){
  e.preventDefault()
  try{
   const {data}=await api.post('/services',{...form,mechanic_id:Number(form.mechanic_id)||0,parts:picked.map(({part_id,quantity})=>({part_id,quantity}))})
   setShow(false);setForm(emptyService);setPicked([]);load()
   toast(`Service ${data.data.work_order} dibuat`,'success')
  }catch(e){setErr(e.response?.data?.error||'Gagal membuat service')}
 }
 async function update(id,status){
  try{
   const {data}=await api.patch(`/services/${id}/status`,{status})
   load()
   toast(status==='completed'?`${data.data.work_order} selesai · sparepart masuk penjualan`:`${data.data.work_order} → ${statusLabels[status]}`,'success')
  }catch(e){setErr(e.response?.data?.error||'Gagal update status')}
 }
 const partsTotal=picked.reduce((a,x)=>a+x.subtotal,0)
 return <>
  <PageTitle eyebrow="WORKSHOP FLOOR" title="Service board" no="03" desc="Pantau kendaraan, keluhan, montir, dan progres pengerjaan." action={<div className="title-actions"><RefreshButton busy={busy} onClick={load}/><button className="primary btn-new" onClick={openForm}><Wrench size={17}/> Service baru</button></div>}/>
  {err&&<div className="error">{err}</div>}
  <div className="service-columns">
   {['queued','in_progress','waiting_parts','completed'].map(status=><section className="service-column" key={status}>
    <div className="column-heading"><span className={`column-dot ${status}`}/><b>{statusLabels[status]}</b><span className="column-count">{items.filter(x=>x.status===status).length}</span></div>
    {items.filter(x=>x.status===status).map(job=><article className="job-card" key={job.id}>
     <div className="job-top"><span>{job.work_order}</span><Clock3 size={15}/></div>
     <h3>{job.motorcycle}</h3>
     <p className="job-customer">{job.customer_name} · {job.plate_number||'Tanpa plat'}</p>
     <div className="job-complaint">{job.complaint||'Tidak ada catatan keluhan'}</div>
     {(job.parts?.length||0)>0&&<div className="job-parts">{(job.parts||[]).map(p=><span key={p.id}>{p.part?.name||`Part #${p.part_id}`} ×{p.quantity}</span>)}</div>}
     <div className="job-footer"><div className="avatar mini">{job.mechanic?.name?.slice(0,1)||'M'}</div><span>{job.mechanic?.name||'Belum ditugaskan'}</span></div>
     <select value={job.status} onChange={e=>update(job.id,e.target.value)} aria-label="Update status">{Object.entries(statusLabels).map(([v,l])=><option key={v} value={v}>{l}</option>)}</select>
    </article>)}
    {!items.some(x=>x.status===status)&&<div className="column-empty">Belum ada pekerjaan</div>}
   </section>)}
  </div>
  {show&&<div className="modal-backdrop"><form className="modal" onSubmit={create}>
   <div className="panel-head"><div><h3>Buat service order</h3><p>Catat kendaraan, keluhan, dan sparepart yang dibutuhkan.</p></div><button type="button" className="icon-btn" onClick={()=>setShow(false)}>×</button></div>
   <div className="form-grid">
    {[['customer_name','Nama pelanggan'],['motorcycle','Motor'],['plate_number','Nomor polisi'],['complaint','Keluhan']].map(([k,l])=><label key={k}>{l}<input required={k!=='plate_number'&&k!=='complaint'} value={form[k]} onChange={e=>setForm({...form,[k]:e.target.value})}/></label>)}
    <label>Tanggal service<input type="date" required value={form.service_date} onChange={e=>setForm({...form,service_date:e.target.value})}/></label>
   </div>
   <div className="parts-picker">
    <div className="parts-picker-head"><div><b>Sparepart untuk service</b><p>Pilih part yang dipakai. Otomatis tercatat sebagai penjualan saat status Selesai.</p></div><span className="tag">{picked.length} ITEM</span></div>
    <div className="parts-picker-row">
     <select value={pick.part_id} onChange={e=>setPick({...pick,part_id:e.target.value})}>
      <option value="">Pilih sparepart</option>
      {catalog.map(p=><option key={p.id} value={p.id}>{p.name} · stok {p.stock}</option>)}
     </select>
     <input type="number" min="1" value={pick.quantity} onChange={e=>setPick({...pick,quantity:e.target.value})}/>
     <button type="button" className="secondary btn-add" onClick={addLine}><Plus size={14}/> Tambah</button>
    </div>
    {picked.length>0&&<div className="picked-list">
     {picked.map(line=><div className="picked-row" key={line.part_id}>
      <b>{line.name}</b><span>{line.quantity} × {money(line.unit_price)}</span><strong>{money(line.subtotal)}</strong>
      <button type="button" className="icon-btn" onClick={()=>setPicked(picked.filter(x=>x.part_id!==line.part_id))}>×</button>
     </div>)}
     <div className="picked-total"><span>Total sparepart</span><b>{money(partsTotal)}</b></div>
    </div>}
    <div className="hint"><CalendarDays size={14}/> Penjualan sparepart dibuat otomatis ketika job diubah ke status <b>Selesai</b>. Jika dibatalkan, transaksi dibalik dan stok dikembalikan.</div>
   </div>
   {err&&<div className="error">{err}</div>}
   <div className="modal-actions"><button type="button" className="secondary" onClick={()=>setShow(false)}>Batal</button><button className="primary btn-new">Buat service</button></div>
  </form></div>}
 </>
}

function Sales(){
 const toast=useToast()
 const [rows,setRows]=useState([])
 const [parts,setParts]=useState([])
 const [part,setPart]=useState('')
 const [qty,setQty]=useState(1)
 const [date,setDate]=useState(todayISO())
 const [busy,setBusy]=useState(false)
 const [error,setError]=useState('')
 const load=()=>{
  setBusy(true)
  Promise.all([
   api.get('/sales').then(r=>setRows(r.data.data)).catch(()=>setError('Gagal memuat penjualan')),
   api.get('/parts').then(r=>setParts(r.data.data)).catch(()=>{})
  ]).finally(()=>setBusy(false))
 }
 useEffect(()=>{load()},[])
 const selected=parts.find(p=>p.id===Number(part))
 const total=selected?selected.unit_price*Number(qty||0):0
 async function submit(e){
  e.preventDefault()
  try{
   await api.post('/sales',{part_id:Number(part),quantity:Number(qty),sold_at:date===todayISO()?'':date})
   setError('');load();toast('Penjualan tersimpan','success')
  }catch(e){setError(e.response?.data?.error||'Transaksi gagal')}
 }
 return <>
  <PageTitle eyebrow="REVENUE & TRANSACTIONS" title="Penjualan" no="04" desc="Catat penjualan sparepart, termasuk transaksi dari service." action={<div className="title-actions"><RefreshButton busy={busy} onClick={load}/></div>}/>
  <section className="panel sale-form-panel">
   <div><h3>Transaksi baru</h3><p>Pilih sparepart, jumlah, dan tanggal transaksi.</p></div>
   <form className="sale-form" onSubmit={submit}>
    <select required value={part} onChange={e=>setPart(e.target.value)}><option value="">Pilih sparepart</option>{parts.map(p=><option key={p.id} value={p.id}>{p.name} · stok {p.stock}</option>)}</select>
    <input type="number" min="1" required value={qty} onChange={e=>setQty(e.target.value)}/>
    <input className="date-input" type="date" required value={date} onChange={e=>setDate(e.target.value)} title="Tanggal penjualan"/>
    <button className="primary btn-sell"><ReceiptText size={15}/> Simpan penjualan</button>
   </form>
   {selected&&<div className="sale-preview"><span>Total transaksi</span><b>{money(total)}</b></div>}
   {error&&<div className="error">{error}</div>}
  </section>
  <section className="panel inventory-panel">
   <div className="panel-head"><div><h3>Riwayat transaksi</h3><p>200 transaksi terakhir · manual dan service</p></div><span className="tag">{rows.length} TRANSAKSI</span></div>
   <div className="table-wrap"><table className="sales-table">
    <thead><tr><th>TANGGAL</th><th>SPAREPART</th><th>SUMBER</th><th>JUMLAH</th><th>HARGA SATUAN</th><th>TOTAL</th></tr></thead>
    <tbody>{rows.map(s=><tr key={s.id}>
     <td>{new Date(s.sold_at).toLocaleString('id-ID')}</td>
     <td><b>{s.part?.name||`Part #${s.part_id}`}</b></td>
     <td>{s.source==='service'?<><span className="badge service">Service</span>{s.work_order&&<small className="cell-sub">{s.work_order}</small>}</>:<span className="badge manual">Manual</span>}</td>
     <td>{s.quantity} pcs</td>
     <td>{money(s.unit_price)}</td>
     <td><b>{money(s.total)}</b></td>
    </tr>)}</tbody>
   </table>{!rows.length&&<div className="empty">Belum ada transaksi penjualan.</div>}</div>
  </section>
 </>
}

function Protected(){
 const {user}=useAuth()
 return user?<Shell><Routes><Route path="/" element={<Dashboard/>}/><Route path="/parts" element={<Parts/>}/><Route path="/services" element={<Services/>}/><Route path="/sales" element={<Sales/>}/><Route path="*" element={<Navigate to="/" replace/>}/></Routes></Shell>:<Navigate to="/login" replace/>
}
export default function App(){return <AuthProvider><ToastProvider><Routes><Route path="/login" element={<Login/>}/><Route path="/*" element={<Protected/>}/></Routes></ToastProvider></AuthProvider>}
```


## File: `frontend/src/api.js`

```javascript
import axios from 'axios'
const api = axios.create({ baseURL: import.meta.env.VITE_API_URL || 'http://localhost:8080/api/v1' })
api.interceptors.request.use(config => {
  const token = localStorage.getItem('garage_token')
  if (token) config.headers.Authorization = `Bearer ${token}`
  return config
})
api.interceptors.response.use(r=>r, err=>{
  if (err.response?.status===401) { localStorage.removeItem('garage_token'); localStorage.removeItem('garage_user') }
  return Promise.reject(err)
})
export default api
```


## File: `frontend/src/main.jsx`

```jsx
import React from 'react'
import { createRoot } from 'react-dom/client'
import { BrowserRouter } from 'react-router-dom'
import App from './App'
import './styles.css'
createRoot(document.getElementById('root')).render(<React.StrictMode><BrowserRouter><App/></BrowserRouter></React.StrictMode>)
```


## File: `frontend/src/styles.css`

```css
@import url('https://fonts.googleapis.com/css2?family=DM+Sans:wght@400;500;600;700&family=Manrope:wght@400;500;600;700;800&family=Chakra+Petch:wght@500;600;700&family=Share+Tech+Mono&display=swap');
:root{font-family:'DM Sans',sans-serif;color:#202832;background:#f6f7f9;font-synthesis:none;text-rendering:optimizeLegibility}*{box-sizing:border-box}body{margin:0;min-width:320px}button,input,select{font:inherit}button{cursor:pointer}.app-shell{min-height:100vh}.sidebar{position:fixed;inset:0 auto 0 0;width:244px;background:#171e26;color:#d7dce2;padding:27px 18px 16px;display:flex;flex-direction:column;z-index:5}.brand{display:flex;align-items:center;gap:11px;padding:0 8px 36px}.brand b{display:block;font-family:Manrope;font-size:17px;letter-spacing:-.7px;color:#fff}.brand small{font-size:9px;letter-spacing:2px;color:#7e8995}.brand-mark{width:38px;height:38px;background:#df7b3d;border-radius:11px;color:white;font-family:Manrope;font-weight:800;font-size:17px;display:grid;place-items:center;letter-spacing:-2px}.brand-mark span{color:#292d32}.brand-mark.small{width:44px;height:44px;margin-bottom:30px}.nav-caption,.eyebrow{font-size:10px;font-weight:700;letter-spacing:1.6px;color:#8d97a3}.nav-caption{padding:0 12px 13px}.sidebar nav{display:grid;gap:5px}.sidebar nav a{display:flex;align-items:center;gap:12px;color:#9ba5b1;text-decoration:none;padding:12px;border-radius:8px;font-size:13px}.sidebar nav a.active{background:#2b333d;color:white}.sidebar nav a.active svg{color:#f09a61}.sidebar-bottom{margin-top:auto;font-size:11px;color:#9ca6b2;padding:15px 7px 0;border-top:1px solid #303842}.online-dot,.status-pill span{display:inline-block;width:7px;height:7px;border-radius:50%;background:#50b58b;margin-right:7px}.online-label{float:right;color:#5bc096;font-size:9px;letter-spacing:1px}.profile{display:flex;align-items:center;gap:9px;margin-top:24px;padding-top:17px;border-top:1px solid #303842}.avatar{height:34px;width:34px;display:grid;place-items:center;border-radius:50%;background:#f4dfd0;color:#98572e;font-weight:700;font-size:13px;flex-shrink:0}.profile-copy{min-width:0;flex:1}.profile-copy b,.profile-copy small{display:block;overflow:hidden;text-overflow:ellipsis;white-space:nowrap}.profile-copy b{color:#f0f2f4;font-size:11px}.profile-copy small{font-size:10px;margin-top:4px}.icon-btn{border:0;background:transparent;color:inherit;display:grid;place-items:center;padding:6px;border-radius:6px}.icon-btn:hover{background:#e9edf2;color:#27303a}.sidebar .icon-btn:hover{background:#343e49;color:#fff}.main{margin-left:244px;min-height:100vh}.topbar{height:67px;background:#fff;border-bottom:1px solid #e9edf0;display:flex;align-items:center;justify-content:space-between;padding:0 38px}.breadcrumb{font-size:12px;color:#89929c}.breadcrumb span{padding:0 10px;color:#c4cbd1}.breadcrumb b{font-weight:600;color:#4a535e}.top-right{display:flex;align-items:center;gap:17px}.today{font-size:11px;color:#7d8792}.content{padding:33px 38px 48px;max-width:1600px;margin:auto}.page-title{display:flex;align-items:center;justify-content:space-between;gap:20px;margin-bottom:27px}.page-title .eyebrow{margin:0 0 9px;color:#d17b42}.page-title h1{font:800 29px Manrope,sans-serif;letter-spacing:-1.1px;margin:0 0 7px;color:#202832}.subtle{color:#8a939e;font-size:12px;margin:0;line-height:1.6}.status-pill{border:1px solid #dfeee6;border-radius:20px;color:#43896d;padding:8px 12px;font-size:11px;white-space:nowrap}.status-pill span{width:6px;height:6px}.primary{display:inline-flex;align-items:center;justify-content:center;gap:8px;background:#d9783d;color:white;border:1px solid #d9783d;border-radius:7px;padding:10px 15px;font-size:12px;font-weight:700;box-shadow:0 2px 4px #d9783d18}.primary:hover{background:#c86831}.primary:disabled{opacity:.6;cursor:wait}.secondary{border:1px solid #dfe3e8;border-radius:7px;padding:10px 15px;background:white;color:#4b5561;font-size:12px}.full{width:100%;padding:13px}.stat-grid{display:grid;grid-template-columns:repeat(4,minmax(0,1fr));gap:15px;margin-bottom:19px}.stat-card,.panel{background:#fff;border:1px solid #e9edf0;border-radius:11px;box-shadow:0 2px 5px #1b2a3805}.stat-card{padding:18px 19px;min-width:0}.stat-top{display:flex;align-items:center;justify-content:space-between;gap:6px;color:#7e8792;font-size:11px}.stat-icon{width:32px;height:32px;display:grid;place-items:center;border-radius:8px;background:#edf4f7;color:#557c8c}.stat-icon svg{width:17px}.stat-icon.warning{background:#fff2e8;color:#d9783d}.stat-card>strong{display:block;font:800 26px Manrope,sans-serif;letter-spacing:-.8px;margin:12px 0 5px;overflow-wrap:anywhere}.stat-card>small{color:#9aa2ab;font-size:10px}.dashboard-grid{display:grid;grid-template-columns:minmax(0,1.45fr) minmax(310px,1fr);gap:18px;margin-bottom:19px}.panel{padding:21px}.panel-head{display:flex;align-items:center;justify-content:space-between;gap:15px}.panel-head h3,.sale-form-panel h3{font:700 14px Manrope,sans-serif;margin:0 0 6px;letter-spacing:-.2px}.panel-head p,.sale-form-panel p{margin:0;color:#9aa2ab;font-size:11px}.tag{border:1px solid #e6e9ed;border-radius:5px;padding:6px 8px;color:#8a939d;font-size:9px;font-weight:700;letter-spacing:.7px;white-space:nowrap}.chart{height:235px;margin-top:24px}.chart-foot{border-top:1px solid #eff1f3;padding-top:13px;display:flex;justify-content:space-between;font-size:10px;color:#969fa9}.legend-dot{display:inline-block;width:7px;height:7px;background:#d9783d;border-radius:50%;margin-right:7px}.link{color:#cc713a;text-decoration:none;font-size:11px;font-weight:600;white-space:nowrap}.stock-list{margin-top:17px}.stock-row{display:flex;align-items:center;gap:10px;padding:12px 0;border-bottom:1px solid #f0f2f4}.stock-row:last-child{border:0}.part-icon{width:34px;height:34px;display:grid;place-items:center;background:#f4f5f7;color:#7d8792;border-radius:8px}.stock-copy{flex:1;min-width:0}.stock-copy b,.stock-copy small{display:block;overflow:hidden;text-overflow:ellipsis;white-space:nowrap}.stock-copy b{font-size:11px;font-weight:600}.stock-copy small{font-size:10px;color:#9ba3ac;margin-top:5px}.stock-count{font-weight:800;color:#c86b34;font-size:14px}.stock-count small{font-size:9px;font-weight:500;color:#a0a7af}.stock-count.zero{color:#c34e4e}.empty{padding:30px 12px;text-align:center;color:#9aa2ab;font-size:12px;display:flex;justify-content:center;align-items:center;gap:8px}.workflow{display:flex;align-items:center;gap:13px;margin-top:25px}.workflow>div:not(.workflow-line){display:grid;grid-template-columns:37px auto;align-items:center;column-gap:10px;min-width:0}.workflow-number{grid-row:span 2;width:34px;height:34px;border-radius:50%;background:#edf1f5;display:grid;place-items:center;color:#7b8792;font-size:11px;font-weight:700}.workflow-number.orange{background:#fff0e5;color:#d9783d}.workflow-number.green{background:#e4f5ed;color:#4a9b78}.workflow b{font-size:11px}.workflow small{font-size:10px;color:#9ba3ac;margin-top:3px}.workflow-line{height:1px;background:#e8ebee;flex:1;min-width:10px}.toolbar{display:flex;align-items:center;gap:18px;margin-bottom:20px}.searchbox{display:flex;align-items:center;gap:9px;border:1px solid #e5e9ed;border-radius:7px;padding:0 11px;height:37px;flex:1;max-width:420px;color:#8b95a0}.searchbox input{border:0;outline:0;width:100%;font-size:11px;color:#343d47}.check-filter{display:flex;align-items:center;gap:7px;color:#77818d;font-size:11px;white-space:nowrap}.check-filter input{accent-color:#d9783d}.result-count{margin-left:auto;color:#9aa2ab;font-size:11px;white-space:nowrap}.table-wrap{width:100%;overflow-x:auto}table{width:100%;border-collapse:collapse;text-align:left;white-space:nowrap}th{font-size:9px;letter-spacing:.9px;color:#9aa2ab;font-weight:700;padding:12px 13px;border-bottom:1px solid #edf0f2}td{padding:15px 13px;border-bottom:1px solid #f0f2f4;font-size:11px;color:#626d78}tbody tr:last-child td{border-bottom:0}td b{font-size:11px;color:#303944;font-weight:600}.cell-sub{display:block;margin-top:5px;color:#9aa2ab;font-size:10px}.sku{font-family:monospace;color:#66717d;background:#f6f7f8;padding:5px 6px;border-radius:4px;font-size:10px}.location-pill{background:#f4f6f8;border-radius:4px;padding:6px 8px;color:#697580;font-size:10px}.badge{display:inline-block;padding:5px 8px;border-radius:5px;font-size:9px;font-weight:700}.badge.low{background:#fff0e6;color:#c76d35}.badge.good{background:#e6f5ed;color:#438b69}.small-action{border:1px solid #e1e6ea;border-radius:5px;background:#fff;padding:6px 9px;font-size:10px;color:#737e89}.small-action:hover{border-color:#d9783d;color:#c56d37}.service-columns{display:grid;grid-template-columns:repeat(4,minmax(0,1fr));gap:13px;align-items:start}.service-column{background:#eef1f4;border-radius:10px;padding:12px;min-height:230px}.column-heading{display:flex;align-items:center;gap:8px;padding:5px 2px 14px;font-size:11px}.column-dot{width:8px;height:8px;border-radius:50%;background:#98a2ae}.column-dot.in_progress{background:#e18a4d}.column-dot.waiting_parts{background:#8c7bc4}.column-dot.completed{background:#56a985}.column-count{margin-left:auto;color:#9aa2ab;background:#fff;padding:3px 7px;border-radius:5px;font-size:10px}.job-card{background:white;border:1px solid #e5e9ed;border-radius:8px;padding:13px;margin-bottom:10px;box-shadow:0 2px 4px #1b2a3805}.job-top{display:flex;justify-content:space-between;color:#9ba3ac;font-size:9px;font-family:monospace}.job-card h3{font:700 13px Manrope;margin:12px 0 5px;letter-spacing:-.3px}.job-customer{font-size:10px;color:#8d97a1;margin:0 0 12px}.job-complaint{border-top:1px solid #f0f2f4;padding-top:10px;color:#7d8792;font-size:10px;line-height:1.6;min-height:38px}.job-footer{display:flex;align-items:center;gap:7px;margin:13px 0;font-size:10px;color:#6e7985}.avatar.mini{width:23px;height:23px;font-size:9px}.job-card select{width:100%;border:1px solid #e7eaed;border-radius:5px;padding:7px;font-size:10px;color:#68737e;background:#fafbfc}.column-empty{text-align:center;padding:26px 6px;color:#a2a9b1;font-size:10px}.sale-form-panel{margin-bottom:20px}.sale-form{display:flex;gap:10px;margin-top:17px}.sale-form select{flex:1}.sale-form select,.sale-form input{border:1px solid #e2e6ea;border-radius:6px;padding:10px;font-size:11px;min-width:0;background:white;color:#4d5863}.sale-form input{width:85px}.error{padding:10px 12px;background:#fff0f0;color:#b44a4a;border-radius:6px;font-size:11px;margin:12px 0}.modal-backdrop{position:fixed;inset:0;background:#141b23a8;z-index:10;display:grid;place-items:center;padding:20px}.modal{background:white;border-radius:12px;padding:24px;width:min(560px,100%);max-height:90vh;overflow:auto;box-shadow:0 20px 60px #0002}.modal .panel-head{margin-bottom:20px}.form-grid{display:grid;grid-template-columns:1fr 1fr;gap:14px}.form-grid label,.login-card label{display:grid;gap:7px;font-size:11px;color:#65707c;font-weight:600}.form-grid input,.login-card input{width:100%;border:1px solid #e1e6ea;border-radius:6px;padding:10px;font-size:12px;outline:none;color:#303943;background:#fff}.form-grid input:focus,.login-card input:focus{border-color:#d9783d}.modal-actions{display:flex;justify-content:flex-end;gap:9px;margin-top:23px}.login-page{min-height:100vh;display:grid;grid-template-columns:1.1fr .9fr;background:#fff}.login-art{background:#1a222b;color:white;padding:9vh 8vw;display:flex;flex-direction:column;justify-content:center;position:relative;overflow:hidden}.login-art:after{content:"";position:absolute;width:430px;height:430px;border:1px solid #ffffff13;border-radius:50%;right:-140px;bottom:-170px;box-shadow:0 0 0 50px #ffffff05,0 0 0 100px #ffffff04}.login-art .brand-mark{margin-bottom:52px}.login-art .eyebrow{color:#d98752}.login-art h1{font:800 clamp(34px,4vw,58px) Manrope;letter-spacing:-2px;line-height:1.12;margin:14px 0 18px}.muted-light{color:#aab4bf;font-size:13px;max-width:370px;line-height:1.8}.login-metric{display:flex;justify-content:space-between;gap:12px;position:absolute;bottom:30px;left:8vw;right:8vw;color:#687581;font-size:9px;letter-spacing:1px}.login-card{width:min(390px,85%);align-self:center;justify-self:center}.login-card .eyebrow{color:#d17b42;margin:0 0 9px}.login-card h2{font:800 27px Manrope;letter-spacing:-1px;margin:0 0 9px}.login-card>.subtle{margin-bottom:27px}.login-card label{margin-bottom:16px}.login-card input{padding:12px}.switch{text-align:center;color:#89929c;font-size:11px;margin-top:19px}.text-button{border:0;background:none;color:#cc713a;font-weight:700;font-size:11px;padding:0}.demo-hint{background:#f7f8f9;border-radius:6px;padding:10px;color:#8d96a0;font-size:10px;text-align:center}.mobile-menu{display:none}.login-card .error{margin-top:0}
@media(max-width:1150px){.content{padding:28px 24px}.topbar{padding:0 24px}.sidebar{width:220px}.main{margin-left:220px}.stat-grid{grid-template-columns:repeat(2,minmax(0,1fr))}.service-columns{grid-template-columns:repeat(2,minmax(0,1fr))}.dashboard-grid{grid-template-columns:1fr}.chart{height:230px}}
@media(max-width:700px){.sidebar{transform:translateX(-100%);transition:transform .2s}.sidebar.open{transform:translateX(0)}.main{margin-left:0}.topbar{height:58px;padding:0 15px}.mobile-menu{display:grid;margin-right:8px}.breadcrumb{margin-right:auto;font-size:10px}.topbar{justify-content:flex-start}.top-right{margin-left:auto;gap:8px}.today{display:none}.content{padding:25px 14px}.page-title{align-items:flex-start;flex-direction:column;margin-bottom:20px}.page-title h1{font-size:25px}.stat-grid{gap:9px}.stat-card{padding:14px}.stat-card>strong{font-size:22px}.stat-top{font-size:10px}.dashboard-grid{gap:12px}.panel{padding:15px}.workflow{gap:6px;align-items:flex-start}.workflow>div:not(.workflow-line){grid-template-columns:28px auto;gap:6px}.workflow-number{width:27px;height:27px;font-size:9px}.workflow b{font-size:10px}.workflow small{font-size:9px}.workflow-line{margin-top:13px}.toolbar{flex-wrap:wrap;gap:12px}.searchbox{max-width:none;flex-basis:100%}.result-count{margin-left:auto}.service-columns{grid-template-columns:1fr}.service-column{min-height:unset}.sale-form{flex-wrap:wrap}.sale-form select{flex-basis:100%}.form-grid{grid-template-columns:1fr}.login-page{grid-template-columns:1fr}.login-art{display:none}.login-card{width:min(390px,88%);padding:45px 0}.login-card .brand-mark.small{margin-bottom:35px}}

@media(min-width:1151px){.stat-grid{grid-template-columns:repeat(auto-fit,minmax(185px,1fr))}}
.completed-panel{margin-bottom:19px}
.chart.bar-chart{height:200px}
.legend-dot.green{background:#56a985}
.badge.service{background:#e8eefc;color:#4f6bb0}
.badge.manual{background:#f2f3f5;color:#6b7480}
.row-actions{display:flex;gap:6px}
.row-actions .small-action{display:inline-flex;align-items:center;gap:4px;white-space:nowrap}
.stock-history{margin-top:19px}
.type-pill{display:inline-block;padding:5px 8px;border-radius:5px;font-size:9px;font-weight:700;background:#eef1f4;color:#68737e}
.type-pill.in{background:#e6f5ed;color:#438b69}
.type-pill.out{background:#fff0e6;color:#c76d35}
.qty-in{color:#438b69;font-weight:700}
.qty-out{color:#c76d35;font-weight:700}
.restock-badge{display:flex;align-items:center;gap:10px;background:#f7f8f9;border-radius:7px;padding:11px 13px;margin-bottom:16px;color:#7d8792}
.restock-badge svg{color:#d9783d}
.restock-badge div{flex:1;min-width:0}
.restock-badge b,.restock-badge small{display:block}
.restock-badge b{font-family:monospace;font-size:10px;color:#4d5863}
.restock-badge small{font-size:10px;color:#9aa2ab;margin-top:4px}
.restock-badge>span{font-size:10px;color:#438b69;font-weight:700;white-space:nowrap}
.form-grid label.full{grid-column:1/-1}
.parts-picker{border:1px solid #e9edf0;border-radius:8px;padding:14px;margin-top:16px}
.parts-picker-head{display:flex;align-items:flex-start;justify-content:space-between;gap:10px}
.parts-picker-head b{font-size:11px;color:#4d5863}
.parts-picker-head p{margin:4px 0 0;color:#9aa2ab;font-size:10px;line-height:1.5;max-width:360px}
.parts-picker-row{display:flex;gap:8px;margin-top:12px}
.parts-picker-row select{flex:1;min-width:0}
.parts-picker-row input{width:72px;flex-shrink:0}
.parts-picker-row select,.parts-picker-row input{border:1px solid #e2e6ea;border-radius:6px;padding:9px;font-size:11px;background:#fff;color:#4d5863}
.parts-picker-row .secondary{display:inline-flex;align-items:center;gap:4px;padding:8px 11px;white-space:nowrap}
.picked-list{display:grid;gap:6px;margin-top:11px}
.picked-row{display:flex;align-items:center;gap:9px;background:#f7f8f9;border-radius:6px;padding:7px 9px;font-size:10px;color:#77818d}
.picked-row b{flex:1;min-width:0;overflow:hidden;text-overflow:ellipsis;white-space:nowrap;font-size:10px;color:#3d4752}
.picked-row strong{color:#3d4752}
.picked-total{display:flex;justify-content:space-between;border-top:1px solid #e9edf0;padding:9px 3px 0;font-size:11px;color:#77818d}
.picked-total b{color:#303944}
.hint{display:flex;gap:7px;align-items:flex-start;background:#fbf5ef;border-radius:6px;padding:9px 10px;color:#9a7a62;font-size:10px;line-height:1.55;margin-top:12px}
.hint svg{flex-shrink:0;margin-top:1px;color:#d9783d}
.job-parts{display:flex;flex-wrap:wrap;gap:5px;margin-top:10px}
.job-parts span{background:#f2f5f8;border-radius:4px;padding:4px 6px;font-size:9px;color:#68737e;max-width:100%;overflow:hidden;text-overflow:ellipsis;white-space:nowrap}
.sale-form .date-input{width:150px}
@media(max-width:700px){.sale-form .date-input{width:100%}.parts-picker-row{flex-wrap:wrap}.parts-picker-row select{flex-basis:100%}.row-actions{flex-wrap:wrap}}

/* ============================================================
   GARAGEFLOW · WORKSHOP THEME LAYER
   Carbon texture, hazard accents, animated icons, garage UI.
   ============================================================ */
:root{
  --gf-ink:#1b222b;
  --gf-steel:#2a323c;
  --gf-orange:#e07b39;
  --gf-orange-dark:#c4632a;
  --gf-orange-soft:#fff3ea;
  --gf-green:#4a9b78;
  --gf-line:#e4e8ec;
  --gf-shadow:0 14px 32px #1b2a3812;
}

body{
  background-color:#f1f3f5;
  background-image:
    radial-gradient(1100px 520px at 88% -8%, #ffd9bd4d 0%, transparent 58%),
    radial-gradient(900px 520px at -8% 108%, #cfe4ee4d 0%, transparent 60%),
    repeating-linear-gradient(135deg, #171e2604 0 2px, transparent 2px 9px);
  background-attachment:fixed;
}

/* ---------- Sidebar ---------- */
.sidebar{
  background:linear-gradient(180deg,#131920 0%,#1b232c 55%,#141a21 100%);
  border-right:1px solid #0b0f14;
  box-shadow:inset -1px 0 0 #ffffff08,10px 0 34px #10182014;
}
.sidebar::before{
  content:"";position:absolute;top:0;left:0;right:0;height:5px;
  background:repeating-linear-gradient(135deg,var(--gf-orange) 0 10px,#1b222b 10px 20px);
}
.sidebar::after{
  content:"";position:absolute;left:0;right:0;bottom:0;height:130px;pointer-events:none;
  background:radial-gradient(190px 90px at 50% 105%,#e07b3918,transparent 70%);
}
.brand,.nav-caption,.sidebar nav,.sidebar-bottom{position:relative;z-index:1}
.brand b{font-family:'Chakra Petch',Manrope,sans-serif;font-size:18px;letter-spacing:.2px}
.brand small{letter-spacing:2.4px;color:#6f7c88}
.brand-mark{
  box-shadow:0 7px 20px #e07b3952,inset 0 1px 0 #ffffff66;
  transition:transform .55s cubic-bezier(.2,.8,.2,1),box-shadow .3s;
}
.brand:hover .brand-mark{transform:rotate(-12deg) scale(1.06);box-shadow:0 10px 26px #e07b3966,inset 0 1px 0 #ffffff80}
.nav-caption{display:flex;align-items:center;gap:9px}
.nav-caption::after{content:"";flex:1;height:1px;background:linear-gradient(90deg,#3a444f,transparent)}
.sidebar nav a{
  position:relative;border:1px solid transparent;
  transition:background .22s,color .22s,transform .22s,border-color .22s;
}
.sidebar nav a svg{transition:transform .3s,color .3s,filter .3s;opacity:.85}
.sidebar nav a:hover{background:#232c36;color:#fff;border-color:#ffffff0f;transform:translateX(3px)}
.sidebar nav a:hover svg{transform:rotate(-9deg) scale(1.12);color:#f2a06a}
.sidebar nav a.active{
  background:linear-gradient(90deg,#2d3742,#242c35);
  box-shadow:inset 3px 0 0 var(--gf-orange),0 5px 16px #00000024;
}
.sidebar nav a.active svg{color:#f09a61;filter:drop-shadow(0 0 6px #e07b3966);opacity:1}
.sidebar-bottom{border-top:1px solid #303842}
.online-dot{animation:pulseGreen 2.6s ease-out infinite}
.sidebar .avatar{box-shadow:0 0 0 2px #ffffff12}
.profile .icon-btn:hover svg{animation:iconSlide .5s ease}

/* ---------- Topbar ---------- */
.topbar{
  position:sticky;top:0;z-index:4;
  background:#ffffffd9;
  -webkit-backdrop-filter:blur(10px);
  backdrop-filter:blur(10px);
  box-shadow:0 1px 0 #ffffffb3,0 10px 26px #1b2a380a;
}
.topbar::after{
  content:"";position:absolute;left:0;right:0;bottom:-1px;height:2px;
  background:linear-gradient(90deg,transparent,#e07b392e 25%,#e07b3980 50%,#e07b392e 75%,transparent);
}
.breadcrumb b{font-family:'Chakra Petch',Manrope,sans-serif;letter-spacing:.2px}
.today{font-size:10px;letter-spacing:.6px;text-transform:uppercase}
.engine-chip{
  display:inline-flex;align-items:center;gap:6px;
  font-family:'Chakra Petch',Manrope,sans-serif;font-size:9px;letter-spacing:1.5px;
  color:#7d8792;border:1px solid var(--gf-line);background:#fbfcfd;
  padding:7px 10px;border-radius:20px;white-space:nowrap;
}
.engine-chip svg{color:var(--gf-orange);animation:gearSpin 9s linear infinite}
.mobile-menu:hover svg{animation:gearSpin .6s ease}
.top-right .avatar{box-shadow:0 0 0 3px #e07b391f;transition:transform .3s}
.top-right .avatar:hover{transform:scale(1.08) rotate(-4deg)}

/* ---------- Page title + content motion ---------- */
.content{animation:contentFade .3s ease}
@keyframes contentFade{from{opacity:0}to{opacity:1}}
.page-title .eyebrow{display:inline-flex;align-items:center;gap:7px}
.page-title .eyebrow::before{
  content:"";width:7px;height:7px;border-radius:50%;background:var(--gf-orange);
  box-shadow:0 0 0 4px #e07b391f;
}
.page-title h1{
  position:relative;display:inline-block;
  font-family:'Chakra Petch',Manrope,sans-serif;font-weight:700;letter-spacing:-.4px;
}
.page-title h1::after{
  content:"";display:block;height:3px;width:48px;margin-top:8px;border-radius:3px;
  background:repeating-linear-gradient(90deg,var(--gf-orange) 0 6px,transparent 6px 10px);
}
.title-actions{display:flex;align-items:center;gap:9px;flex-wrap:wrap;justify-content:flex-end}

/* ---------- Buttons ---------- */
.primary{
  position:relative;overflow:hidden;border-radius:9px;
  background:linear-gradient(135deg,#eb8a47,#d06a2e);
  border-color:#c96329;
  box-shadow:0 6px 16px #e07b3933,inset 0 1px 0 #ffffff4d;
  transition:transform .2s,box-shadow .2s,filter .2s;
}
.primary::after{
  content:"";position:absolute;top:-120%;left:-70%;width:38%;height:340%;transform:rotate(20deg);
  background:linear-gradient(90deg,transparent,#ffffff5c,transparent);
  transition:left .55s ease;pointer-events:none;
}
.primary:hover{transform:translateY(-1px);box-shadow:0 11px 24px #e07b3944;filter:saturate(1.06)}
.primary:hover::after{left:135%}
.primary:active{transform:translateY(1px) scale(.99)}
.primary svg{transition:transform .3s}
.btn-new:hover svg{transform:rotate(-18deg) scale(1.1)}
.btn-sell:hover svg{animation:iconPrint .55s ease}
.btn-restock:hover svg{animation:iconBounce .6s ease}
.btn-add:hover svg{transform:rotate(90deg)}
.ghost-btn{
  display:inline-flex;align-items:center;gap:6px;
  border:1px solid #e2e6ea;background:#fff;color:#68737e;
  font-size:11px;border-radius:8px;padding:9px 11px;
  transition:color .2s,border-color .2s,background .2s,transform .2s,box-shadow .2s;
}
.ghost-btn:hover{border-color:var(--gf-orange);color:var(--gf-orange-dark);background:#fff8f2;transform:translateY(-1px);box-shadow:0 6px 14px #e07b391a}
.ghost-btn:hover svg{animation:gearSpin .7s ease}
.ghost-btn:disabled{opacity:.65;cursor:wait;transform:none;box-shadow:none}
.spin-icon{animation:gearSpin .85s linear infinite}
.small-action{
  display:inline-flex;align-items:center;gap:5px;border-radius:6px;
  transition:color .2s,border-color .2s,background .2s,transform .2s,box-shadow .2s;
}
.small-action.sell{color:#438b69;border-color:#cfe7da;background:#f6fbf8}
.small-action.sell:hover{background:#e6f5ed;border-color:#9ed3b9;transform:translateY(-1px);box-shadow:0 5px 12px #4a9b781f}
.small-action.restock{color:var(--gf-orange-dark);border-color:#f0cdb4;background:#fff8f2}
.small-action.restock:hover{background:var(--gf-orange);border-color:var(--gf-orange);color:#fff;transform:translateY(-1px);box-shadow:0 6px 14px #e07b3938}
.small-action.restock:hover svg{animation:iconBounce .6s ease}
.text-button{transition:color .2s,transform .2s}
.text-button:hover{color:#a94f1e;transform:translateX(2px)}
.icon-btn{transition:background .2s,color .2s,transform .2s}
.icon-btn:hover{transform:rotate(6deg)}
.status-pill{border-color:#cfe7da;background:linear-gradient(180deg,#f7fdf9,#effaf4);box-shadow:0 3px 10px #4a9b7814}
.status-pill span{animation:pulseGreen 2.4s ease-out infinite}

/* ---------- Panels + stat cards ---------- */
.panel{
  position:relative;border-radius:14px;
  background:linear-gradient(180deg,#fff 0%,#fefdfc 100%);
  transition:border-color .25s,box-shadow .25s;
}
.panel::before{
  content:"";position:absolute;top:0;left:0;right:0;height:3px;
  background:repeating-linear-gradient(90deg,var(--gf-orange) 0 14px,transparent 14px 26px);
  opacity:0;transition:opacity .3s;
}
.panel:hover{border-color:#f0d3c0;box-shadow:0 16px 34px #1b2a3812}
.panel:hover::before{opacity:.65}
.panel-head h3{
  position:relative;padding-left:11px;
  font-family:'Chakra Petch',Manrope,sans-serif;letter-spacing:.1px;
}
.panel-head h3::before{
  content:"";position:absolute;left:0;top:1px;bottom:1px;width:3px;border-radius:2px;
  background:linear-gradient(180deg,var(--gf-orange),#f0b27a);
}
.sale-form-panel h3{font-family:'Chakra Petch',Manrope,sans-serif}
.tag{background:#fafbfc;border-color:#e6e9ed;font-family:'Chakra Petch',Manrope,sans-serif;letter-spacing:1.1px}
.stat-card{
  position:relative;overflow:hidden;border-radius:14px;
  background:linear-gradient(180deg,#fff,#fdfaf7);
  transition:transform .28s,box-shadow .28s,border-color .28s;
  animation:riseIn .5s cubic-bezier(.2,.8,.2,1) both;
}
.stat-card:nth-child(2){animation-delay:.05s}
.stat-card:nth-child(3){animation-delay:.1s}
.stat-card:nth-child(4){animation-delay:.15s}
.stat-card:nth-child(5){animation-delay:.2s}
.stat-card::before{
  content:"";position:absolute;top:0;left:18px;right:18px;height:2px;
  background:repeating-linear-gradient(90deg,var(--gf-orange) 0 8px,transparent 8px 14px);
  opacity:.3;transition:opacity .3s;
}
.stat-card::after{
  content:"";position:absolute;right:-45px;bottom:-55px;width:150px;height:150px;pointer-events:none;
  background:radial-gradient(circle,#e07b3922,transparent 68%);
  opacity:0;transition:opacity .35s;
}
.stat-card:hover{transform:translateY(-4px);border-color:#f0d3c0;box-shadow:0 18px 34px #1b2a3815}
.stat-card:hover::before{opacity:.8}
.stat-card:hover::after{opacity:1}
.stat-top span:first-child{
  font-family:'Chakra Petch',Manrope,sans-serif;letter-spacing:.7px;text-transform:uppercase;font-size:10px;
}
.stat-card>strong{
  font-family:'Chakra Petch',Manrope,sans-serif;font-weight:700;letter-spacing:-.4px;
  transition:color .25s;
}
.stat-card:hover>strong{color:#17202a}
.stat-icon{transition:transform .35s}
.stat-card:hover .stat-icon{transform:rotate(-8deg) scale(1.1)}
.stat-card:hover .stat-icon svg{animation:iconWiggle .65s ease}
.skeleton{
  display:inline-block;border-radius:6px;
  background:linear-gradient(90deg,#eceff2 25%,#f8f9fa 37%,#eceff2 63%);
  background-size:400% 100%;animation:shimmer 1.25s ease infinite;
}
.skeleton-line{width:74%;height:24px}
.skeleton-pill{width:46px;height:14px}
.chart-skeleton{width:100%;height:100%;border-radius:10px}

/* ---------- Charts ---------- */
.chart-foot{font-family:'Chakra Petch',Manrope,sans-serif;letter-spacing:.7px;text-transform:uppercase;font-size:9px}
.legend-dot{box-shadow:0 0 0 3px #e07b391f}
.legend-dot.green{box-shadow:0 0 0 3px #4a9b781f}
.stock-row{border-radius:8px;transition:background .25s,transform .25s;padding-left:6px;padding-right:6px}
.stock-row:hover{background:#fffaf6;transform:translateX(2px)}
.stock-copy b{font-size:11.5px}
.stock-count{font-family:'Chakra Petch',Manrope,sans-serif;font-size:15px}
.part-icon{transition:transform .3s,background .3s,color .3s}
.stock-row:hover .part-icon{transform:rotate(-6deg);background:var(--gf-orange-soft);color:var(--gf-orange)}
.workflow-number{transition:transform .3s,box-shadow .3s}
.workflow>div:not(.workflow-line):hover .workflow-number{transform:translateY(-2px);box-shadow:0 6px 14px #1b2a3814}
.empty svg{animation:floatY 2.6s ease-in-out infinite}

/* ---------- Tables ---------- */
thead th{
  font-family:'Chakra Petch',Manrope,sans-serif;letter-spacing:1.2px;
  background:#fafbfc;
}
tbody tr{transition:background .2s}
tbody tr:hover{background:#fffaf6}
tbody tr:hover td:first-child{box-shadow:inset 3px 0 0 var(--gf-orange)}
.searchbox{transition:border-color .2s,box-shadow .2s,background .2s}
.searchbox:focus-within{border-color:var(--gf-orange);box-shadow:0 0 0 3px #e07b391a;background:#fff}
.check-filter input{transition:transform .15s}
.check-filter input:checked{transform:scale(1.08)}
.result-count{font-family:'Chakra Petch',Manrope,sans-serif;letter-spacing:.4px;background:#f7f8f9;border-radius:5px;padding:5px 8px}
.badge{display:inline-flex;align-items:center;gap:5px;border:1px solid transparent}
.badge::before{content:"";width:5px;height:5px;border-radius:50%;background:currentColor}
.badge.low{box-shadow:0 2px 8px #c76d351a}
.badge.good{box-shadow:0 2px 8px #438b691a}
.badge.service{background:#e8eefc;color:#4f6bb0;border-color:#d8e2fa}
.badge.manual{background:#f2f3f5;color:#6b7480;border-color:#e7e9ec}
.type-pill{display:inline-flex;align-items:center;gap:5px}
.type-pill::before{content:"";width:5px;height:5px;border-radius:50%;background:currentColor}
.qty-in,.qty-out{font-family:'Chakra Petch',Manrope,sans-serif;letter-spacing:.3px}
.sku,.location-pill{transition:background .2s,color .2s}
tbody tr:hover .sku{background:#fff0e5;color:#a95c2f}
tbody tr:hover .location-pill{background:#fff4ec;color:#a95c2f}

/* ---------- Service board ---------- */
.service-column{
  position:relative;overflow:hidden;border:1px solid #e1e6eb;
  background:linear-gradient(180deg,#eef1f4,#e9edf1);
}
.service-column::before{
  content:"";position:absolute;top:0;left:0;right:0;height:4px;
  background:repeating-linear-gradient(90deg,#c9d1d9 0 8px,transparent 8px 16px);opacity:.75;
}
.column-heading b{font-family:'Chakra Petch',Manrope,sans-serif;letter-spacing:.3px}
.column-dot{box-shadow:0 0 0 3px #98a2ae22}
.column-dot.in_progress{animation:pulseDot 1.9s ease-out infinite}
.column-dot.completed{box-shadow:0 0 0 3px #56a98526}
.column-count{font-family:'Chakra Petch',Manrope,sans-serif}
.job-card{
  transition:transform .24s,box-shadow .24s,border-color .24s;
}
.job-card:hover{transform:translateY(-3px) rotate(-.25deg);box-shadow:0 13px 26px #1b2a3814;border-color:#efcdb6}
.job-card h3{font-family:'Chakra Petch',Manrope,sans-serif;letter-spacing:0}
.job-top span{letter-spacing:.3px}
.job-parts span{transition:background .2s,color .2s}
.job-card:hover .job-parts span{background:#fff1e7;color:#a95c2f}
.job-card select{transition:border-color .2s,box-shadow .2s}
.job-card select:focus{outline:none;border-color:var(--gf-orange);box-shadow:0 0 0 3px #e07b3917}

/* ---------- Forms + modals ---------- */
input,select,textarea{transition:border-color .2s,box-shadow .2s,background .2s}
.form-grid input:focus,.login-card input:focus,.sale-form input:focus,.sale-form select:focus,.parts-picker-row select:focus,.parts-picker-row input:focus{
  border-color:var(--gf-orange);box-shadow:0 0 0 3px #e07b391a;outline:none;
}
.parts-picker{background:#fcfdfd;border-color:#e6eaee}
.picked-row{transition:background .2s,transform .2s}
.picked-row:hover{background:#fff6ef;transform:translateX(2px)}
.hint{border:1px dashed #edd8c7;background:linear-gradient(180deg,#fdf7f1,#fbf4ee)}
.sell-summary{
  display:flex;align-items:center;justify-content:space-between;gap:12px;
  background:#f7f8f9;border-radius:8px;padding:11px 13px;margin-bottom:16px;
  color:#68737e;font-size:11px;
}
.sell-summary b{font-family:'Chakra Petch',Manrope,sans-serif;font-size:15px;color:#2c3540}
.sell-total{margin-top:14px;font-family:'Chakra Petch',Manrope,sans-serif;letter-spacing:.3px}
.sell-total b{font-size:18px;color:var(--gf-orange-dark)}
.sale-preview{
  display:flex;align-items:center;justify-content:space-between;gap:12px;
  margin-top:13px;padding:10px 13px;border-radius:8px;
  background:linear-gradient(90deg,#fff8f2,#fdfbf9);border:1px dashed #f0d6c3;
  color:#8a6b55;font-size:11px;
}
.sale-preview b{font-family:'Chakra Petch',Manrope,sans-serif;font-size:16px;color:var(--gf-orange-dark)}
.modal-backdrop{-webkit-backdrop-filter:blur(3px);backdrop-filter:blur(3px);animation:fadeIn .2s ease}
.modal{position:relative;animation:modalIn .3s cubic-bezier(.2,.9,.3,1.15)}
.modal::before{
  content:"";position:absolute;top:0;left:0;right:0;height:4px;border-radius:12px 12px 0 0;
  background:repeating-linear-gradient(135deg,var(--gf-orange) 0 10px,#232b34 10px 20px);
}
.modal .panel-head h3{padding-left:0}
.modal .panel-head h3::before{display:none}
.restock-badge{transition:background .2s}
.restock-badge:hover{background:#fff6ee}
.error{animation:shakeX .35s ease;border:1px solid #f2d2d2}

/* ---------- Toasts ---------- */
.toast-stack{position:fixed;right:22px;bottom:22px;z-index:40;display:grid;gap:9px;max-width:min(360px,90vw)}
.toast{
  display:flex;align-items:center;gap:9px;
  background:#1c242d;color:#f2f4f6;padding:11px 14px;border-radius:9px;
  font-size:11.5px;line-height:1.45;
  box-shadow:0 16px 34px #0b111a42;border-left:3px solid var(--gf-green);
  animation:toastIn .32s cubic-bezier(.2,.9,.3,1.2);
}
.toast svg{flex-shrink:0;color:#7fd0a9}
.toast.error{border-left-color:#c34e4e}
.toast.error svg{color:#f0a1a1}
.toast.info{border-left-color:#d9a441}
.toast.info svg{color:#f0cf8a}

/* ---------- Login ---------- */
.login-art{
  background:
    radial-gradient(720px 420px at 82% 8%,#e07b3926,transparent 62%),
    linear-gradient(158deg,#141b23 0%,#222b35 58%,#161d25 100%);
}
.login-art::before{
  content:"";position:absolute;inset:0;pointer-events:none;
  background:repeating-linear-gradient(135deg,#ffffff05 0 2px,transparent 2px 12px);
}
.login-art:after{
  border-color:#ffffff12;
  box-shadow:0 0 0 50px #ffffff04,0 0 0 100px #ffffff03;
}
.login-art>.brand-mark,.login-art>.eyebrow,.login-art>h1,.login-art>.muted-light,.login-art>.login-metric{position:relative;z-index:1}
.login-gear{position:absolute;color:#ffffff0a;pointer-events:none;z-index:0;animation:gearSpin 30s linear infinite}
.login-gear.g1{right:-74px;top:-48px}
.login-gear.g2{left:-42px;bottom:-42px;animation-direction:reverse;animation-duration:19s;color:#e07b3912}
.login-art h1{font-family:'Chakra Petch',Manrope,sans-serif;letter-spacing:-1.2px}
.login-card{position:relative;animation:riseIn .5s .05s cubic-bezier(.2,.8,.2,1) both}
.login-card::before{
  content:"";position:absolute;top:0;left:0;right:0;height:4px;border-radius:12px 12px 0 0;
  background:repeating-linear-gradient(135deg,var(--gf-orange) 0 9px,transparent 9px 18px);
  opacity:.85;
}
.login-card h2{font-family:'Chakra Petch',Manrope,sans-serif;letter-spacing:-.4px}
.login-card .primary:hover svg{animation:iconSlide .55s ease}
.demo-hint{border:1px dashed #e7eaee;background:#fafbfc}

/* ---------- Keyframes ---------- */
@keyframes pageIn{from{opacity:0;transform:translateY(10px)}to{opacity:1;transform:none}}
@keyframes riseIn{from{opacity:0;transform:translateY(13px)}to{opacity:1;transform:none}}
@keyframes modalIn{from{opacity:0;transform:translateY(16px) scale(.965)}to{opacity:1;transform:none}}
@keyframes toastIn{from{opacity:0;transform:translateX(26px)}to{opacity:1;transform:none}}
@keyframes fadeIn{from{opacity:0}to{opacity:1}}
@keyframes shimmer{0%{background-position:100% 0}100%{background-position:-100% 0}}
@keyframes gearSpin{to{transform:rotate(360deg)}}
@keyframes iconWiggle{0%,100%{transform:rotate(0)}25%{transform:rotate(-10deg)}75%{transform:rotate(10deg)}}
@keyframes iconBounce{0%,100%{transform:translateY(0)}35%{transform:translateY(-4px) rotate(-9deg)}70%{transform:translateY(1px)}}
@keyframes iconPrint{0%,100%{transform:translateY(0)}40%{transform:translateY(-3px) rotate(-6deg)}}
@keyframes iconSlide{0%,100%{transform:translateX(0)}40%{transform:translateX(3px)}}
@keyframes pulseGreen{0%{box-shadow:0 0 0 0 #50b58b70}70%{box-shadow:0 0 0 6px #50b58b00}100%{box-shadow:0 0 0 0 #50b58b00}}
@keyframes pulseDot{0%{box-shadow:0 0 0 0 #e18a4d70}70%{box-shadow:0 0 0 7px #e18a4d00}100%{box-shadow:0 0 0 0 #e18a4d00}}
@keyframes shakeX{0%,100%{transform:translateX(0)}25%{transform:translateX(-4px)}75%{transform:translateX(4px)}}
@keyframes floatY{0%,100%{transform:translateY(0)}50%{transform:translateY(-3px)}}

/* ---------- Scrollbar + accessibility ---------- */
*{scrollbar-width:thin;scrollbar-color:#c4cbd1 transparent}
::-webkit-scrollbar{width:9px;height:9px}
::-webkit-scrollbar-thumb{background:#c4cbd1;border-radius:8px;border:2px solid transparent;background-clip:content-box}
::-webkit-scrollbar-thumb:hover{background:#a7b0b9;background-clip:content-box}
:focus-visible{outline:2px solid #e07b3980;outline-offset:2px}

@media(max-width:700px){
  .title-actions{width:100%;justify-content:flex-start}
  .engine-chip{display:none}
  .toast-stack{right:14px;bottom:14px;left:14px;max-width:none}
  .stat-card:hover{transform:none}
  .job-card:hover{transform:none}
  .topbar::after{display:none}
}

@media(prefers-reduced-motion:reduce){
  *,*::before,*::after{animation-duration:.01ms!important;animation-iteration-count:1!important;transition-duration:.01ms!important;scroll-behavior:auto!important}
}

.login-page{background:linear-gradient(135deg,#ffffff,#faf8f6)}
.login-card{
  background:#fff;border:1px solid #edf0f3;border-radius:16px;padding:34px;
  box-shadow:0 26px 60px #1b2a381a,0 2px 0 #ffffff inset;
}
@media(max-width:700px){.login-card{padding:30px 22px}}

/* logo lockup + balanced stat grid */
.brand-mark{display:flex;align-items:center;justify-content:center;gap:0;white-space:nowrap;font-size:16px;letter-spacing:-1px;line-height:1}
@media(max-width:1150px){
  .stat-grid{grid-template-columns:repeat(2,minmax(0,1fr))}
  .stat-card:nth-child(5){grid-column:1/-1}
}
@media(min-width:1151px){
  .stat-grid{grid-template-columns:repeat(5,minmax(0,1fr))}
  .stat-card>strong{font-size:clamp(20px,1.75vw,26px)}
}

@media(min-width:801px) and (max-width:1150px){
  .stat-grid{grid-template-columns:repeat(6,minmax(0,1fr))}
  .stat-grid .stat-card{grid-column:span 2}
  .stat-grid .stat-card:nth-child(4),.stat-grid .stat-card:nth-child(5){grid-column:span 3}
}
@media(max-width:800px){
  .stat-grid{grid-template-columns:repeat(2,minmax(0,1fr))}
  .stat-grid .stat-card:nth-child(5){grid-column:1/-1}
}

.stat-card .stat-icon svg{animation:floatY 3.8s ease-in-out infinite}
.stat-card:nth-child(2) .stat-icon svg{animation-delay:.3s}
.stat-card:nth-child(3) .stat-icon svg{animation-delay:.6s}
.stat-card:nth-child(4) .stat-icon svg{animation-delay:.9s}
.stat-card:nth-child(5) .stat-icon svg{animation-delay:1.2s}

/* login layout: keep the metric in flow so it never overlaps the copy */
.login-art{justify-content:center;padding:8vh 8vw 56px}
.login-metric{position:static;left:auto;right:auto;bottom:auto;width:100%;margin-top:52px}
@media(max-height:780px){
  .login-art .brand-mark{margin-bottom:30px}
  .login-art h1{margin:10px 0 12px}
  .login-metric{margin-top:30px}
}

/* ---------- Narrow viewport: parts table becomes a stack of cards ---------- */
@media(max-width:900px){
  .parts-table{display:block;white-space:normal}
  .parts-table thead{display:none}
  .parts-table tbody{display:block}
  .parts-table tbody tr{
    display:grid;grid-template-columns:minmax(0,1fr) auto;
    grid-template-areas:"name status" "sku stock" "loc price" "actions actions";
    align-items:center;gap:7px 14px;
    padding:14px 10px;border-bottom:1px solid #f0f2f4;border-left:2px solid transparent;
    transition:background .2s,border-color .2s;
  }
  .parts-table tbody tr:last-child{border-bottom:0}
  .parts-table tbody tr:hover{background:#fffaf6;border-left-color:var(--gf-orange)}
  .parts-table tbody td{padding:0;border:0;font-size:11px;display:block}
  .parts-table tbody tr:hover td:first-child{box-shadow:none}
  .parts-table td:nth-child(1){grid-area:name}
  .parts-table td:nth-child(2){grid-area:sku}
  .parts-table td:nth-child(3){grid-area:loc}
  .parts-table td:nth-child(4){grid-area:stock;text-align:right}
  .parts-table td:nth-child(5){grid-area:price;text-align:right;color:#4d5865}
  .parts-table td:nth-child(6){grid-area:status;justify-self:end}
  .parts-table td:nth-child(7){
    grid-area:actions;display:flex;justify-content:flex-end;
    margin-top:4px;padding-top:9px;border-top:1px dashed #eef1f4;
  }
  .parts-table .cell-sub{margin-top:3px}
  .parts-table .row-actions{gap:7px}
  .parts-table .small-action{padding:7px 10px}
}

/* title flow fix: keep the eyebrow on its own line */
.page-title h1{display:block;width:fit-content}
.page-title .eyebrow{display:flex;width:fit-content}

/* ---------- Narrow viewport: sale form + sales table ---------- */
@media(max-width:900px){
  .sale-form{flex-wrap:wrap}
  .sale-form select{flex:1 1 100%}
  .sale-form input[type="number"]{flex:1 1 30%}
  .sale-form .date-input{flex:1 1 55%}
  .sale-form .primary{flex:1 1 100%;justify-content:center}

  .sales-table{display:block;white-space:normal}
  .sales-table thead{display:none}
  .sales-table tbody{display:block}
  .sales-table tbody tr{
    display:grid;grid-template-columns:minmax(0,1fr) auto;
    grid-template-areas:"part total" "date source" "qty price";
    align-items:center;gap:6px 14px;
    padding:13px 10px;border-bottom:1px solid #f0f2f4;border-left:2px solid transparent;
  }
  .sales-table tbody tr:last-child{border-bottom:0}
  .sales-table tbody tr:hover{background:#fffaf6;border-left-color:var(--gf-orange)}
  .sales-table tbody td{padding:0;border:0;display:block;font-size:11px}
  .sales-table tbody tr:hover td:first-child{box-shadow:none}
  .sales-table td:nth-child(1){grid-area:date;color:#8d97a3;font-size:10px}
  .sales-table td:nth-child(2){grid-area:part}
  .sales-table td:nth-child(3){grid-area:source;justify-self:end}
  .sales-table td:nth-child(4){grid-area:qty;color:#8d97a3;font-size:10px}
  .sales-table td:nth-child(5){grid-area:price;justify-self:end;color:#8d97a3;font-size:10px}
  .sales-table td:nth-child(6){grid-area:total;justify-self:end}
  .sales-table td:nth-child(6) b{font-size:12px}
}

/* ============================================================
   GARAGEFLOW · STAGE 2 — INSTRUMENT PANEL / WORKSHOP KIT
   Stencil numbers, LED readouts, ticker, tickets, textures.
   ============================================================ */

/* ---------- Global texture ---------- */
::selection{background:#e07b3966;color:#10161d}
body::after{
  content:"";position:fixed;inset:0;pointer-events:none;z-index:2;
  background-image:
    radial-gradient(#0d111614 1px,transparent 1.2px),
    radial-gradient(#ffffff26 1px,transparent 1.2px);
  background-size:5px 5px,7px 7px;
  background-position:0 0,3px 4px;
  opacity:.5;
}

/* ---------- Page title: stencil ghost number + sticker eyebrow ---------- */
.page-title{position:relative;z-index:1}
.page-no{
  position:absolute;left:-10px;top:-48px;z-index:-1;pointer-events:none;user-select:none;
  font-family:'Share Tech Mono',monospace;font-size:138px;line-height:1;letter-spacing:-12px;
  color:transparent;-webkit-text-stroke:1.5px #1b222b22;
}
.page-title .eyebrow{
  display:flex;align-items:center;width:fit-content;
  background:#10161d;color:#ffcf9e;padding:6px 11px;border-radius:3px;
  transform:rotate(-1.3deg);box-shadow:3px 3px 0 #10161d33;
  font-family:'Share Tech Mono',monospace;letter-spacing:2.4px;font-size:10px;
  transition:transform .25s,box-shadow .25s;
}
.page-title .eyebrow:hover{transform:rotate(1.2deg) translateY(-1px);box-shadow:5px 5px 0 #10161d40}
.page-title .eyebrow::before{background:#ff8a3c;box-shadow:0 0 8px #ff8a3c;animation:pulseDot 2.2s ease-out infinite}
.page-title h1{color:#10161d;text-shadow:0 3px 0 #10161d0f}
.page-title .subtle{max-width:56ch}

/* ---------- Stat cards → instrument cluster ---------- */
.stat-card{
  background:
    radial-gradient(circle 3px at 11px 11px,#ffffff30 96%,transparent 100%),
    radial-gradient(circle 3px at calc(100% - 11px) 11px,#ffffff30 96%,transparent 100%),
    radial-gradient(circle 3px at 11px calc(100% - 11px),#ffffff20 96%,transparent 100%),
    radial-gradient(circle 3px at calc(100% - 11px) calc(100% - 11px),#ffffff20 96%,transparent 100%),
    linear-gradient(165deg,#1b232c 0%,#151b23 55%,#10151b 100%);
  border:1px solid #0b0f14;
  color:#cfd8e1;
  box-shadow:inset 0 1px 0 #ffffff1a,6px 6px 0 #10151b17,0 18px 34px #10151b1f;
}
.stat-card::before{top:0;left:0;right:0;height:4px;opacity:.9;
  background:repeating-linear-gradient(135deg,var(--gf-orange) 0 9px,#10161d 9px 18px)}
.stat-card:hover::before{opacity:1}
.stat-card .stat-top span:first-child{color:#7f8b99;letter-spacing:1.1px}
.stat-card>strong{
  display:block;font-family:'Share Tech Mono',monospace;font-weight:400;
  font-size:clamp(24px,2vw,33px);letter-spacing:.4px;color:#ffab6b;
  text-shadow:0 0 20px #ff7a2e59;margin:13px 0 6px;overflow-wrap:anywhere;
}
.stat-card:hover>strong{color:#ffc79b;text-shadow:0 0 26px #ff7a2e7a}
.stat-card>small{color:#6d7885}
.stat-card .stat-icon{
  display:grid;place-items:center;width:34px;height:34px;border-radius:9px;
  background:#ffffff0d;border:1px solid #ffffff1a;color:#ff9d5c;
}
.stat-card .stat-icon.warning{background:#e07b3924;border-color:#e07b394d;color:#ffb073}
.stat-card .skeleton-line{background:#2a333d}
.stat-card:hover{border-color:#e07b39;border-color:#e07b3980;transform:translateY(-5px) rotate(-.35deg)}

/* running LED meter (decorative, always moving) */
.led-meter{display:flex;gap:5px;margin-top:13px}
.led-meter i{
  width:17px;height:5px;border-radius:1.5px;background:#ffffff14;
  box-shadow:inset 0 0 0 1px #ffffff10;animation:ledRun 2.4s linear infinite;
}
.led-meter i:nth-child(2){animation-delay:.18s}
.led-meter i:nth-child(3){animation-delay:.36s}
.led-meter i:nth-child(4){animation-delay:.54s}
.led-meter i:nth-child(5){animation-delay:.72s}
@keyframes ledRun{
  0%,34%{background:#ffffff14;box-shadow:inset 0 0 0 1px #ffffff10}
  42%,54%{background:#ff8f45;box-shadow:0 0 11px #ff8f45aa}
  62%,100%{background:#ffffff14;box-shadow:inset 0 0 0 1px #ffffff10}
}

/* ---------- Low-stock ticker ---------- */
.ticker{
  display:flex;align-items:stretch;overflow:hidden;margin-bottom:17px;
  background:linear-gradient(90deg,#151b23,#1b232c 60%,#151b23);
  border:1px solid #0b0f14;border-radius:8px;
  box-shadow:4px 4px 0 #10151b14,inset 0 1px 0 #ffffff14;
}
.ticker-label{
  display:flex;align-items:center;gap:7px;padding:10px 14px;flex-shrink:0;
  background:repeating-linear-gradient(135deg,#e07b39 0 9px,#d06a2e 9px 18px);
  color:#141a21;font-family:'Share Tech Mono',monospace;font-size:10px;letter-spacing:2px;
}
.ticker-label svg{animation:iconBounce 1.6s ease infinite}
.ticker-window{
  position:relative;flex:1;min-width:0;overflow:hidden;
  -webkit-mask-image:linear-gradient(90deg,transparent,#000 5%,#000 94%,transparent);
  mask-image:linear-gradient(90deg,transparent,#000 5%,#000 94%,transparent);
}
.ticker-track{display:flex;width:max-content;animation:marquee 30s linear infinite;will-change:transform}
.ticker:hover .ticker-track{animation-play-state:paused}
.ticker-group{display:flex;flex-shrink:0}
.ticker-group span{
  display:flex;align-items:center;gap:8px;padding:10px 22px;
  border-right:1px dashed #ffffff1f;color:#c9d3dc;
  font-family:'Chakra Petch',sans-serif;font-size:11px;letter-spacing:.7px;white-space:nowrap;
}
.ticker-group span svg{color:#ff9d5c;opacity:.8}
.ticker-group span:last-child{border-right:1px dashed #ffffff1f}
@keyframes marquee{from{transform:translateX(0)}to{transform:translateX(-50%)}}

/* ---------- Buttons: tactile, stencil-label ---------- */
.primary{
  font-family:'Chakra Petch',sans-serif;font-weight:600;font-size:11.5px;
  text-transform:uppercase;letter-spacing:.7px;
  box-shadow:0 6px 16px #e07b3933,inset 0 1px 0 #ffffff4d,4px 4px 0 #10161d22;
}
.primary:hover{transform:translate(-2px,-2px);
  box-shadow:0 12px 26px #e07b3947,inset 0 1px 0 #ffffff4d,7px 7px 0 #10161d2b}
.primary:active{transform:translate(2px,2px);
  box-shadow:inset 0 1px 0 #ffffff4d,1px 1px 0 #10161d22}
.ghost-btn{
  font-family:'Share Tech Mono',monospace;text-transform:uppercase;
  letter-spacing:1.6px;font-size:10px;border-style:dashed;
}
.ghost-btn:hover{border-style:solid;background:#fff6ee}
.small-action{
  font-family:'Chakra Petch',sans-serif;font-weight:600;font-size:9.5px;
  text-transform:uppercase;letter-spacing:.7px;
}
.status-pill{font-family:'Share Tech Mono',monospace;letter-spacing:1.4px;text-transform:uppercase;font-size:10px}
.engine-chip{background:#151b23;border-color:#0b0f14;color:#8b96a3;
  box-shadow:3px 3px 0 #10151b17,inset 0 1px 0 #ffffff14}

/* ---------- Tables: workshop ledger ---------- */
thead th{
  color:#5d6875;border-bottom:2px solid #1b222b26;letter-spacing:1.5px;
  background-image:repeating-linear-gradient(135deg,#10161d0d 0 5px,transparent 5px 11px);
}
tbody tr{position:relative}
tbody tr::after{
  content:"";position:absolute;inset:0;pointer-events:none;opacity:0;transition:opacity .3s;
  background:linear-gradient(90deg,#e07b3926,transparent 58%);
}
tbody tr:hover::after{opacity:1}
tbody tr>*{position:relative}
.sku{font-family:'Share Tech Mono',monospace;letter-spacing:.4px}
.badge{
  font-family:'Chakra Petch',sans-serif;text-transform:uppercase;letter-spacing:.7px;
  border-width:1.5px;transform:rotate(-1.6deg);transition:transform .2s,box-shadow .2s;
}
tbody tr:hover .badge{transform:rotate(2.2deg) scale(1.07)}
.type-pill{
  font-family:'Chakra Petch',sans-serif;font-size:9.5px;text-transform:uppercase;
  letter-spacing:.6px;transform:rotate(-1deg);
}
.result-count,.tag{font-family:'Share Tech Mono',monospace;letter-spacing:1.4px}
.cell-sub{font-family:'Chakra Petch',sans-serif;letter-spacing:.4px}

/* ---------- Service board: work-order tickets ---------- */
.service-columns>.service-column:nth-child(1){--col:#8b95a2}
.service-columns>.service-column:nth-child(2){--col:#e07b39}
.service-columns>.service-column:nth-child(3){--col:#d9a441}
.service-columns>.service-column:nth-child(4){--col:#56a985}
.service-column{box-shadow:6px 6px 0 #10161d12}
.service-column::before{height:5px;opacity:.85;
  background:repeating-linear-gradient(90deg,var(--col,#8b95a2) 0 11px,transparent 11px 22px)}
.column-heading{
  margin:-12px -12px 12px;padding:11px 13px;
  background:linear-gradient(180deg,#171e26,#12171e);
  border-bottom:1px solid #0b0f14;color:#dfe6ec;
}
.column-heading b{font-family:'Chakra Petch',sans-serif;letter-spacing:.5px;text-transform:uppercase;font-size:11px}
.column-dot{background:var(--col);box-shadow:0 0 10px var(--col)}
.column-dot.completed{box-shadow:0 0 10px #56a985}
.column-count{
  background:var(--col);color:#10161d;font-weight:700;padding:2px 9px;border-radius:4px;
  box-shadow:0 2px 8px #00000040;font-family:'Share Tech Mono',monospace;
}
.job-card{position:relative;overflow:hidden;padding-left:20px;border-radius:6px}
.job-card::before{
  content:"";position:absolute;left:0;top:0;bottom:0;width:7px;transition:.3s;
  background:repeating-linear-gradient(180deg,var(--col,#8b95a2) 0 9px,transparent 9px 18px);
  opacity:.45;
}
.job-card:hover::before{opacity:1;box-shadow:0 0 14px var(--col)}
.job-card::after{
  position:absolute;right:7px;bottom:44px;pointer-events:none;
  font-family:'Chakra Petch',sans-serif;font-weight:700;font-size:10px;letter-spacing:3px;
  color:var(--col);border:2px solid var(--col);padding:3px 8px;border-radius:4px;
  transform:rotate(-8deg);opacity:.16;transition:.3s;
}
.service-columns>.service-column:nth-child(1) .job-card::after{content:"ANTRIAN"}
.service-columns>.service-column:nth-child(2) .job-card::after{content:"DIPROSES"}
.service-columns>.service-column:nth-child(3) .job-card::after{content:"PART ORDER"}
.service-columns>.service-column:nth-child(4) .job-card::after{content:"SELESAI"}
.job-card:hover::after{opacity:.6;transform:rotate(-8deg) translateY(-3px)}
.job-card:hover .job-top{color:#a95c2f}
.job-card select{
  font-family:'Chakra Petch',sans-serif;font-size:10.5px;letter-spacing:.6px;
  background:#f7f8f9;border-style:dashed;
}
.job-card select:focus{background:#fff;border-style:solid}
.column-empty{
  font-family:'Share Tech Mono',monospace;font-size:10px;letter-spacing:1.4px;
  border:1px dashed #b9c1ca;padding:16px 10px;text-align:center;color:#8b95a2;
  background:#ffffff8c;
}

/* ---------- Charts: gauge + gradient ---------- */
.tag{background:#151b23;border-color:#0b0f14;color:#b9c4ce;letter-spacing:1.6px}
.recharts-default-tooltip{
  background:#12171de6!important;border:1px solid #e07b394d!important;
  border-radius:6px!important;box-shadow:0 14px 30px #0b111a55!important;
}
.recharts-tooltip-label{
  color:#ffab6b!important;font-family:'Share Tech Mono',monospace!important;
  font-size:10px!important;letter-spacing:1.4px!important;
}
.recharts-tooltip-item{
  color:#e6ecf2!important;font-family:'Share Tech Mono',monospace!important;font-size:11px!important;
}
.recharts-cartesian-axis-tick-value{font-family:'Share Tech Mono',monospace!important}
.chart .recharts-area-curve{stroke-dasharray:2400;stroke-dashoffset:2400;animation:drawLine 1.7s .15s ease forwards}
@keyframes drawLine{to{stroke-dashoffset:0}}
.chart-foot span:first-child{color:#8b6f5c}

/* ---------- Toast: sticker ---------- */
.toast{
  border:1px solid #00000061;border-radius:6px;
  box-shadow:5px 5px 0 #0b111a66;
  font-family:'Chakra Petch',sans-serif;letter-spacing:.4px;
}
.toast svg{filter:drop-shadow(0 0 6px #7fd0a9aa)}
.toast.error svg{filter:drop-shadow(0 0 6px #f0a1a1aa)}

/* ---------- Sidebar: bolted plate + moving gear ---------- */
.side-gear{
  position:absolute;right:-150px;top:36%;color:#ffffff0a;pointer-events:none;
  animation:gearSpin 55s linear infinite;
}
.sidebar nav a{font-family:'Chakra Petch',sans-serif;letter-spacing:.3px;font-weight:500}
.sidebar nav a::before{
  content:"";position:absolute;inset:0;border-radius:8px;pointer-events:none;
  background:linear-gradient(90deg,transparent,#e07b392e,transparent);
  transform:translateX(-115%);transition:transform .55s ease;
}
.sidebar nav a:hover::before{transform:translateX(115%)}
.sidebar nav a::after{
  content:"";position:absolute;right:11px;top:50%;width:5px;height:5px;border-radius:50%;
  background:#39434e;transform:translateY(-50%);transition:.25s;
}
.sidebar nav a:hover::after{background:#6d7885}
.sidebar nav a.active::after{background:#ff8a3c;box-shadow:0 0 10px #ff8a3c;animation:pulseDot 2s ease-out infinite}
.nav-caption{font-family:'Share Tech Mono',monospace;letter-spacing:3px}
.brand small{font-family:'Share Tech Mono',monospace;letter-spacing:3.4px}
.sidebar .profile{background:#ffffff0a;border:1px dashed #ffffff1f;border-radius:9px;padding:7px}
.avatar{font-family:'Share Tech Mono',monospace}

/* ---------- Topbar ---------- */
.today{font-family:'Share Tech Mono',monospace}
.breadcrumb{font-family:'Chakra Petch',sans-serif;letter-spacing:.3px}

/* ---------- Login: tachometer ---------- */
.login-tach,.login-needle{position:absolute;right:-72px;top:-64px;pointer-events:none;z-index:0}
.login-tach{
  width:250px;height:250px;border-radius:50%;opacity:.5;
  background:conic-gradient(from 210deg,#e07b39 0deg 76deg,#ffffff1a 76deg 300deg,transparent 300deg);
  -webkit-mask:radial-gradient(circle,transparent 62%,#000 63%);
  mask:radial-gradient(circle,transparent 62%,#000 63%);
}
.login-needle{width:250px;height:250px}
.login-needle::after{
  content:"";position:absolute;left:50%;top:50%;width:44%;height:4px;border-radius:4px;
  transform-origin:0 50%;transform:rotate(336deg);
  background:linear-gradient(90deg,#ff8a3c,#ffd9b5);box-shadow:0 0 16px #ff8a3cb3;
  animation:needleSweep 5.5s cubic-bezier(.55,.05,.4,1) infinite;
}
@keyframes needleSweep{
  0%,100%{transform:rotate(336deg)}
  30%{transform:rotate(200deg)}
  55%{transform:rotate(246deg)}
  78%{transform:rotate(190deg)}
}
.login-card h2{text-transform:none;letter-spacing:-.6px}
.login-card .demo-hint{font-family:'Share Tech Mono',monospace;letter-spacing:.6px}

/* ---------- Dashboard micro-rail ---------- */
.workflow-line{
  height:4px;border-radius:4px;
  background:repeating-linear-gradient(90deg,#e07b39 0 9px,transparent 9px 18px);
  background-size:18px 4px;animation:marchX 1.1s linear infinite;
}
@keyframes marchX{to{background-position:18px 0}}
.workflow-number{
  font-family:'Share Tech Mono',monospace;font-weight:400;letter-spacing:.5px;
  background:#fff;border:1px dashed #c9d1d9;box-shadow:3px 3px 0 #10161d12;
  transition:transform .25s,box-shadow .25s,color .25s,border-color .25s;
}
.workflow>div:not(.workflow-line):hover .workflow-number{
  transform:rotate(-9deg) translateY(-3px);border-color:#e07b39;color:#c4632a;
  box-shadow:5px 5px 0 #e07b3926;
}
.workflow b{font-family:'Chakra Petch',sans-serif;letter-spacing:.5px}
.workflow small{font-family:'Chakra Petch',sans-serif}

/* ---------- Inline link sweep ---------- */
.link{position:relative;transition:color .2s}
.link:hover{color:#a94f1e}
.link::after{
  content:"";position:absolute;left:0;right:100%;bottom:-5px;height:2px;border-radius:2px;
  background:#e07b39;transition:right .3s ease;
}
.link:hover::after{right:0}

/* ---------- Stock list + empties ---------- */
.stock-count{font-family:'Share Tech Mono',monospace;letter-spacing:.5px}
.part-icon{border:1px solid #e4e8ec;background:linear-gradient(180deg,#fff,#f5f7f8)}
.empty{
  border:1px dashed #d3d9de;border-radius:8px;background:#fbfcfd;
  font-family:'Chakra Petch',sans-serif;letter-spacing:.4px;
}
.column-empty{border-radius:6px}

/* ---------- Service header + preview ---------- */
.workflow-number.orange{box-shadow:3px 3px 0 #e07b392b}
.workflow-number.green{box-shadow:3px 3px 0 #4a9b7830}

/* profile spacing fix */
.sidebar .profile{background:#ffffff08;border:1px dashed #ffffff1a;border-radius:9px;padding:6px 5px 6px 7px}
.sidebar .profile .icon-btn{flex-shrink:0}
.profile-copy b{white-space:nowrap;overflow:hidden;text-overflow:ellipsis}

.profile-copy b{font-size:12px;letter-spacing:.2px}
.sidebar .profile .avatar{width:28px;height:28px;flex-shrink:0}
.sidebar .profile .icon-btn{padding:5px}

.sidebar .profile{gap:7px;padding:5px 4px 5px 6px}
.sidebar .profile .avatar{width:26px;height:26px}
.sidebar .profile .icon-btn{padding:4px}
.profile-copy b{font-size:11.5px}

/* interactive affordances */
.stat-card.clickable{cursor:pointer}
.stat-card.clickable .stat-top span:first-child::after{content:"  \2197";opacity:0;transition:opacity .25s;color:#ff9d5c;font-family:monospace}
.stat-card.clickable:hover .stat-top span:first-child::after,
.stat-card.clickable:focus-visible .stat-top span:first-child::after{opacity:1}
.stat-card.clickable:focus-visible{outline:2px solid #e07b3980;outline-offset:3px}
.ticker{cursor:pointer;transition:transform .2s,box-shadow .2s}
.ticker:hover{box-shadow:6px 6px 0 #10151b20,inset 0 1px 0 #ffffff1a}
.ticker:active{transform:translateY(1px)}
.sku{border:1px solid transparent;font-family:inherit;cursor:pointer;transition:background .2s,color .2s,border-color .2s,transform .2s}
.sku:hover{background:#fff0e5;color:#a95c2f;border-color:#f0cdb4;transform:translateY(-1px)}
.sku:active{transform:translateY(0)}
```


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
	"motorcycle-garage-inventory/backend/internal/models"
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

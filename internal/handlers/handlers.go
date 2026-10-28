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

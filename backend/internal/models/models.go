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

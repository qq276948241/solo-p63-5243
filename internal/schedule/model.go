package schedule

import (
	"clinic/internal/user"
	"time"

	"gorm.io/gorm"
)

const (
	StatusAvailable = "available"
	StatusCanceled  = "canceled"
	StatusFull      = "full"
)

type Schedule struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	DoctorID  uint           `gorm:"not null;index" json:"doctor_id"`
	Date      string         `gorm:"size:20;not null;index" json:"date"`
	StartTime string         `gorm:"size:10;not null" json:"start_time"`
	EndTime   string         `gorm:"size:10;not null" json:"end_time"`
	MaxCount  int            `gorm:"default:1" json:"max_count"`
	Booked    int            `gorm:"default:0" json:"booked"`
	Status    string         `gorm:"size:20;default:available" json:"status"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
	Doctor    user.User      `gorm:"foreignKey:DoctorID" json:"doctor,omitempty"`
}

type ScheduleDetail struct {
	ID         uint   `json:"id"`
	DoctorID   uint   `json:"doctor_id"`
	DoctorName string `json:"doctor_name"`
	Department string `json:"department"`
	Title      string `json:"title"`
	Date       string `json:"date"`
	StartTime  string `json:"start_time"`
	EndTime    string `json:"end_time"`
	MaxCount   int    `json:"max_count"`
	Booked     int    `json:"booked"`
	Status     string `json:"status"`
}

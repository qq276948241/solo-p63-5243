package appointment

import (
	"clinic/internal/user"
	"time"

	"gorm.io/gorm"
)

const (
	StatusPending   = "pending"
	StatusConfirmed = "confirmed"
	StatusCompleted = "completed"
	StatusCanceled  = "canceled"
)

type Appointment struct {
	ID         uint           `gorm:"primaryKey" json:"id"`
	PatientID  uint           `gorm:"not null;index" json:"patient_id"`
	DoctorID   uint           `gorm:"not null;index" json:"doctor_id"`
	ScheduleID uint           `gorm:"not null;index" json:"schedule_id"`
	Date       string         `gorm:"size:20;not null;index" json:"date"`
	StartTime  string         `gorm:"size:10;not null" json:"start_time"`
	EndTime    string         `gorm:"size:10;not null" json:"end_time"`
	Status     string         `gorm:"size:20;default:pending;index" json:"status"`
	Symptoms   string         `gorm:"type:text" json:"symptoms"`
	Remark     string         `gorm:"type:text" json:"remark"`
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
	DeletedAt  gorm.DeletedAt `gorm:"index" json:"-"`
	Patient    user.User      `gorm:"foreignKey:PatientID" json:"patient,omitempty"`
	Doctor     user.User      `gorm:"foreignKey:DoctorID" json:"doctor,omitempty"`
}

type AppointmentDetail struct {
	ID            uint   `json:"id"`
	PatientID     uint   `json:"patient_id"`
	PatientName   string `json:"patient_name"`
	PatientPhone  string `json:"patient_phone"`
	PatientGender string `json:"patient_gender"`
	PatientAge    int    `json:"patient_age"`
	DoctorID      uint   `json:"doctor_id"`
	DoctorName    string `json:"doctor_name"`
	Department    string `json:"department"`
	Title         string `json:"title"`
	ScheduleID    uint   `json:"schedule_id"`
	Date          string `json:"date"`
	StartTime     string `json:"start_time"`
	EndTime       string `json:"end_time"`
	Status        string `json:"status"`
	Symptoms      string `json:"symptoms"`
	Remark        string `json:"remark"`
	CreatedAt     string `json:"created_at"`
}

type Review struct {
	ID            uint           `gorm:"primaryKey" json:"id"`
	AppointmentID uint           `gorm:"not null;uniqueIndex" json:"appointment_id"`
	PatientID     uint           `gorm:"not null;index" json:"patient_id"`
	DoctorID      uint           `gorm:"not null;index" json:"doctor_id"`
	Rating        int            `gorm:"not null" json:"rating"`
	Comment       string         `gorm:"type:text" json:"comment"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	DeletedAt     gorm.DeletedAt `gorm:"index" json:"-"`
	Appointment   Appointment    `gorm:"foreignKey:AppointmentID" json:"-"`
	Patient       user.User      `gorm:"foreignKey:PatientID" json:"patient,omitempty"`
	Doctor        user.User      `gorm:"foreignKey:DoctorID" json:"-"`
}

type DoctorRating struct {
	DoctorID    uint    `json:"doctor_id"`
	AvgRating   float64 `json:"avg_rating"`
	TotalCount  int64   `json:"total_count"`
	RatingCount map[int]int64 `json:"rating_count"`
}

type ReviewDetail struct {
	ID          uint   `json:"id"`
	PatientName string `json:"patient_name"`
	Rating      int    `json:"rating"`
	Comment     string `json:"comment"`
	CreatedAt   string `json:"created_at"`
}

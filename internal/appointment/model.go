package appointment

import (
	"clinic/internal/user"
	"fmt"
	"time"

	"gorm.io/gorm"
)

type AppointmentStatus string

const (
	AppointmentStatusPending   AppointmentStatus = "pending"
	AppointmentStatusConfirmed AppointmentStatus = "confirmed"
	AppointmentStatusCompleted AppointmentStatus = "completed"
	AppointmentStatusCanceled  AppointmentStatus = "canceled"
)

func (s AppointmentStatus) Valid() bool {
	switch s {
	case AppointmentStatusPending,
		AppointmentStatusConfirmed,
		AppointmentStatusCompleted,
		AppointmentStatusCanceled:
		return true
	default:
		return false
	}
}

func (s AppointmentStatus) String() string {
	return string(s)
}

const (
	ReviewRatingMin = 1
	ReviewRatingMax = 5
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

func (a *Appointment) GetStatus() AppointmentStatus {
	return AppointmentStatus(a.Status)
}

func (a *Appointment) SetStatus(s AppointmentStatus) {
	a.Status = s.String()
}

func (a *Appointment) CanCancel() bool {
	s := a.GetStatus()
	return s == AppointmentStatusPending || s == AppointmentStatusConfirmed
}

func (a *Appointment) CanComplete() bool {
	s := a.GetStatus()
	return s == AppointmentStatusPending || s == AppointmentStatusConfirmed
}

func (a *Appointment) CanReview() bool {
	return a.GetStatus() == AppointmentStatusCompleted
}

func (a *Appointment) BelongsToPatient(patientID uint) bool {
	return a.PatientID == patientID
}

func (a *Appointment) BelongsToDoctor(doctorID uint) bool {
	return a.DoctorID == doctorID
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

func (r *Review) ValidateRating() error {
	if r.Rating < ReviewRatingMin || r.Rating > ReviewRatingMax {
		return fmt.Errorf("评分必须在 %d-%d 之间", ReviewRatingMin, ReviewRatingMax)
	}
	return nil
}

type DoctorRating struct {
	DoctorID    uint            `json:"doctor_id"`
	AvgRating   float64         `json:"avg_rating"`
	TotalCount  int64           `json:"total_count"`
	RatingCount map[int]int64   `json:"rating_count"`
}

type ReviewDetail struct {
	ID          uint   `json:"id"`
	PatientName string `json:"patient_name"`
	Rating      int    `json:"rating"`
	Comment     string `json:"comment"`
	CreatedAt   string `json:"created_at"`
}

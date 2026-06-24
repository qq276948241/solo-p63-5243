package user

import (
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

const (
	RolePatient = "patient"
	RoleDoctor  = "doctor"
)

type User struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	Username  string         `gorm:"size:50;uniqueIndex;not null" json:"username"`
	Password  string         `gorm:"size:255;not null" json:"-"`
	RealName  string         `gorm:"size:50;not null" json:"real_name"`
	Role      string         `gorm:"size:20;not null;index" json:"role"`
	Phone     string         `gorm:"size:20" json:"phone"`
	Gender    string         `gorm:"size:10" json:"gender"`
	Age       int            `json:"age"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

type DoctorProfile struct {
	ID           uint           `gorm:"primaryKey" json:"id"`
	UserID       uint           `gorm:"not null;uniqueIndex" json:"user_id"`
	Department   string         `gorm:"size:100;not null" json:"department"`
	Title        string         `gorm:"size:50;not null" json:"title"`
	Introduction string         `gorm:"type:text" json:"introduction"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
	User         User           `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

func (u *User) HashPassword() error {
	hashedBytes, err := bcrypt.GenerateFromPassword([]byte(u.Password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	u.Password = string(hashedBytes)
	return nil
}

func (u *User) CheckPassword(password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(password))
	return err == nil
}

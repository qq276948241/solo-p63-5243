package user

import (
	"clinic/internal/common"
	"errors"

	"gorm.io/gorm"
)

type RegisterRequest struct {
	Username string `json:"username" binding:"required,min=3,max=50"`
	Password string `json:"password" binding:"required,min=6,max=50"`
	RealName string `json:"real_name" binding:"required"`
	Role     string `json:"role" binding:"required,oneof=patient doctor"`
	Phone    string `json:"phone"`
	Gender   string `json:"gender"`
	Age      int    `json:"age"`
}

type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type LoginResponse struct {
	Token string `json:"token"`
	User  *User  `json:"user"`
}

type DoctorInfo struct {
	ID           uint   `json:"id"`
	RealName     string `json:"real_name"`
	Department   string `json:"department"`
	Title        string `json:"title"`
	Introduction string `json:"introduction"`
}

type Service struct {
	db *gorm.DB
}

func NewService(db *gorm.DB) *Service {
	return &Service{db: db}
}

func (s *Service) Register(req *RegisterRequest) (*User, error) {
	var count int64
	s.db.Model(&User{}).Where("username = ?", req.Username).Count(&count)
	if count > 0 {
		return nil, errors.New("用户名已存在")
	}

	user := &User{
		Username: req.Username,
		Password: req.Password,
		RealName: req.RealName,
		Role:     req.Role,
		Phone:    req.Phone,
		Gender:   req.Gender,
		Age:      req.Age,
	}

	if err := user.HashPassword(); err != nil {
		return nil, err
	}

	if err := s.db.Create(user).Error; err != nil {
		return nil, err
	}

	return user, nil
}

func (s *Service) Login(req *LoginRequest) (*LoginResponse, error) {
	var user User
	if err := s.db.Where("username = ?", req.Username).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("用户名或密码错误")
		}
		return nil, err
	}

	if !user.CheckPassword(req.Password) {
		return nil, errors.New("用户名或密码错误")
	}

	token, err := common.GenerateToken(user.ID, user.Username, user.Role)
	if err != nil {
		return nil, err
	}

	return &LoginResponse{
		Token: token,
		User:  &user,
	}, nil
}

func (s *Service) GetUserByID(id uint) (*User, error) {
	var user User
	if err := s.db.First(&user, id).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (s *Service) GetDoctorList() ([]DoctorInfo, error) {
	var doctors []DoctorInfo
	err := s.db.Table("users").
		Select("users.id, users.real_name, doctor_profiles.department, doctor_profiles.title, doctor_profiles.introduction").
		Joins("LEFT JOIN doctor_profiles ON users.id = doctor_profiles.user_id").
		Where("users.role = ?", RoleDoctor).
		Scan(&doctors).Error
	return doctors, err
}

func (s *Service) CreateDoctorProfile(profile *DoctorProfile) error {
	return s.db.Create(profile).Error
}

func (s *Service) GetDoctorProfile(userID uint) (*DoctorProfile, error) {
	var profile DoctorProfile
	if err := s.db.Where("user_id = ?", userID).First(&profile).Error; err != nil {
		return nil, err
	}
	return &profile, nil
}

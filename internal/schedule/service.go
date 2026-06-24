package schedule

import (
	"errors"
	"time"

	"gorm.io/gorm"
)

type CreateScheduleRequest struct {
	DoctorID  uint   `json:"doctor_id" binding:"required"`
	Date      string `json:"date" binding:"required"`
	StartTime string `json:"start_time" binding:"required"`
	EndTime   string `json:"end_time" binding:"required"`
	MaxCount  int    `json:"max_count" binding:"min=1"`
}

type QueryScheduleRequest struct {
	DoctorID uint   `form:"doctor_id"`
	Date     string `form:"date"`
	StartDate string `form:"start_date"`
	EndDate   string `form:"end_date"`
}

type CancelScheduleRequest struct {
	ScheduleID uint `json:"schedule_id" binding:"required"`
}

type Service struct {
	db *gorm.DB
}

func NewService(db *gorm.DB) *Service {
	return &Service{db: db}
}

func (s *Service) CreateSchedule(req *CreateScheduleRequest) (*Schedule, error) {
	var existing Schedule
	err := s.db.Where("doctor_id = ? AND date = ? AND start_time = ? AND end_time = ?",
		req.DoctorID, req.Date, req.StartTime, req.EndTime).First(&existing).Error
	if err == nil {
		return nil, errors.New("该时段已存在排班")
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	schedule := &Schedule{
		DoctorID:  req.DoctorID,
		Date:      req.Date,
		StartTime: req.StartTime,
		EndTime:   req.EndTime,
		MaxCount:  req.MaxCount,
		Booked:    0,
		Status:    StatusAvailable,
	}

	if schedule.MaxCount == 0 {
		schedule.MaxCount = 1
	}

	if err := s.db.Create(schedule).Error; err != nil {
		return nil, err
	}

	return schedule, nil
}

func (s *Service) GetScheduleList(req *QueryScheduleRequest) ([]ScheduleDetail, error) {
	var schedules []ScheduleDetail

	query := s.db.Table("schedules").
		Select("schedules.id, schedules.doctor_id, users.real_name as doctor_name, doctor_profiles.department, doctor_profiles.title, schedules.date, schedules.start_time, schedules.end_time, schedules.max_count, schedules.booked, schedules.status").
		Joins("LEFT JOIN users ON schedules.doctor_id = users.id").
		Joins("LEFT JOIN doctor_profiles ON users.id = doctor_profiles.user_id")

	if req.DoctorID > 0 {
		query = query.Where("schedules.doctor_id = ?", req.DoctorID)
	}
	if req.Date != "" {
		query = query.Where("schedules.date = ?", req.Date)
	}
	if req.StartDate != "" {
		query = query.Where("schedules.date >= ?", req.StartDate)
	}
	if req.EndDate != "" {
		query = query.Where("schedules.date <= ?", req.EndDate)
	}

	query = query.Where("schedules.status != ?", StatusCanceled).
		Order("schedules.date ASC, schedules.start_time ASC")

	err := query.Scan(&schedules).Error
	return schedules, err
}

func (s *Service) GetDoctorScheduleList(doctorID uint, date string) ([]ScheduleDetail, error) {
	var schedules []ScheduleDetail

	query := s.db.Table("schedules").
		Select("schedules.id, schedules.doctor_id, users.real_name as doctor_name, doctor_profiles.department, doctor_profiles.title, schedules.date, schedules.start_time, schedules.end_time, schedules.max_count, schedules.booked, schedules.status").
		Joins("LEFT JOIN users ON schedules.doctor_id = users.id").
		Joins("LEFT JOIN doctor_profiles ON users.id = doctor_profiles.user_id").
		Where("schedules.doctor_id = ?", doctorID)

	if date != "" {
		query = query.Where("schedules.date = ?", date)
	} else {
		today := time.Now().Format("2006-01-02")
		query = query.Where("schedules.date = ?", today)
	}

	err := query.Order("schedules.start_time ASC").Scan(&schedules).Error
	return schedules, err
}

func (s *Service) CancelSchedule(scheduleID uint, doctorID uint) error {
	var schedule Schedule
	if err := s.db.First(&schedule, scheduleID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("排班不存在")
		}
		return err
	}

	if schedule.DoctorID != doctorID {
		return errors.New("无权取消他人排班")
	}

	if schedule.Status == StatusCanceled {
		return errors.New("该排班已取消")
	}

	return s.db.Model(&schedule).Update("status", StatusCanceled).Error
}

func (s *Service) GetScheduleByID(id uint) (*Schedule, error) {
	var schedule Schedule
	if err := s.db.First(&schedule, id).Error; err != nil {
		return nil, err
	}
	return &schedule, nil
}

func (s *Service) UpdateBookedCount(scheduleID uint, delta int) error {
	return s.db.Model(&Schedule{}).Where("id = ?", scheduleID).
		UpdateColumn("booked", gorm.Expr("booked + ?", delta)).Error
}

func (s *Service) CheckConflict(doctorID uint, date string, startTime string, endTime string, excludeID uint) (bool, error) {
	var count int64
	query := s.db.Model(&Schedule{}).
		Where("doctor_id = ? AND date = ? AND status = ?", doctorID, date, StatusAvailable).
		Where("((start_time < ? AND end_time > ?) OR (start_time < ? AND end_time > ?) OR (start_time >= ? AND end_time <= ?))",
			endTime, startTime, endTime, startTime, startTime, endTime)

	if excludeID > 0 {
		query = query.Where("id != ?", excludeID)
	}

	err := query.Count(&count).Error
	return count > 0, err
}

func (s *Service) BatchCreateSchedules(doctorID uint, dates []string, timeSlots []struct {
	Start string `json:"start"`
	End   string `json:"end"`
}, maxCount int) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		for _, date := range dates {
			for _, slot := range timeSlots {
				schedule := &Schedule{
					DoctorID:  doctorID,
					Date:      date,
					StartTime: slot.Start,
					EndTime:   slot.End,
					MaxCount:  maxCount,
					Booked:    0,
					Status:    StatusAvailable,
				}
				if err := tx.Create(schedule).Error; err != nil {
					return err
				}
			}
		}
		return nil
	})
}

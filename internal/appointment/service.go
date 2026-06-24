package appointment

import (
	"clinic/internal/schedule"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
)

type CreateAppointmentRequest struct {
	ScheduleID uint   `json:"schedule_id" binding:"required"`
	Symptoms   string `json:"symptoms"`
}

type QueryAppointmentRequest struct {
	Status   string `form:"status"`
	Date     string `form:"date"`
	DoctorID uint   `form:"doctor_id"`
}

type UpdateStatusRequest struct {
	Status string `json:"status" binding:"required,oneof=completed"`
	Remark string `json:"remark"`
}

type CreateReviewRequest struct {
	Rating  int    `json:"rating" binding:"required,min=1,max=5"`
	Comment string `json:"comment"`
}

type Service struct {
	db              *gorm.DB
	scheduleService *schedule.Service
}

func NewService(db *gorm.DB, scheduleService *schedule.Service) *Service {
	return &Service{
		db:              db,
		scheduleService: scheduleService,
	}
}

func (s *Service) CreateAppointment(patientID uint, req *CreateAppointmentRequest) (*Appointment, error) {
	sched, err := s.scheduleService.GetScheduleByID(req.ScheduleID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("排班不存在")
		}
		return nil, err
	}

	if sched.Status == schedule.StatusCanceled {
		return nil, errors.New("该时段已停诊，无法预约")
	}

	if sched.Booked >= sched.MaxCount {
		return nil, errors.New("该时段已约满")
	}

	hasConflict, err := s.checkTimeConflict(sched.DoctorID, sched.Date, sched.StartTime, sched.EndTime, 0)
	if err != nil {
		return nil, err
	}
	if hasConflict {
		return nil, errors.New("该时段已被预约，请选择其他时段")
	}

	hasPatientConflict, err := s.checkPatientConflict(patientID, sched.Date, sched.StartTime, sched.EndTime)
	if err != nil {
		return nil, err
	}
	if hasPatientConflict {
		return nil, errors.New("您在该时段已有预约")
	}

	tx := s.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	appointment := &Appointment{
		PatientID:  patientID,
		DoctorID:   sched.DoctorID,
		ScheduleID: req.ScheduleID,
		Date:       sched.Date,
		StartTime:  sched.StartTime,
		EndTime:    sched.EndTime,
		Status:     StatusConfirmed,
		Symptoms:   req.Symptoms,
	}

	if err := tx.Create(appointment).Error; err != nil {
		tx.Rollback()
		return nil, err
	}

	if err := tx.Model(&schedule.Schedule{}).Where("id = ?", req.ScheduleID).
		UpdateColumn("booked", gorm.Expr("booked + 1")).Error; err != nil {
		tx.Rollback()
		return nil, err
	}

	if err := tx.Commit().Error; err != nil {
		return nil, err
	}

	return appointment, nil
}

func (s *Service) checkTimeConflict(doctorID uint, date string, startTime string, endTime string, excludeID uint) (bool, error) {
	var count int64
	query := s.db.Model(&Appointment{}).
		Where("doctor_id = ? AND date = ? AND status IN (?, ?)", doctorID, date, StatusPending, StatusConfirmed).
		Where("((start_time < ? AND end_time > ?) OR (start_time < ? AND end_time > ?) OR (start_time >= ? AND end_time <= ?))",
			endTime, startTime, endTime, startTime, startTime, endTime)

	if excludeID > 0 {
		query = query.Where("id != ?", excludeID)
	}

	err := query.Count(&count).Error
	return count > 0, err
}

func (s *Service) checkPatientConflict(patientID uint, date string, startTime string, endTime string) (bool, error) {
	var count int64
	err := s.db.Model(&Appointment{}).
		Where("patient_id = ? AND date = ? AND status IN (?, ?)", patientID, date, StatusPending, StatusConfirmed).
		Where("((start_time < ? AND end_time > ?) OR (start_time < ? AND end_time > ?) OR (start_time >= ? AND end_time <= ?))",
			endTime, startTime, endTime, startTime, startTime, endTime).
		Count(&count).Error
	return count > 0, err
}

func (s *Service) GetPatientAppointments(patientID uint, req *QueryAppointmentRequest) ([]AppointmentDetail, error) {
	var appointments []AppointmentDetail

	query := s.buildDetailQuery().Where("appointments.patient_id = ?", patientID)

	if req.Status != "" {
		query = query.Where("appointments.status = ?", req.Status)
	}
	if req.Date != "" {
		query = query.Where("appointments.date = ?", req.Date)
	}

	err := query.Order("appointments.date DESC, appointments.start_time DESC").Scan(&appointments).Error
	return appointments, err
}

func (s *Service) GetDoctorAppointments(doctorID uint, req *QueryAppointmentRequest) ([]AppointmentDetail, error) {
	var appointments []AppointmentDetail

	query := s.buildDetailQuery().Where("appointments.doctor_id = ?", doctorID)

	if req.Status != "" {
		query = query.Where("appointments.status = ?", req.Status)
	}
	if req.Date != "" {
		query = query.Where("appointments.date = ?", req.Date)
	} else {
		today := time.Now().Format("2006-01-02")
		query = query.Where("appointments.date = ?", today)
	}

	err := query.Order("appointments.start_time ASC").Scan(&appointments).Error
	return appointments, err
}

func (s *Service) buildDetailQuery() *gorm.DB {
	return s.db.Table("appointments").
		Select(`
			appointments.id,
			appointments.patient_id,
			p.real_name as patient_name,
			p.phone as patient_phone,
			p.gender as patient_gender,
			p.age as patient_age,
			appointments.doctor_id,
			d.real_name as doctor_name,
			dp.department,
			dp.title,
			appointments.schedule_id,
			appointments.date,
			appointments.start_time,
			appointments.end_time,
			appointments.status,
			appointments.symptoms,
			appointments.remark,
			appointments.created_at
		`).
		Joins("LEFT JOIN users p ON appointments.patient_id = p.id").
		Joins("LEFT JOIN users d ON appointments.doctor_id = d.id").
		Joins("LEFT JOIN doctor_profiles dp ON d.id = dp.user_id")
}

func (s *Service) CancelAppointment(appointmentID uint, patientID uint) error {
	var appointment Appointment
	if err := s.db.First(&appointment, appointmentID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("预约不存在")
		}
		return err
	}

	if appointment.PatientID != patientID {
		return errors.New("无权取消他人预约")
	}

	if appointment.Status == StatusCanceled {
		return errors.New("该预约已取消")
	}

	if appointment.Status == StatusCompleted {
		return errors.New("已完成的预约无法取消")
	}

	tx := s.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	if err := tx.Model(&appointment).Update("status", StatusCanceled).Error; err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Model(&schedule.Schedule{}).Where("id = ?", appointment.ScheduleID).
		UpdateColumn("booked", gorm.Expr("GREATEST(booked - 1, 0)")).Error; err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}

func (s *Service) CompleteAppointment(appointmentID uint, doctorID uint, remark string) error {
	var appointment Appointment
	if err := s.db.First(&appointment, appointmentID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("预约不存在")
		}
		return err
	}

	if appointment.DoctorID != doctorID {
		return errors.New("无权操作他人预约")
	}

	if appointment.Status != StatusConfirmed && appointment.Status != StatusPending {
		return errors.New(fmt.Sprintf("当前状态为%s，无法标记完成", appointment.Status))
	}

	updates := map[string]interface{}{
		"status": StatusCompleted,
	}
	if remark != "" {
		updates["remark"] = remark
	}

	return s.db.Model(&appointment).Updates(updates).Error
}

func (s *Service) GetAppointmentByID(id uint) (*Appointment, error) {
	var appointment Appointment
	if err := s.db.First(&appointment, id).Error; err != nil {
		return nil, err
	}
	return &appointment, nil
}

func (s *Service) CreateReview(appointmentID uint, patientID uint, req *CreateReviewRequest) (*Review, error) {
	var appointment Appointment
	if err := s.db.First(&appointment, appointmentID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("预约不存在")
		}
		return nil, err
	}

	if appointment.PatientID != patientID {
		return nil, errors.New("无权评价他人预约")
	}

	if appointment.Status != StatusCompleted {
		return nil, errors.New("仅已完成的预约可评价")
	}

	var existing Review
	err := s.db.Where("appointment_id = ?", appointmentID).First(&existing).Error
	if err == nil {
		return nil, errors.New("该预约已评价过")
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	review := &Review{
		AppointmentID: appointmentID,
		PatientID:     patientID,
		DoctorID:      appointment.DoctorID,
		Rating:        req.Rating,
		Comment:       req.Comment,
	}

	if err := s.db.Create(review).Error; err != nil {
		return nil, err
	}

	return review, nil
}

func (s *Service) GetDoctorRating(doctorID uint) (*DoctorRating, error) {
	var result struct {
		AvgRating  float64
		TotalCount int64
	}

	err := s.db.Model(&Review{}).
		Where("doctor_id = ?", doctorID).
		Select("AVG(rating) as avg_rating, COUNT(*) as total_count").
		Scan(&result).Error
	if err != nil {
		return nil, err
	}

	ratingCount := make(map[int]int64)
	for i := 1; i <= 5; i++ {
		var count int64
		s.db.Model(&Review{}).Where("doctor_id = ? AND rating = ?", doctorID, i).Count(&count)
		ratingCount[i] = count
	}

	return &DoctorRating{
		DoctorID:    doctorID,
		AvgRating:   result.AvgRating,
		TotalCount:  result.TotalCount,
		RatingCount: ratingCount,
	}, nil
}

func (s *Service) GetDoctorReviews(doctorID uint) ([]ReviewDetail, error) {
	var reviews []ReviewDetail
	err := s.db.Table("reviews").
		Select("reviews.id, users.real_name as patient_name, reviews.rating, reviews.comment, reviews.created_at").
		Joins("LEFT JOIN users ON reviews.patient_id = users.id").
		Where("reviews.doctor_id = ?", doctorID).
		Order("reviews.created_at DESC").
		Scan(&reviews).Error
	return reviews, err
}

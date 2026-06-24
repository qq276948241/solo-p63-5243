package appointment

import (
	"clinic/internal/schedule"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
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

type CompleteAppointmentRequest struct {
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

// ==================== 预约状态变更方法 ====================

func (s *Service) getAppointment(id uint) (*Appointment, error) {
	var apt Appointment
	if err := s.db.First(&apt, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("预约不存在")
		}
		return nil, err
	}
	return &apt, nil
}

func getAppointmentForUpdate(tx *gorm.DB, id uint) (*Appointment, error) {
	var apt Appointment
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&apt, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("预约不存在")
		}
		return nil, err
	}
	return &apt, nil
}

// ==================== 预约核心方法 ====================

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

	hasConflict, err := s.checkDoctorTimeConflict(sched.DoctorID, sched.Date, sched.StartTime, sched.EndTime, 0)
	if err != nil {
		return nil, err
	}
	if hasConflict {
		return nil, errors.New("该时段已被预约，请选择其他时段")
	}

	hasPatientConflict, err := s.checkPatientTimeConflict(patientID, sched.Date, sched.StartTime, sched.EndTime)
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

	apt := &Appointment{
		PatientID:  patientID,
		DoctorID:   sched.DoctorID,
		ScheduleID: req.ScheduleID,
		Date:       sched.Date,
		StartTime:  sched.StartTime,
		EndTime:    sched.EndTime,
		Symptoms:   req.Symptoms,
	}
	apt.SetStatus(AppointmentStatusConfirmed)

	if err := tx.Create(apt).Error; err != nil {
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

	return apt, nil
}

func (s *Service) CancelAppointment(appointmentID uint, patientID uint) error {
	tx := s.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	apt, err := getAppointmentForUpdate(tx, appointmentID)
	if err != nil {
		tx.Rollback()
		return err
	}

	if !apt.BelongsToPatient(patientID) {
		tx.Rollback()
		return errors.New("无权取消他人预约")
	}

	if !apt.CanCancel() {
		tx.Rollback()
		return fmt.Errorf("当前预约状态为%s，无法取消", apt.GetStatus())
	}

	if err := tx.Model(apt).Update("status", AppointmentStatusCanceled.String()).Error; err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Model(&schedule.Schedule{}).Where("id = ?", apt.ScheduleID).
		UpdateColumn("booked", gorm.Expr("GREATEST(booked - 1, 0)")).Error; err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}

func (s *Service) CompleteAppointment(appointmentID uint, doctorID uint, remark string) error {
	tx := s.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	apt, err := getAppointmentForUpdate(tx, appointmentID)
	if err != nil {
		tx.Rollback()
		return err
	}

	if !apt.BelongsToDoctor(doctorID) {
		tx.Rollback()
		return errors.New("无权操作他人预约")
	}

	if !apt.CanComplete() {
		tx.Rollback()
		return fmt.Errorf("当前预约状态为%s，无法标记完成", apt.GetStatus())
	}

	updates := map[string]interface{}{
		"status": AppointmentStatusCompleted.String(),
	}
	if remark != "" {
		updates["remark"] = remark
	}

	if err := tx.Model(apt).Updates(updates).Error; err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}

// ==================== 预约查询方法 ====================

func (s *Service) GetAppointmentByID(id uint) (*Appointment, error) {
	return s.getAppointment(id)
}

func (s *Service) GetPatientAppointments(patientID uint, req *QueryAppointmentRequest) ([]AppointmentDetail, error) {
	var list []AppointmentDetail
	query := s.buildDetailQuery().Where("appointments.patient_id = ?", patientID)
	if req.Status != "" {
		query = query.Where("appointments.status = ?", req.Status)
	}
	if req.Date != "" {
		query = query.Where("appointments.date = ?", req.Date)
	}
	err := query.Order("appointments.date DESC, appointments.start_time DESC").Scan(&list).Error
	return list, err
}

func (s *Service) GetDoctorAppointments(doctorID uint, req *QueryAppointmentRequest) ([]AppointmentDetail, error) {
	var list []AppointmentDetail
	query := s.buildDetailQuery().Where("appointments.doctor_id = ?", doctorID)
	if req.Status != "" {
		query = query.Where("appointments.status = ?", req.Status)
	}
	if req.Date != "" {
		query = query.Where("appointments.date = ?", req.Date)
	} else {
		query = query.Where("appointments.date = ?", time.Now().Format("2006-01-02"))
	}
	err := query.Order("appointments.start_time ASC").Scan(&list).Error
	return list, err
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

// ==================== 冲突检测方法 ====================

func (s *Service) checkDoctorTimeConflict(doctorID uint, date string, startTime string, endTime string, excludeID uint) (bool, error) {
	var count int64
	query := s.db.Model(&Appointment{}).
		Where("doctor_id = ? AND date = ? AND status IN ?", doctorID, date,
			[]string{AppointmentStatusPending.String(), AppointmentStatusConfirmed.String()}).
		Where("((start_time < ? AND end_time > ?) OR (start_time < ? AND end_time > ?) OR (start_time >= ? AND end_time <= ?))",
			endTime, startTime, endTime, startTime, startTime, endTime)
	if excludeID > 0 {
		query = query.Where("id != ?", excludeID)
	}
	err := query.Count(&count).Error
	return count > 0, err
}

func (s *Service) checkPatientTimeConflict(patientID uint, date string, startTime string, endTime string) (bool, error) {
	var count int64
	err := s.db.Model(&Appointment{}).
		Where("patient_id = ? AND date = ? AND status IN ?", patientID, date,
			[]string{AppointmentStatusPending.String(), AppointmentStatusConfirmed.String()}).
		Where("((start_time < ? AND end_time > ?) OR (start_time < ? AND end_time > ?) OR (start_time >= ? AND end_time <= ?))",
			endTime, startTime, endTime, startTime, startTime, endTime).
		Count(&count).Error
	return count > 0, err
}

// ==================== 评价方法（独立模块） ====================

func (s *Service) CreateReview(appointmentID uint, patientID uint, req *CreateReviewRequest) (*Review, error) {
	tx := s.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	apt, err := getAppointmentForUpdate(tx, appointmentID)
	if err != nil {
		tx.Rollback()
		return nil, err
	}

	if !apt.BelongsToPatient(patientID) {
		tx.Rollback()
		return nil, errors.New("无权评价他人预约")
	}

	if !apt.CanReview() {
		tx.Rollback()
		return nil, fmt.Errorf("仅已完成的预约可评价，当前状态为%s", apt.GetStatus())
	}

	var existing Review
	if err := tx.Where("appointment_id = ?", appointmentID).First(&existing).Error; err == nil {
		tx.Rollback()
		return nil, errors.New("该预约已评价过")
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		tx.Rollback()
		return nil, err
	}

	review := &Review{
		AppointmentID: appointmentID,
		PatientID:     patientID,
		DoctorID:      apt.DoctorID,
		Rating:        req.Rating,
		Comment:       req.Comment,
	}

	if err := review.ValidateRating(); err != nil {
		tx.Rollback()
		return nil, err
	}

	if err := tx.Create(review).Error; err != nil {
		tx.Rollback()
		return nil, err
	}

	if err := tx.Commit().Error; err != nil {
		return nil, err
	}

	return review, nil
}

func (s *Service) GetReviewByAppointment(appointmentID uint) (*Review, error) {
	var review Review
	if err := s.db.Where("appointment_id = ?", appointmentID).First(&review).Error; err != nil {
		return nil, err
	}
	return &review, nil
}

func (s *Service) GetDoctorRating(doctorID uint) (*DoctorRating, error) {
	var agg struct {
		AvgRating  float64
		TotalCount int64
	}
	err := s.db.Model(&Review{}).
		Where("doctor_id = ?", doctorID).
		Select("IFNULL(AVG(rating), 0) as avg_rating, COUNT(*) as total_count").
		Scan(&agg).Error
	if err != nil {
		return nil, err
	}

	ratingCount := make(map[int]int64)
	for i := ReviewRatingMin; i <= ReviewRatingMax; i++ {
		var count int64
		s.db.Model(&Review{}).Where("doctor_id = ? AND rating = ?", doctorID, i).Count(&count)
		ratingCount[i] = count
	}

	return &DoctorRating{
		DoctorID:    doctorID,
		AvgRating:   agg.AvgRating,
		TotalCount:  agg.TotalCount,
		RatingCount: ratingCount,
	}, nil
}

func (s *Service) GetDoctorReviews(doctorID uint) ([]ReviewDetail, error) {
	var list []ReviewDetail
	err := s.db.Table("reviews").
		Select("reviews.id, users.real_name as patient_name, reviews.rating, reviews.comment, reviews.created_at").
		Joins("LEFT JOIN users ON reviews.patient_id = users.id").
		Where("reviews.doctor_id = ?", doctorID).
		Order("reviews.created_at DESC").
		Scan(&list).Error
	return list, err
}

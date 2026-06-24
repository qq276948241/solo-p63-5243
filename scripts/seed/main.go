package main

import (
	"clinic/internal/common"
	"clinic/internal/schedule"
	"clinic/internal/user"
	"fmt"
	"log"
	"time"

	"gorm.io/gorm"
)

type SeedDoctor struct {
	Username     string
	Password     string
	RealName     string
	Phone        string
	Department   string
	Title        string
	Introduction string
}

func main() {
	if err := common.LoadConfig(); err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	if err := common.InitDB(); err != nil {
		log.Fatalf("Failed to init database: %v", err)
	}

	if err := common.AutoMigrate(
		&user.User{},
		&user.DoctorProfile{},
		&schedule.Schedule{},
	); err != nil {
		log.Fatalf("Failed to migrate database: %v", err)
	}

	doctors := []SeedDoctor{
		{
			Username:     "zhangyisheng",
			Password:     "123456",
			RealName:     "张医生",
			Phone:        "13800138001",
			Department:   "内科",
			Title:        "主任医师",
			Introduction: "从事内科临床工作20年，擅长心血管疾病、高血压、糖尿病等慢性病诊治。",
		},
		{
			Username:     "liyisheng",
			Password:     "123456",
			RealName:     "李医生",
			Phone:        "13800138002",
			Department:   "外科",
			Title:        "副主任医师",
			Introduction: "擅长普外科常见疾病诊治，对阑尾炎、疝气、痔疮等手术经验丰富。",
		},
		{
			Username:     "wangyisheng",
			Password:     "123456",
			RealName:     "王医生",
			Phone:        "13800138003",
			Department:   "儿科",
			Title:        "主治医师",
			Introduction: "专注儿科临床10年，擅长小儿呼吸系统、消化系统疾病诊治。",
		},
		{
			Username:     "zhaoyisheng",
			Password:     "123456",
			RealName:     "赵医生",
			Phone:        "13800138004",
			Department:   "妇产科",
			Title:        "副主任医师",
			Introduction: "擅长妇科常见病、孕期保健、产后康复等。",
		},
	}

	for _, d := range doctors {
		if err := seedDoctor(common.DB, d); err != nil {
			log.Printf("Warning: %v", err)
		} else {
			log.Printf("Successfully seeded doctor: %s", d.RealName)
		}
	}

	if err := seedPatient(common.DB); err != nil {
		log.Printf("Warning: %v", err)
	} else {
		log.Println("Successfully seeded test patient")
	}

	var seededDoctors []user.User
	common.DB.Where("role = ?", user.RoleDoctor).Find(&seededDoctors)

	timeSlots := []struct {
		Start string
		End   string
	}{
		{"08:00", "08:30"},
		{"08:30", "09:00"},
		{"09:00", "09:30"},
		{"09:30", "10:00"},
		{"10:00", "10:30"},
		{"10:30", "11:00"},
		{"14:00", "14:30"},
		{"14:30", "15:00"},
		{"15:00", "15:30"},
		{"15:30", "16:00"},
		{"16:00", "16:30"},
		{"16:30", "17:00"},
	}

	dates := generateNextDays(7)

	scheduleService := schedule.NewService(common.DB)

	for _, doctor := range seededDoctors {
		for _, date := range dates {
			for _, slot := range timeSlots {
				req := &schedule.CreateScheduleRequest{
					DoctorID:  doctor.ID,
					Date:      date,
					StartTime: slot.Start,
					EndTime:   slot.End,
					MaxCount:  1,
				}
				_, err := scheduleService.CreateSchedule(req)
				if err != nil {
					continue
				}
			}
		}
		log.Printf("Successfully seeded schedules for doctor: %s", doctor.RealName)
	}

	log.Println("Seed data completed successfully!")
	log.Println("Test accounts:")
	log.Println("  Doctor: zhangyisheng / 123456")
	log.Println("  Doctor: liyisheng / 123456")
	log.Println("  Doctor: wangyisheng / 123456")
	log.Println("  Doctor: zhaoyisheng / 123456")
	log.Println("  Patient: patient1 / 123456")
}

func seedDoctor(db *gorm.DB, d SeedDoctor) error {
	var count int64
	db.Model(&user.User{}).Where("username = ?", d.Username).Count(&count)
	if count > 0 {
		return fmt.Errorf("doctor %s already exists, skipping", d.Username)
	}

	userModel := &user.User{
		Username: d.Username,
		Password: d.Password,
		RealName: d.RealName,
		Role:     user.RoleDoctor,
		Phone:    d.Phone,
	}
	if err := userModel.HashPassword(); err != nil {
		return err
	}

	return db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(userModel).Error; err != nil {
			return err
		}

		profile := &user.DoctorProfile{
			UserID:       userModel.ID,
			Department:   d.Department,
			Title:        d.Title,
			Introduction: d.Introduction,
		}
		if err := tx.Create(profile).Error; err != nil {
			return err
		}

		return nil
	})
}

func seedPatient(db *gorm.DB) error {
	var count int64
	db.Model(&user.User{}).Where("username = ?", "patient1").Count(&count)
	if count > 0 {
		return fmt.Errorf("patient patient1 already exists, skipping")
	}

	patient := &user.User{
		Username: "patient1",
		Password: "123456",
		RealName: "测试患者",
		Role:     user.RolePatient,
		Phone:    "13900139001",
		Gender:   "男",
		Age:      30,
	}
	if err := patient.HashPassword(); err != nil {
		return err
	}

	return db.Create(patient).Error
}

func generateNextDays(n int) []string {
	var dates []string
	now := time.Now()
	for i := 0; i < n; i++ {
		date := now.AddDate(0, 0, i).Format("2006-01-02")
		dates = append(dates, date)
	}
	return dates
}

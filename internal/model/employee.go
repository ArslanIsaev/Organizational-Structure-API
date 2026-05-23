package model

import "time"

type Employee struct {
	ID           uint       `gorm:"primaryKey;autoIncrement" json:"id"`
	DepartmentID uint       `gorm:"not null;index"           json:"department_id"`
	FullName     string     `gorm:"size:200;not null"        json:"full_name"`
	Position     string     `gorm:"size:200;not null"        json:"position"`
	HiredAt      *time.Time `gorm:"type:date"                json:"hired_at"`
	CreatedAt    time.Time  `gorm:"autoCreateTime"           json:"created_at"`
}

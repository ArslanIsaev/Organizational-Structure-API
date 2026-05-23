package model

import "time"

type Department struct {
	ID        uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	Name      string    `gorm:"size:200;not null"        json:"name"`
	ParentID  *uint     `gorm:"index"                    json:"parent_id"`
	CreatedAt time.Time `gorm:"autoCreateTime"           json:"created_at"`

	Parent    *Department  `gorm:"foreignKey:ParentID"     json:"-"`
	Children  []Department `gorm:"foreignKey:ParentID"     json:"-"`
	Employees []Employee   `gorm:"foreignKey:DepartmentID" json:"-"`
}

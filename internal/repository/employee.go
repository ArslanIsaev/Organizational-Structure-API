package repository

import (
	"errors"

	"github.com/yourname/org-api/internal/model"
	"gorm.io/gorm"
)

type EmployeeRepository struct {
	db *gorm.DB
}

func NewEmployeeRepository(db *gorm.DB) *EmployeeRepository {
	return &EmployeeRepository{db: db}
}

func (r *EmployeeRepository) Create(emp *model.Employee) error {
	return r.db.Create(emp).Error
}

func (r *EmployeeRepository) GetByID(id uint) (*model.Employee, error) {
	var emp model.Employee
	err := r.db.First(&emp, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	return &emp, err
}

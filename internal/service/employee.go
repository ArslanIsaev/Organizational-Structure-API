package service

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/yourname/org-api/internal/model"
	"github.com/yourname/org-api/internal/repository"
)

type EmployeeService struct {
	empRepo  *repository.EmployeeRepository
	deptRepo *repository.DepartmentRepository
}

func NewEmployeeService(empRepo *repository.EmployeeRepository, deptRepo *repository.DepartmentRepository) *EmployeeService {
	return &EmployeeService{empRepo: empRepo, deptRepo: deptRepo}
}

type CreateEmployeeInput struct {
	FullName string
	Position string
	HiredAt  *time.Time
}

func (s *EmployeeService) Create(deptID uint, input CreateEmployeeInput) (*model.Employee, error) {
	if _, err := s.deptRepo.GetByID(deptID); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, fmt.Errorf("%w: department not found", ErrNotFound)
		}
		return nil, err
	}

	fullName := strings.TrimSpace(input.FullName)
	if err := validateNotEmpty(fullName, "full_name"); err != nil {
		return nil, fmt.Errorf("%w: %s", ErrBadRequest, err.Error())
	}

	position := strings.TrimSpace(input.Position)
	if err := validateNotEmpty(position, "position"); err != nil {
		return nil, fmt.Errorf("%w: %s", ErrBadRequest, err.Error())
	}

	emp := &model.Employee{
		DepartmentID: deptID,
		FullName:     fullName,
		Position:     position,
		HiredAt:      input.HiredAt,
	}
	if err := s.empRepo.Create(emp); err != nil {
		return nil, err
	}
	return emp, nil
}

func validateNotEmpty(val, field string) error {
	if val == "" {
		return fmt.Errorf("%s must not be empty", field)
	}
	if len(val) > 200 {
		return fmt.Errorf("%s must not exceed 200 characters", field)
	}
	return nil
}

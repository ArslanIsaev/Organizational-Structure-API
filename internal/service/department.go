package service

import (
	"errors"
	"fmt"
	"strings"

	"github.com/yourname/org-api/internal/model"
	"github.com/yourname/org-api/internal/repository"
)

var (
	ErrNotFound   = errors.New("not found")
	ErrConflict   = errors.New("conflict")
	ErrBadRequest = errors.New("bad request")
)

type DepartmentService struct {
	deptRepo *repository.DepartmentRepository
}

func NewDepartmentService(deptRepo *repository.DepartmentRepository) *DepartmentService {
	return &DepartmentService{deptRepo: deptRepo}
}

type CreateDepartmentInput struct {
	Name     string `json:"name"`
	ParentID *uint  `json:"parent_id"`
}

type UpdateDepartmentInput struct {
	Name        *string
	ParentID    *uint
	ClearParent bool
}

type DeleteMode string

const (
	DeleteModeCascade  DeleteMode = "cascade"
	DeleteModeReassign DeleteMode = "reassign"
)

// DepartmentNode рекурсивный ответ с вложенными подразделениями
type DepartmentNode struct {
	model.Department
	Employees []model.Employee  `json:"employees,omitempty"`
	Children  []*DepartmentNode `json:"children,omitempty"`
}

func (s *DepartmentService) Create(input CreateDepartmentInput) (*model.Department, error) {
	name := strings.TrimSpace(input.Name)
	if err := validateName(name); err != nil {
		return nil, fmt.Errorf("%w: %s", ErrBadRequest, err.Error())
	}

	if input.ParentID != nil {
		if _, err := s.deptRepo.GetByID(*input.ParentID); err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				return nil, fmt.Errorf("%w: parent department not found", ErrNotFound)
			}
			return nil, err
		}
	}

	exists, err := s.deptRepo.ExistsNameUnderParent(name, input.ParentID, nil)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, fmt.Errorf("%w: department with this name already exists under the same parent", ErrConflict)
	}

	dept := &model.Department{Name: name, ParentID: input.ParentID}
	if err := s.deptRepo.Create(dept); err != nil {
		return nil, err
	}
	return dept, nil
}

func (s *DepartmentService) GetByID(id uint, depth int, includeEmployees bool, sortBy string) (*DepartmentNode, error) {
	dept, err := s.deptRepo.GetByID(id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return s.buildNode(dept, depth, includeEmployees, sortBy)
}

func (s *DepartmentService) buildNode(dept *model.Department, depth int, includeEmployees bool, sortBy string) (*DepartmentNode, error) {
	node := &DepartmentNode{Department: *dept}

	if includeEmployees {
		emps, err := s.deptRepo.GetEmployees(dept.ID, sortBy)
		if err != nil {
			return nil, err
		}
		node.Employees = emps
	}

	if depth > 0 {
		children, err := s.deptRepo.GetChildren(dept.ID)
		if err != nil {
			return nil, err
		}
		for i := range children {
			child, err := s.buildNode(&children[i], depth-1, includeEmployees, sortBy)
			if err != nil {
				return nil, err
			}
			node.Children = append(node.Children, child)
		}
	}
	return node, nil
}

func (s *DepartmentService) Update(id uint, input UpdateDepartmentInput) (*model.Department, error) {
	dept, err := s.deptRepo.GetByID(id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	if input.Name != nil {
		name := strings.TrimSpace(*input.Name)
		if err := validateName(name); err != nil {
			return nil, fmt.Errorf("%w: %s", ErrBadRequest, err.Error())
		}
		effectiveParent := dept.ParentID
		if input.ClearParent {
			effectiveParent = nil
		} else if input.ParentID != nil {
			effectiveParent = input.ParentID
		}
		exists, err := s.deptRepo.ExistsNameUnderParent(name, effectiveParent, &id)
		if err != nil {
			return nil, err
		}
		if exists {
			return nil, fmt.Errorf("%w: name already exists under same parent", ErrConflict)
		}
		dept.Name = name
	}

	if input.ClearParent {
		dept.ParentID = nil
	} else if input.ParentID != nil {
		newParentID := *input.ParentID
		if newParentID == id {
			return nil, fmt.Errorf("%w: department cannot be its own parent", ErrConflict)
		}
		isDesc, err := s.deptRepo.IsDescendant(id, newParentID)
		if err != nil {
			return nil, err
		}
		if isDesc {
			return nil, fmt.Errorf("%w: moving would create a cycle", ErrConflict)
		}
		if _, err := s.deptRepo.GetByID(newParentID); err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				return nil, fmt.Errorf("%w: parent department not found", ErrNotFound)
			}
			return nil, err
		}
		dept.ParentID = &newParentID
	}

	if err := s.deptRepo.Update(dept); err != nil {
		return nil, err
	}
	return dept, nil
}

func (s *DepartmentService) Delete(id uint, mode DeleteMode, reassignToID *uint) error {
	if _, err := s.deptRepo.GetByID(id); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return ErrNotFound
		}
		return err
	}
	switch mode {
	case DeleteModeCascade:
		return s.deleteCascade(id)
	case DeleteModeReassign:
		if reassignToID == nil {
			return fmt.Errorf("%w: reassign_to_department_id is required", ErrBadRequest)
		}
		return s.deleteReassign(id, *reassignToID)
	default:
		return fmt.Errorf("%w: mode must be 'cascade' or 'reassign'", ErrBadRequest)
	}
}

func (s *DepartmentService) deleteCascade(id uint) error {
	ids, err := s.deptRepo.GetDescendantIDs(id)
	if err != nil {
		return err
	}
	if err := s.deptRepo.DeleteEmployeesByDepartmentIDs(ids); err != nil {
		return err
	}
	return s.deptRepo.DeleteByIDs(ids)
}

func (s *DepartmentService) deleteReassign(id uint, toID uint) error {
	if _, err := s.deptRepo.GetByID(toID); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return fmt.Errorf("%w: reassign target not found", ErrNotFound)
		}
		return err
	}
	ids, err := s.deptRepo.GetDescendantIDs(id)
	if err != nil {
		return err
	}
	if err := s.deptRepo.ReassignEmployees(ids, toID); err != nil {
		return err
	}
	return s.deptRepo.DeleteByIDs(ids)
}

func validateName(name string) error {
	if name == "" {
		return errors.New("name must not be empty")
	}
	if len(name) > 200 {
		return errors.New("name must not exceed 200 characters")
	}
	return nil
}

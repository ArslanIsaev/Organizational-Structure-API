package repository

import (
	"errors"
	"fmt"

	"github.com/yourname/org-api/internal/model"
	"gorm.io/gorm"
)

type DepartmentRepository struct {
	db *gorm.DB
}

func NewDepartmentRepository(db *gorm.DB) *DepartmentRepository {
	return &DepartmentRepository{db: db}
}

func (r *DepartmentRepository) Create(dept *model.Department) error {
	return r.db.Create(dept).Error
}

func (r *DepartmentRepository) GetByID(id uint) (*model.Department, error) {
	var dept model.Department
	err := r.db.First(&dept, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	return &dept, err
}

func (r *DepartmentRepository) Update(dept *model.Department) error {
	return r.db.Save(dept).Error
}

func (r *DepartmentRepository) Delete(id uint) error {
	return r.db.Delete(&model.Department{}, id).Error
}

func (r *DepartmentRepository) ExistsNameUnderParent(name string, parentID *uint, excludeID *uint) (bool, error) {
	query := r.db.Model(&model.Department{}).Where("name = ?", name)
	if parentID == nil {
		query = query.Where("parent_id IS NULL")
	} else {
		query = query.Where("parent_id = ?", *parentID)
	}
	if excludeID != nil {
		query = query.Where("id != ?", *excludeID)
	}
	var count int64
	err := query.Count(&count).Error
	return count > 0, err
}

func (r *DepartmentRepository) IsDescendant(ancestorID, targetID uint) (bool, error) {
	if ancestorID == targetID {
		return true, nil
	}
	query := fmt.Sprintf(`
        WITH RECURSIVE subtree AS (
            SELECT id FROM departments WHERE id = %d
            UNION ALL
            SELECT d.id FROM departments d
            INNER JOIN subtree s ON d.parent_id = s.id
        )
        SELECT COUNT(*) FROM subtree WHERE id = %d
    `, ancestorID, targetID)
	var count int64
	err := r.db.Raw(query).Scan(&count).Error
	return count > 0, err
}

func (r *DepartmentRepository) GetDescendantIDs(id uint) ([]uint, error) {
	query := fmt.Sprintf(`
        WITH RECURSIVE subtree AS (
            SELECT id FROM departments WHERE id = %d
            UNION ALL
            SELECT d.id FROM departments d
            INNER JOIN subtree s ON d.parent_id = s.id
        )
        SELECT id FROM subtree
    `, id)
	var ids []uint
	err := r.db.Raw(query).Scan(&ids).Error
	return ids, err
}

func (r *DepartmentRepository) DeleteByIDs(ids []uint) error {
	return r.db.Where("id IN ?", ids).Delete(&model.Department{}).Error
}

func (r *DepartmentRepository) DeleteEmployeesByDepartmentIDs(ids []uint) error {
	return r.db.Where("department_id IN ?", ids).Delete(&model.Employee{}).Error
}

func (r *DepartmentRepository) ReassignEmployees(fromIDs []uint, toID uint) error {
	return r.db.Model(&model.Employee{}).
		Where("department_id IN ?", fromIDs).
		Update("department_id", toID).Error
}

func (r *DepartmentRepository) GetChildren(parentID uint) ([]model.Department, error) {
	var children []model.Department
	err := r.db.Where("parent_id = ?", parentID).Find(&children).Error
	return children, err
}

func (r *DepartmentRepository) GetEmployees(deptID uint, sortBy string) ([]model.Employee, error) {
	var employees []model.Employee
	order := "created_at ASC"
	if sortBy == "full_name" {
		order = "full_name ASC"
	}
	err := r.db.Where("department_id = ?", deptID).Order(order).Find(&employees).Error
	return employees, err
}

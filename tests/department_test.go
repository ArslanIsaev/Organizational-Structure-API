package tests

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/yourname/org-api/internal/handler"
	"github.com/yourname/org-api/internal/model"
	"github.com/yourname/org-api/internal/repository"
	"github.com/yourname/org-api/internal/service"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func setupDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	db.AutoMigrate(&model.Department{}, &model.Employee{})
	return db
}

func newTestMux(t *testing.T) http.Handler {
	t.Helper()
	db := setupDB(t)
	deptRepo := repository.NewDepartmentRepository(db)
	empRepo := repository.NewEmployeeRepository(db)
	deptSvc := service.NewDepartmentService(deptRepo)
	empSvc := service.NewEmployeeService(empRepo, deptRepo)
	mux := http.NewServeMux()
	handler.RegisterRoutes(mux, handler.NewDepartmentHandler(deptSvc), handler.NewEmployeeHandler(empSvc))
	return mux
}

func TestCreateDepartment_Success(t *testing.T) {
	mux := newTestMux(t)
	req := httptest.NewRequest(http.MethodPost, "/departments/", bytes.NewBufferString(`{"name":"Engineering"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("want 201, got %d: %s", rec.Code, rec.Body)
	}
}

func TestCreateDepartment_EmptyName(t *testing.T) {
	mux := newTestMux(t)
	req := httptest.NewRequest(http.MethodPost, "/departments/", bytes.NewBufferString(`{"name":""}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", rec.Code)
	}
}

func TestCreateDepartment_DuplicateName(t *testing.T) {
	mux := newTestMux(t)
	body := `{"name":"Backend"}`
	for i, want := range []int{http.StatusCreated, http.StatusConflict} {
		req := httptest.NewRequest(http.MethodPost, "/departments/", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		if rec.Code != want {
			t.Fatalf("attempt %d: want %d, got %d", i+1, want, rec.Code)
		}
	}
}

func TestGetDepartment_NotFound(t *testing.T) {
	mux := newTestMux(t)
	req := httptest.NewRequest(http.MethodGet, "/departments/9999", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("want 404, got %d", rec.Code)
	}
}

func TestUpdateDepartment_SelfParent(t *testing.T) {
	mux := newTestMux(t)
	req := httptest.NewRequest(http.MethodPost, "/departments/", bytes.NewBufferString(`{"name":"DevOps"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	var dept map[string]any
	json.NewDecoder(rec.Body).Decode(&dept)
	id := int(dept["id"].(float64))

	body := `{"parent_id":` + itoa(id) + `}`
	req2 := httptest.NewRequest(http.MethodPatch, "/departments/"+itoa(id), bytes.NewBufferString(body))
	req2.Header.Set("Content-Type", "application/json")
	rec2 := httptest.NewRecorder()
	mux.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusConflict {
		t.Fatalf("want 409, got %d: %s", rec2.Code, rec2.Body)
	}
}

func TestDeleteCascade(t *testing.T) {
	mux := newTestMux(t)
	// Create parent
	req := httptest.NewRequest(http.MethodPost, "/departments/", bytes.NewBufferString(`{"name":"Parent"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	var parent map[string]any
	json.NewDecoder(rec.Body).Decode(&parent)
	pid := int(parent["id"].(float64))

	// Create child
	childBody := `{"name":"Child","parent_id":` + itoa(pid) + `}`
	req2 := httptest.NewRequest(http.MethodPost, "/departments/", bytes.NewBufferString(childBody))
	req2.Header.Set("Content-Type", "application/json")
	rec2 := httptest.NewRecorder()
	mux.ServeHTTP(rec2, req2)
	var child map[string]any
	json.NewDecoder(rec2.Body).Decode(&child)
	cid := int(child["id"].(float64))

	// Delete cascade
	req3 := httptest.NewRequest(http.MethodDelete, "/departments/"+itoa(pid)+"?mode=cascade", nil)
	rec3 := httptest.NewRecorder()
	mux.ServeHTTP(rec3, req3)
	if rec3.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", rec3.Code)
	}

	req4 := httptest.NewRequest(http.MethodGet, "/departments/"+itoa(cid), nil)
	rec4 := httptest.NewRecorder()
	mux.ServeHTTP(rec4, req4)
	if rec4.Code != http.StatusNotFound {
		t.Fatalf("child must be deleted, got %d", rec4.Code)
	}
}

func TestCreateEmployee_DeptNotFound(t *testing.T) {
	mux := newTestMux(t)
	body := `{"full_name":"Jane","position":"Dev"}`
	req := httptest.NewRequest(http.MethodPost, "/departments/9999/employees/", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("want 404, got %d", rec.Code)
	}
}

package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/yourname/org-api/internal/service"
)

type DepartmentHandler struct {
	svc *service.DepartmentService
}

func NewDepartmentHandler(svc *service.DepartmentService) *DepartmentHandler {
	return &DepartmentHandler{svc: svc}
}

// POST /departments/
func (h *DepartmentHandler) Create(w http.ResponseWriter, r *http.Request) {
	var input service.CreateDepartmentInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	dept, err := h.svc.Create(input)
	if err != nil {
		handleServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, dept)
}

// GET /departments/{id}
func (h *DepartmentHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, err := extractID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid department id")
		return
	}

	depth := 1
	if d := r.URL.Query().Get("depth"); d != "" {
		depth, err = strconv.Atoi(d)
		if err != nil || depth < 0 {
			writeError(w, http.StatusBadRequest, "depth must be a non-negative integer")
			return
		}
		if depth > 5 {
			depth = 5
		}
	}

	includeEmployees := true
	if ie := r.URL.Query().Get("include_employees"); ie == "false" {
		includeEmployees = false
	}

	sortBy := r.URL.Query().Get("sort_by")

	node, err := h.svc.GetByID(uint(id), depth, includeEmployees, sortBy)
	if err != nil {
		handleServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, node)
}

// PATCH /departments/{id}
func (h *DepartmentHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := extractID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid department id")
		return
	}

	// Используем map[string]json.RawMessage, чтобы различать
	// явный null и отсутствующее поле
	var raw map[string]json.RawMessage
	if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	input := service.UpdateDepartmentInput{}

	if nameRaw, ok := raw["name"]; ok {
		var name string
		if err := json.Unmarshal(nameRaw, &name); err != nil {
			writeError(w, http.StatusBadRequest, "invalid name")
			return
		}
		input.Name = &name
	}

	if parentRaw, ok := raw["parent_id"]; ok {
		if string(parentRaw) == "null" {
			input.ClearParent = true
		} else {
			var pid uint
			if err := json.Unmarshal(parentRaw, &pid); err != nil {
				writeError(w, http.StatusBadRequest, "invalid parent_id")
				return
			}
			input.ParentID = &pid
		}
	}

	dept, err := h.svc.Update(uint(id), input)
	if err != nil {
		handleServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, dept)
}

// DELETE /departments/{id}
func (h *DepartmentHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := extractID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid department id")
		return
	}

	mode := service.DeleteMode(r.URL.Query().Get("mode"))
	var reassignTo *uint
	if rto := r.URL.Query().Get("reassign_to_department_id"); rto != "" {
		v, err := strconv.ParseUint(rto, 10, 64)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid reassign_to_department_id")
			return
		}
		uid := uint(v)
		reassignTo = &uid
	}

	if err := h.svc.Delete(uint(id), mode, reassignTo); err != nil {
		handleServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func extractID(r *http.Request) (int, error) {
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	for i, p := range parts {
		if p == "departments" && i+1 < len(parts) {
			return strconv.Atoi(parts[i+1])
		}
	}
	return 0, errors.New("id not found in path")
}

func handleServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, service.ErrNotFound):
		writeError(w, http.StatusNotFound, err.Error())
	case errors.Is(err, service.ErrConflict):
		writeError(w, http.StatusConflict, err.Error())
	case errors.Is(err, service.ErrBadRequest):
		writeError(w, http.StatusBadRequest, err.Error())
	default:
		writeError(w, http.StatusInternalServerError, "internal server error")
	}
}

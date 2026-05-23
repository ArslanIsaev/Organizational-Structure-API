package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/yourname/org-api/internal/service"
)

type EmployeeHandler struct {
	svc *service.EmployeeService
}

func NewEmployeeHandler(svc *service.EmployeeService) *EmployeeHandler {
	return &EmployeeHandler{svc: svc}
}

// POST /departments/{id}/employees/
func (h *EmployeeHandler) Create(w http.ResponseWriter, r *http.Request) {
	deptID, err := extractDeptIDFromEmployeePath(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid department id")
		return
	}

	var raw map[string]json.RawMessage
	if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	input := service.CreateEmployeeInput{}
	if v, ok := raw["full_name"]; ok {
		_ = json.Unmarshal(v, &input.FullName)
	}
	if v, ok := raw["position"]; ok {
		_ = json.Unmarshal(v, &input.Position)
	}
	if v, ok := raw["hired_at"]; ok && string(v) != "null" {
		var dateStr string
		if err := json.Unmarshal(v, &dateStr); err == nil {
			if parsed, err := time.Parse("2006-01-02", dateStr); err == nil {
				input.HiredAt = &parsed
			}
		}
	}

	emp, err := h.svc.Create(uint(deptID), input)
	if err != nil {
		handleServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, emp)
}

func extractDeptIDFromEmployeePath(r *http.Request) (int, error) {
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	for i, p := range parts {
		if p == "departments" && i+1 < len(parts) {
			return strconv.Atoi(parts[i+1])
		}
	}
	return 0, nil
}

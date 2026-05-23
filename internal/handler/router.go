package handler

import (
	"net/http"
	"strings"
)

func RegisterRoutes(mux *http.ServeMux, deptH *DepartmentHandler, empH *EmployeeHandler) {
	mux.HandleFunc("/departments/", func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimSuffix(r.URL.Path, "/")
		parts := strings.Split(strings.TrimPrefix(path, "/"), "/")

		switch {
		// POST /departments
		case len(parts) == 1 && parts[0] == "departments":
			if r.Method == http.MethodPost {
				deptH.Create(w, r)
			} else {
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			}

		// GET|PATCH|DELETE /departments/{id}
		case len(parts) == 2 && parts[0] == "departments":
			switch r.Method {
			case http.MethodGet:
				deptH.GetByID(w, r)
			case http.MethodPatch:
				deptH.Update(w, r)
			case http.MethodDelete:
				deptH.Delete(w, r)
			default:
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			}

		// POST /departments/{id}/employees
		case len(parts) == 3 && parts[0] == "departments" && parts[2] == "employees":
			if r.Method == http.MethodPost {
				empH.Create(w, r)
			} else {
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			}

		default:
			http.NotFound(w, r)
		}
	})
}

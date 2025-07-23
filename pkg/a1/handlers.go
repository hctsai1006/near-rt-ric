package a1

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/hctsai1006/near-rt-ric/pkg/common/monitoring"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/bcrypt"
)

// APIHandlers contains all A1 REST API handlers
type APIHandlers struct {
	policyManager     *PolicyManager
	modelManager      *MLModelManager
	enrichmentManager *EnrichmentManager
	authService       *AuthService
	logger            *logrus.Logger
	metrics           *monitoring.MetricsCollector
	startTime         time.Time
}

// NewAPIHandlers creates new API handlers
func NewAPIHandlers(policyManager *PolicyManager, modelManager *MLModelManager, enrichmentManager *EnrichmentManager, authService *AuthService, logger *logrus.Logger, metrics *monitoring.MetricsCollector) *APIHandlers {
	return &APIHandlers{
		policyManager:     policyManager,
		modelManager:      modelManager,
		enrichmentManager: enrichmentManager,
		authService:       authService,
		logger:            logger.WithField("component", "a1-handlers"),
		metrics:           metrics,
		startTime:         time.Now(),
	}
}

// SetupRoutes sets up all A1 REST API routes
func (h *APIHandlers) SetupRoutes(r *mux.Router) {
	// Health and status endpoints (no auth required)
	r.HandleFunc("/a1-p/healthcheck", h.HealthCheck).Methods("GET")
	r.HandleFunc("/a1-p/status", h.GetStatus).Methods("GET")

	// Authentication endpoints
	auth := r.PathPrefix("/a1-p/auth").Subrouter()
	auth.HandleFunc("/token", h.GenerateToken).Methods("POST")
	auth.HandleFunc("/refresh", h.RefreshToken).Methods("POST")
	auth.HandleFunc("/revoke", h.RevokeToken).Methods("POST")

	// Policy Type endpoints (require authentication)
	policyTypes := r.PathPrefix("/a1-p/policytypes").Subrouter()
	policyTypes.Use(h.authService.AuthMiddleware())
	
	policyTypes.HandleFunc("", h.GetPolicyTypes).Methods("GET")
	policyTypes.HandleFunc("/{policy_type_id}", h.GetPolicyType).Methods("GET")
	policyTypes.HandleFunc("/{policy_type_id}", h.authService.RequirePermission("policytype:write")(http.HandlerFunc(h.CreatePolicyType))).Methods("PUT")
	policyTypes.HandleFunc("/{policy_type_id}", h.authService.RequirePermission("policytype:delete")(http.HandlerFunc(h.DeletePolicyType))).Methods("DELETE")

	// Policy Instance endpoints (require authentication)
	policies := r.PathPrefix("/a1-p/policytypes/{policy_type_id}/policies").Subrouter()
	policies.Use(h.authService.AuthMiddleware())
	
	policies.HandleFunc("", h.GetPolicyInstances).Methods("GET")
	policies.HandleFunc("/{policy_id}", h.GetPolicyInstance).Methods("GET")
	policies.HandleFunc("/{policy_id}", h.authService.RequirePermission("policy:write")(http.HandlerFunc(h.CreatePolicyInstance))).Methods("PUT")
	policies.HandleFunc("/{policy_id}", h.authService.RequirePermission("policy:delete")(http.HandlerFunc(h.DeletePolicyInstance))).Methods("DELETE")
	policies.HandleFunc("/{policy_id}/status", h.GetPolicyStatus).Methods("GET")

	// Enrichment Information endpoints
	enrichment := r.PathPrefix("/a1-ei").Subrouter()
	enrichment.Use(h.authService.AuthMiddleware())
	
	enrichment.HandleFunc("/eitypes", h.GetEITypes).Methods("GET")
	enrichment.HandleFunc("/eitypes/{ei_type_id}", h.GetEIType).Methods("GET")
	enrichment.HandleFunc("/eijobs", h.GetEIJobs).Methods("GET")
	enrichment.HandleFunc("/eijobs/{ei_job_id}", h.GetEIJob).Methods("GET")
	enrichment.HandleFunc("/eijobs/{ei_job_id}", h.authService.RequirePermission("enrichment:write")(http.HandlerFunc(h.CreateEIJob))).Methods("PUT")
	enrichment.HandleFunc("/eijobs/{ei_job_id}", h.authService.RequirePermission("enrichment:write")(http.HandlerFunc(h.DeleteEIJob))).Methods("DELETE")

	// ML Model Management endpoints
	models := r.PathPrefix("/a1-p/models").Subrouter()
	models.Use(h.authService.AuthMiddleware())
	
	models.HandleFunc("", h.GetMLModels).Methods("GET")
	models.HandleFunc("/{model_id}", h.GetMLModel).Methods("GET")
	models.HandleFunc("/{model_id}", h.authService.RequirePermission("model:write")(http.HandlerFunc(h.DeployMLModel))).Methods("PUT")
	models.HandleFunc("/{model_id}", h.authService.RequirePermission("model:write")(http.HandlerFunc(h.DeleteMLModel))).Methods("DELETE")
}

// HealthCheck returns the health status of the A1 interface
func (h *APIHandlers) HealthCheck(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	defer h.recordMetrics(r, start, http.StatusOK)

	health := &A1HealthCheck{
		Status:    "UP",
		Timestamp: time.Now(),
		Version:   "1.0.0",
		Components: map[string]string{
			"policy_manager": "UP",
			"auth_service":   "UP",
		},
		Uptime: time.Since(h.startTime),
	}

	h.writeJSONResponse(w, http.StatusOK, health)
}

// GetStatus returns detailed status information
func (h *APIHandlers) GetStatus(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	defer h.recordMetrics(r, start, http.StatusOK)

	stats := h.policyManager.GetStatistics()
	stats.Uptime = time.Since(h.startTime)

	h.writeJSONResponse(w, http.StatusOK, stats)
}

// GenerateToken generates a new JWT token
func (h *APIHandlers) GenerateToken(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	var req TokenRequest
	if err := h.decodeJSONRequest(r, &req); err != nil {
		h.recordMetrics(r, start, http.StatusBadRequest)
		h.writeErrorResponse(w, http.StatusBadRequest, "Invalid request body", err.Error())
		return
	}

	// Get user from the database
	user := &User{}
	err := h.policyManager.db.QueryRow(r.Context(), "SELECT id, username, password_hash, email, roles FROM users WHERE username = $1", req.Username).Scan(&user.ID, &user.Username, &user.PasswordHash, &user.Email, &user.Roles)
	if err != nil {
		h.recordMetrics(r, start, http.StatusUnauthorized)
		h.writeErrorResponse(w, http.StatusUnauthorized, "Invalid credentials", "")
		return
	}

	// Compare the provided password with the stored hash
	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password))
	if err != nil {
		h.recordMetrics(r, start, http.StatusUnauthorized)
		h.writeErrorResponse(w, http.StatusUnauthorized, "Invalid credentials", "")
		return
	}

	// Generate token
	token, err := h.authService.GenerateToken(user.ID, user.Username, user.Email, user.Roles)
	if err != nil {
		h.recordMetrics(r, start, http.StatusInternalServerError)
		h.writeErrorResponse(w, http.StatusInternalServerError, "Failed to generate token", err.Error())
		return
	}

	h.recordMetrics(r, start, http.StatusOK)
	h.writeJSONResponse(w, http.StatusOK, token)
}

// RefreshToken refreshes an existing token
func (h *APIHandlers) RefreshToken(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	
	var req RefreshTokenRequest
	if err := h.decodeJSONRequest(r, &req); err != nil {
		h.recordMetrics(r, start, http.StatusBadRequest)
		h.writeErrorResponse(w, http.StatusBadRequest, "Invalid request body", err.Error())
		return
	}

	// For demo purposes, return error - refresh tokens not implemented
	h.recordMetrics(r, start, http.StatusNotImplemented)
	h.writeErrorResponse(w, http.StatusNotImplemented, "Refresh tokens not implemented", "")
}

// RevokeToken revokes a token
func (h *APIHandlers) RevokeToken(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	
	// Get token from Authorization header
	user, ok := GetUserFromContext(r.Context())
	if !ok {
		h.recordMetrics(r, start, http.StatusUnauthorized)
		h.writeErrorResponse(w, http.StatusUnauthorized, "Not authenticated", "")
		return
	}

	// Revoke token
	if err := h.authService.RevokeToken(user.Token); err != nil {
		h.recordMetrics(r, start, http.StatusInternalServerError)
		h.writeErrorResponse(w, http.StatusInternalServerError, "Failed to revoke token", err.Error())
		return
	}

	h.recordMetrics(r, start, http.StatusOK)
	h.writeJSONResponse(w, http.StatusOK, map[string]string{"message": "Token revoked successfully"})
}

// GetPolicyTypes returns all policy types
func (h *APIHandlers) GetPolicyTypes(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	defer h.recordMetrics(r, start, http.StatusOK)

	policyTypes := h.policyManager.GetAllPolicyTypes()
	
	// Extract just the IDs for the list response
	typeIDs := make([]PolicyTypeID, len(policyTypes))
	for i, pt := range policyTypes {
		typeIDs[i] = pt.PolicyTypeID
	}

	h.writeJSONResponse(w, http.StatusOK, typeIDs)
}

// GetPolicyType returns a specific policy type
func (h *APIHandlers) GetPolicyType(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	
	vars := mux.Vars(r)
	policyTypeID := PolicyTypeID(vars["policy_type_id"])

	policyType, err := h.policyManager.GetPolicyType(policyTypeID)
	if err != nil {
		h.recordMetrics(r, start, http.StatusNotFound)
		h.writeErrorResponse(w, http.StatusNotFound, "Policy type not found", err.Error())
		return
	}

	h.recordMetrics(r, start, http.StatusOK)
	h.writeJSONResponse(w, http.StatusOK, policyType)
}

// CreatePolicyType creates a new policy type
func (h *APIHandlers) CreatePolicyType(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	
	vars := mux.Vars(r)
	policyTypeID := PolicyTypeID(vars["policy_type_id"])

	var req PolicyTypeCreateRequest
	if err := h.decodeJSONRequest(r, &req); err != nil {
		h.recordMetrics(r, start, http.StatusBadRequest)
		h.writeErrorResponse(w, http.StatusBadRequest, "Invalid request body", err.Error())
		return
	}

	// Set the policy type ID from URL
	req.PolicyTypeID = policyTypeID

	policyType, err := h.policyManager.CreatePolicyType(&req)
	if err != nil {
		statusCode := http.StatusInternalServerError
		if err.Error() == "policy type already exists" || err.Error() == "maximum number of policy types reached" {
			statusCode = http.StatusConflict
		}
		h.recordMetrics(r, start, statusCode)
		h.writeErrorResponse(w, statusCode, "Failed to create policy type", err.Error())
		return
	}

	h.recordMetrics(r, start, http.StatusCreated)
	h.writeJSONResponse(w, http.StatusCreated, policyType)
}

// DeletePolicyType deletes a policy type
func (h *APIHandlers) DeletePolicyType(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	
	vars := mux.Vars(r)
	policyTypeID := PolicyTypeID(vars["policy_type_id"])

	if err := h.policyManager.DeletePolicyType(policyTypeID); err != nil {
		statusCode := http.StatusInternalServerError
		if err.Error() == "policy type not found" {
			statusCode = http.StatusNotFound
		} else if err.Error() == "active policy instances exist" {
			statusCode = http.StatusConflict
		}
		h.recordMetrics(r, start, statusCode)
		h.writeErrorResponse(w, statusCode, "Failed to delete policy type", err.Error())
		return
	}

	h.recordMetrics(r, start, http.StatusNoContent)
	w.WriteHeader(http.StatusNoContent)
}

// GetPolicyInstances returns all policy instances for a policy type
func (h *APIHandlers) GetPolicyInstances(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	
	vars := mux.Vars(r)
	policyTypeID := PolicyTypeID(vars["policy_type_id"])

	policies := h.policyManager.GetPolicyInstancesByType(policyTypeID)
	
	// Extract just the IDs for the list response
	policyIDs := make([]PolicyID, len(policies))
	for i, policy := range policies {
		policyIDs[i] = policy.PolicyID
	}

	h.recordMetrics(r, start, http.StatusOK)
	h.writeJSONResponse(w, http.StatusOK, policyIDs)
}

// GetPolicyInstance returns a specific policy instance
func (h *APIHandlers) GetPolicyInstance(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	
	vars := mux.Vars(r)
	policyID := PolicyID(vars["policy_id"])

	policy, err := h.policyManager.GetPolicyInstance(policyID)
	if err != nil {
		h.recordMetrics(r, start, http.StatusNotFound)
		h.writeErrorResponse(w, http.StatusNotFound, "Policy instance not found", err.Error())
		return
	}

	h.recordMetrics(r, start, http.StatusOK)
	h.writeJSONResponse(w, http.StatusOK, policy)
}

// CreatePolicyInstance creates a new policy instance
func (h *APIHandlers) CreatePolicyInstance(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	
	vars := mux.Vars(r)
	policyTypeID := PolicyTypeID(vars["policy_type_id"])
	policyID := PolicyID(vars["policy_id"])

	var req PolicyInstanceCreateRequest
	if err := h.decodeJSONRequest(r, &req); err != nil {
		h.recordMetrics(r, start, http.StatusBadRequest)
		h.writeErrorResponse(w, http.StatusBadRequest, "Invalid request body", err.Error())
		return
	}

	// Get user from context
	user, _ := GetUserFromContext(r.Context())
	userID := ""
	if user != nil {
		userID = user.UserID
	}

	policy, err := h.policyManager.CreatePolicyInstance(policyTypeID, policyID, &req, userID)
	if err != nil {
		statusCode := http.StatusInternalServerError
		if err.Error() == "policy type not found" {
			statusCode = http.StatusNotFound
		} else if err.Error() == "policy instance already exists" || err.Error() == "maximum number of policy instances reached" {
			statusCode = http.StatusConflict
		} else if err.Error() == "policy data validation failed" {
			statusCode = http.StatusBadRequest
		}
		h.recordMetrics(r, start, statusCode)
		h.writeErrorResponse(w, statusCode, "Failed to create policy instance", err.Error())
		return
	}

	h.recordMetrics(r, start, http.StatusCreated)
	h.writeJSONResponse(w, http.StatusCreated, policy)
}

// DeletePolicyInstance deletes a policy instance
func (h *APIHandlers) DeletePolicyInstance(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	
	vars := mux.Vars(r)
	policyID := PolicyID(vars["policy_id"])

	// Get user from context
	user, _ := GetUserFromContext(r.Context())
	userID := ""
	if user != nil {
		userID = user.UserID
	}

	if err := h.policyManager.DeletePolicyInstance(policyID, userID); err != nil {
		statusCode := http.StatusInternalServerError
		if err.Error() == "policy instance not found" || err.Error() == "policy instance is already deleted" {
			statusCode = http.StatusNotFound
		}
		h.recordMetrics(r, start, statusCode)
		h.writeErrorResponse(w, statusCode, "Failed to delete policy instance", err.Error())
		return
	}

	h.recordMetrics(r, start, http.StatusNoContent)
	w.WriteHeader(http.StatusNoContent)
}

// GetPolicyStatus returns the status of a policy instance
func (h *APIHandlers) GetPolicyStatus(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	
	vars := mux.Vars(r)
	policyID := PolicyID(vars["policy_id"])

	status, err := h.policyManager.GetPolicyStatus(policyID)
	if err != nil {
		h.recordMetrics(r, start, http.StatusNotFound)
		h.writeErrorResponse(w, http.StatusNotFound, "Policy instance not found", err.Error())
		return
	}

	h.recordMetrics(r, start, http.StatusOK)
	h.writeJSONResponse(w, http.StatusOK, status)
}

// Enrichment Information handlers

func (h *APIHandlers) GetEITypes(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	defer h.recordMetrics(r, start, http.StatusOK)
	// In a real implementation, we would have a way to manage EI types
	h.writeJSONResponse(w, http.StatusOK, []string{"ue_location", "network_performance"})
}

func (h *APIHandlers) GetEIType(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	defer h.recordMetrics(r, start, http.StatusOK)
	// In a real implementation, we would have a way to manage EI types
	vars := mux.Vars(r)
	eiTypeID := vars["ei_type_id"]
	h.writeJSONResponse(w, http.StatusOK, map[string]string{"id": eiTypeID, "name": "UE Location", "description": "User Equipment Location Information"})
}

func (h *APIHandlers) GetEIJobs(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	defer h.recordMetrics(r, start, http.StatusOK)
	jobs := h.enrichmentManager.GetAllEIJobs()
	h.writeJSONResponse(w, http.StatusOK, jobs)
}

func (h *APIHandlers) GetEIJob(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	vars := mux.Vars(r)
	jobID := vars["ei_job_id"]

	job, err := h.enrichmentManager.GetEIJob(jobID)
	if err != nil {
		h.recordMetrics(r, start, http.StatusNotFound)
		h.writeErrorResponse(w, http.StatusNotFound, "EI Job not found", err.Error())
		return
	}

	h.recordMetrics(r, start, http.StatusOK)
	h.writeJSONResponse(w, http.StatusOK, job)
}

func (h *APIHandlers) CreateEIJob(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	var req struct {
		Type  string `json:"type"`
		Owner string `json:"owner"`
	}
	if err := h.decodeJSONRequest(r, &req); err != nil {
		h.recordMetrics(r, start, http.StatusBadRequest)
		h.writeErrorResponse(w, http.StatusBadRequest, "Invalid request body", err.Error())
		return
	}

	job, err := h.enrichmentManager.CreateEIJob(req.Type, req.Owner)
	if err != nil {
		h.recordMetrics(r, start, http.StatusInternalServerError)
		h.writeErrorResponse(w, http.StatusInternalServerError, "Failed to create EI job", err.Error())
		return
	}

	h.recordMetrics(r, start, http.StatusCreated)
	h.writeJSONResponse(w, http.StatusCreated, job)
}

func (h *APIHandlers) DeleteEIJob(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	vars := mux.Vars(r)
	jobID := vars["ei_job_id"]

	if err := h.enrichmentManager.DeleteEIJob(jobID); err != nil {
		h.recordMetrics(r, start, http.StatusNotFound)
		h.writeErrorResponse(w, http.StatusNotFound, "EI Job not found", err.Error())
		return
	}

	h.recordMetrics(r, start, http.StatusNoContent)
	w.WriteHeader(http.StatusNoContent)
}

// ML Model handlers

func (h *APIHandlers) GetMLModels(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	defer h.recordMetrics(r, start, http.StatusOK)
	models := h.modelManager.GetAllModels()
	h.writeJSONResponse(w, http.StatusOK, models)
}

func (h *APIHandlers) GetMLModel(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	vars := mux.Vars(r)
	modelID := vars["model_id"]

	model, err := h.modelManager.GetModel(modelID)
	if err != nil {
		h.recordMetrics(r, start, http.StatusNotFound)
		h.writeErrorResponse(w, http.StatusNotFound, "ML Model not found", err.Error())
		return
	}

	h.recordMetrics(r, start, http.StatusOK)
	h.writeJSONResponse(w, http.StatusOK, model)
}

func (h *APIHandlers) DeployMLModel(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	var req struct {
		Name        string `json:"name"`
		Version     string `json:"version"`
		Description string `json:"description"`
	}
	if err := h.decodeJSONRequest(r, &req); err != nil {
		h.recordMetrics(r, start, http.StatusBadRequest)
		h.writeErrorResponse(w, http.StatusBadRequest, "Invalid request body", err.Error())
		return
	}

	model, err := h.modelManager.DeployModel(req.Name, req.Version, req.Description)
	if err != nil {
		h.recordMetrics(r, start, http.StatusInternalServerError)
		h.writeErrorResponse(w, http.StatusInternalServerError, "Failed to deploy ML model", err.Error())
		return
	}

	h.recordMetrics(r, start, http.StatusAccepted)
	h.writeJSONResponse(w, http.StatusAccepted, model)
}

func (h *APIHandlers) DeleteMLModel(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	vars := mux.Vars(r)
	modelID := vars["model_id"]

	if err := h.modelManager.DeleteModel(modelID); err != nil {
		h.recordMetrics(r, start, http.StatusNotFound)
		h.writeErrorResponse(w, http.StatusNotFound, "ML Model not found", err.Error())
		return
	}

	h.recordMetrics(r, start, http.StatusNoContent)
	w.WriteHeader(http.StatusNoContent)
}

// Helper methods

func (h *APIHandlers) writeJSONResponse(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	
	if err := json.NewEncoder(w).Encode(data); err != nil {
		h.logger.WithError(err).Error("Failed to encode JSON response")
	}
}

func (h *APIHandlers) writeErrorResponse(w http.ResponseWriter, statusCode int, title, detail string) {
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(statusCode)
	
	errorResp := A1ErrorResponse{
		Type:   "https://tools.ietf.org/html/rfc7807",
		Title:  title,
		Status: statusCode,
		Detail: detail,
	}
	
	if err := json.NewEncoder(w).Encode(errorResp); err != nil {
		h.logger.WithError(err).Error("Failed to encode error response")
	}
}

func (h *APIHandlers) decodeJSONRequest(r *http.Request, dst interface{}) error {
	if r.Header.Get("Content-Type") != "application/json" {
		return fmt.Errorf("content-type must be application/json")
	}
	
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	
	return decoder.Decode(dst)
}

func (h *APIHandlers) recordMetrics(r *http.Request, start time.Time, statusCode int) {
	duration := time.Since(start)
	method := r.Method
	path := r.URL.Path
	
	// Record metrics
	h.metrics.RecordA1Request(method, path, statusCode, duration, 0, 0)
	
	// Log request
	h.logger.WithFields(logrus.Fields{
		"method":      method,
		"path":        path,
		"status_code": statusCode,
		"duration":    duration,
		"remote_addr": r.RemoteAddr,
	}).Info("A1 API request processed")
}
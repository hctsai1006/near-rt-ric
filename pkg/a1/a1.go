package a1

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/hctsai1006/near-rt-ric/internal/config"
	"github.com/sirupsen/logrus"
)

// A1Interface provides the main entry point for the A1 interface.
type A1Interface struct {
	logger     *logrus.Logger
	repository A1Repository
	validator  A1PolicyValidator
	server     *http.Server
	router     *mux.Router
	config     *config.A1Config
}

// NewA1Interface creates a new A1 interface.
func NewA1Interface(cfg *config.A1Config, logger *logrus.Logger, repo A1Repository, validator A1PolicyValidator) *A1Interface {
	router := mux.NewRouter()
	server := &http.Server{
		Addr:    fmt.Sprintf("%s:%d", cfg.ListenAddress, cfg.ListenPort),
		Handler: router,
	}
	return &A1Interface{
		logger:     logger,
		repository: repo,
		validator:  validator,
		server:     server,
		router:     router,
		config:     cfg,
	}
}

// Start starts the A1 interface.
func (a *A1Interface) Start(ctx context.Context) error {
	authMiddleware, err := NewAuthMiddleware(&a.config.Auth)
	if err != nil {
		return err
	}
	handler := NewA1Handler(a, authMiddleware)
	handler.RegisterRoutes(a.router)

	go func() {
		if err := a.server.ListenAndServe(); err != http.ErrServerClosed {
			a.logger.WithError(err).Error("A1 interface server failed")
		}
	}()
	return nil
}

// Stop stops the A1 interface.
func (a *A1Interface) Stop(ctx context.Context) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := a.server.Shutdown(ctx); err != nil {
		a.logger.WithError(err).Error("A1 interface server shutdown failed")
	}
}

// A1Handler handles HTTP requests for the A1 interface.
type A1Handler struct {
	a1Interface    *A1Interface
	authMiddleware *AuthMiddleware
}

// NewA1Handler creates a new A1 handler.
func NewA1Handler(a1Interface *A1Interface, authMiddleware *AuthMiddleware) *A1Handler {
	return &A1Handler{
		a1Interface:    a1Interface,
		authMiddleware: authMiddleware,
	}
}

// RegisterRoutes registers the A1 routes with the given router.
func (h *A1Handler) RegisterRoutes(router *mux.Router) {
	s := router.PathPrefix("/api/v1").Subrouter()
	s.Use(h.authMiddleware.Middleware)

	s.Handle("/policy_types", Authorize(http.HandlerFunc(h.getPolicyTypes), "viewer")).Methods("GET")
	s.Handle("/policy_types/{policy_type_id}", Authorize(http.HandlerFunc(h.getPolicyType), "viewer")).Methods("GET")
	s.Handle("/policies", Authorize(http.HandlerFunc(h.getPolicies), "viewer")).Methods("GET")
	s.Handle("/policies/{policy_id}", Authorize(http.HandlerFunc(h.createPolicy), "operator")).Methods("PUT")
	s.Handle("/policies/{policy_id}", Authorize(http.HandlerFunc(h.getPolicy), "viewer")).Methods("GET")
	s.Handle("/policies/{policy_id}", Authorize(http.HandlerFunc(h.deletePolicy), "admin")).Methods("DELETE")
}

func (h *A1Handler) getPolicyTypes(w http.ResponseWriter, r *http.Request) {
	policyTypes, err := h.a1Interface.repository.ListPolicyTypes()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(policyTypes)
}

func (h *A1Handler) getPolicyType(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	policyTypeID := vars["policy_type_id"]

	policyType, err := h.a1Interface.repository.GetPolicyType(policyTypeID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode(policyType)
}

func (h *A1Handler) getPolicies(w http.ResponseWriter, r *http.Request) {
	policies, err := h.a1Interface.repository.ListPolicies()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(policies)
}

func (h *A1Handler) createPolicy(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	policyID := vars["policy_id"]

	var policy A1Policy
	if err := json.NewDecoder(r.Body).Decode(&policy); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	policy.PolicyID = policyID

	if err := h.a1Interface.repository.CreatePolicy(&policy); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

func (h *A1Handler) getPolicy(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	policyID := vars["policy_id"]

	policy, err := h.a1Interface.repository.GetPolicy(policyID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode(policy)
}

func (h *A1Handler) deletePolicy(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	policyID := vars["policy_id"]

	if err := h.a1Interface.repository.DeletePolicy(policyID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
package a1

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
)

// A1Handler handles A1 interface requests.
type A1Handler struct {
	A1Interface *A1Interface
}

// NewA1Handler creates a new A1Handler.
func NewA1Handler(a1Interface *A1Interface) *A1Handler {
	return &A1Handler{A1Interface: a1Interface}
}

// RegisterRoutes registers the A1 interface routes.
func (h *A1Handler) RegisterRoutes(router *mux.Router) {
	router.HandleFunc("/policy_types", h.createPolicyType).Methods("POST")
	router.HandleFunc("/policy_types/{id}", h.getPolicyType).Methods("GET")
	router.HandleFunc("/policy_instances", h.createPolicyInstance).Methods("POST")
	router.HandleFunc("/policy_instances/{id}", h.getPolicyInstance).Methods("GET")
	router.HandleFunc("/ml_models", h.createMLModel).Methods("POST")
	router.HandleFunc("/ml_models/{id}", h.getMLModel).Methods("GET")
	router.HandleFunc("/enrichment_info", h.createEnrichmentInfo).Methods("POST")
	router.HandleFunc("/enrichment_info/{id}", h.getEnrichmentInfo).Methods("GET")
}

func (h *A1Handler) createPolicyType(w http.ResponseWriter, r *http.Request) {
	var pt PolicyType
	if err := json.NewDecoder(r.Body).Decode(&pt); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := h.A1Interface.PolicyManager.CreatePolicyType(pt); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

func (h *A1Handler) getPolicyType(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	pt, err := h.A1Interface.PolicyManager.GetPolicyType(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	if err := json.NewEncoder(w).Encode(pt); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (h *A1Handler) createPolicyInstance(w http.ResponseWriter, r *http.Request) {
	var pi PolicyInstance
	if err := json.NewDecoder(r.Body).Decode(&pi); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := h.A1Interface.PolicyManager.CreatePolicyInstance(pi); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

func (h *A1Handler) getPolicyInstance(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	pi, err := h.A1Interface.PolicyManager.GetPolicyInstance(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	if err := json.NewEncoder(w).Encode(pi); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (h *A1Handler) createMLModel(w http.ResponseWriter, r *http.Request) {
	var m MLModel
	if err := json.NewDecoder(r.Body).Decode(&m); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := h.A1Interface.MLModelManager.CreateMLModel(m); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

func (h *A1Handler) getMLModel(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	m, err := h.A1Interface.MLModelManager.GetMLModel(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	if err := json.NewEncoder(w).Encode(m); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (h *A1Handler) createEnrichmentInfo(w http.ResponseWriter, r *http.Request) {
	var ei EnrichmentInfo
	if err := json.NewDecoder(r.Body).Decode(&ei); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := h.A1Interface.EnrichmentManager.CreateEnrichmentInfo(ei); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

func (h *A1Handler) getEnrichmentInfo(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	ei, err := h.A1Interface.EnrichmentManager.GetEnrichmentInfo(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	if err := json.NewEncoder(w).Encode(ei); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
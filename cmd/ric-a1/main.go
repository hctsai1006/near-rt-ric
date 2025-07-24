package main

import (
	"log"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/hctsai1006/near-rt-ric/internal/config"
	"github.com/hctsai1006/near-rt-ric/pkg/a1"
	"github.com/sirupsen/logrus"
)

func main() {
	logger := logrus.New()

	// Load A1 configuration
	a1Config, err := config.LoadA1Config()
	if err != nil {
		log.Fatalf("failed to load A1 config: %v", err)
	}

	repo := a1.NewMemoryRepository()
	validator := a1.NewA1PolicyValidator()

	authMiddleware, err := a1.NewAuthMiddleware(&a1Config.Auth)
	if err != nil {
		log.Fatalf("failed to create auth middleware: %v", err)
	}

	a1Interface := a1.NewA1Interface(logger, repo, validator)
	a1Handler := a1.NewA1Handler(a1Interface, authMiddleware)

	router := mux.NewRouter()
	a1Handler.RegisterRoutes(router)

	log.Println("Starting A1 interface on :8080")
	if err := http.ListenAndServe(":8080", router); err != nil {
		log.Fatal(err)
	}
}

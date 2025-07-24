package main

import (
	"log"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/hctsai1006/near-rt-ric/pkg/a1"
)

func main() {
	a1Interface := a1.NewA1Interface()
	a1Handler := a1.NewA1Handler(a1Interface)

	router := mux.NewRouter()
	a1Handler.RegisterRoutes(router)

	log.Println("Starting A1 interface on :8080")
	if err := http.ListenAndServe(":8080", router); err != nil {
		log.Fatal(err)
	}
}
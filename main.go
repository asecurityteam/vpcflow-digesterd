package main

import (
	"net/http"
	"os"

	"bitbucket.org/atlassian/vpcflow-digesterd/pkg"
	"github.com/go-chi/chi"
)

func main() {
	router := chi.NewRouter()
	service := &digesterd.Service{}
	if err := service.BindRoutes(router); err != nil {
		panic(err.Error())
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// TODO: install standard set of middleware

	server := &http.Server{
		Addr:    ":" + port,
		Handler: router,
	}

	r := &digesterd.Runtime{
		Server: server,
	}

	if err := r.Run(); err != nil {
		panic(err.Error())
	}
}

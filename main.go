package main

import (
	"net/http"
	"os"

	"github.com/asecurityteam/vpcflow-digesterd/pkg"
	"github.com/asecurityteam/vpcflow-digesterd/pkg/plugins"
	"github.com/asecurityteam/vpcflow-digesterd/pkg/types"
	"github.com/go-chi/chi"
)

func main() {
	router := chi.NewRouter()
	middleware := []func(http.Handler) http.Handler{
		plugins.DefaultLogMiddleware(),
		plugins.DefaultStatMiddleware(),
	}
	service := &digesterd.Service{Middleware: middleware}
	if err := service.BindRoutes(router); err != nil {
		panic(err.Error())
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	server := &http.Server{
		Addr:    ":" + port,
		Handler: router,
	}

	r := &digesterd.Runtime{
		Server: server,
		ExitSignals: []types.ExitSignal{
			plugins.OS,
		},
	}

	if err := r.Run(); err != nil {
		panic(err.Error())
	}
}

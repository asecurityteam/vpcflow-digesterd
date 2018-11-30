package main

import (
	"net/http"
	"os"
	"time"

	"bitbucket.org/atlassian/transport"
	"bitbucket.org/atlassian/vpcflow-digesterd/pkg"
	"bitbucket.org/atlassian/vpcflow-digesterd/pkg/plugins"
	"bitbucket.org/atlassian/vpcflow-digesterd/pkg/types"
	"github.com/go-chi/chi"
)

func main() {
	retrier := transport.NewRetrier(
		transport.NewFixedBackoffPolicy(50*time.Millisecond),
		transport.NewLimitedRetryPolicy(3),
		transport.NewStatusCodeRetryPolicy(500, 502, 503),
	)
	decorators := transport.Chain{retrier}
	router := chi.NewRouter()
	service := &digesterd.Service{Decorators: decorators}
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
		ExitSignals: types.ExitSignals{
			plugins.OS(),
		},
	}

	if err := r.Run(); err != nil {
		panic(err.Error())
	}
}

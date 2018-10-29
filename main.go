package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"bitbucket.org/atlassian/vpcflow-digesterd/pkg/handlers/v1"
	"github.com/go-chi/chi"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	digesterHandler := &v1.DigesterHandler{}
	router := chi.NewRouter()
	router.Post("/", digesterHandler.Post)

	stop := make(chan os.Signal)
	signal.Notify(stop, os.Interrupt)
	s := &http.Server{
		Addr:    ":" + port,
		Handler: router,
	}

	go func() {
		if err := s.ListenAndServe(); err != nil {
			log.Fatal(err.Error())
		}
	}()

	<-stop

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = s.Shutdown(ctx)
}

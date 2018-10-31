package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"time"

	"bitbucket.org/atlassian/vpcflow-digesterd/pkg/handlers/v1"
	"bitbucket.org/atlassian/vpcflow-digesterd/pkg/stream"
	"github.com/go-chi/chi"
)

func mustEnv(key string) string {
	val := os.Getenv(key)
	if val == "" {
		panic(fmt.Sprintf("%s is required", key))
	}
	return val
}

func main() {
	port := mustEnv("PORT")
	streamApplianceEndpoint := mustEnv("STREAM_APPLIANCE_ENDPOINT")
	streamApplianceTopic := mustEnv("STREAM_APPLIANCE_TOPIC")
	streamApplianceURL, err := url.Parse(streamApplianceEndpoint)
	if err != nil {
		panic(err.Error())
	}

	digestQueuer := &stream.DigestQueuer{
		Client:   &http.Client{},
		Endpoint: streamApplianceURL,
		Topic:    streamApplianceTopic,
	}
	digesterHandler := &v1.DigesterHandler{
		Queuer: digestQueuer,
	}
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

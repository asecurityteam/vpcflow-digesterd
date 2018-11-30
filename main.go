package main

import (
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"time"

	"bitbucket.org/atlassian/logevent"
	hlog "bitbucket.org/atlassian/logevent/http"
	"bitbucket.org/atlassian/vpcflow-digesterd/pkg"
	"bitbucket.org/atlassian/vpcflow-digesterd/pkg/plugins"
	"bitbucket.org/atlassian/vpcflow-digesterd/pkg/types"
	"github.com/go-chi/chi"
	"github.com/rs/xstats"
	"github.com/rs/xstats/dogstatsd"
)

func main() {
	router := chi.NewRouter()
	service := &digesterd.Service{ErrorCallback: plugins.LogCallback}
	if err := service.BindRoutes(router); err != nil {
		panic(err.Error())
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	logger := logevent.New(logevent.Config{})

	var statsdWriter io.Writer
	var errWriter error
	statsdWriter, errWriter = net.Dial("udp", "127.0.0.1:8126")
	if errWriter != nil {
		logger.Error(errWriter.Error())
		logger.Error("stats disabled")
		statsdWriter = ioutil.Discard
	}
	stats := xstats.New(dogstatsd.New(statsdWriter, 10*time.Second))

	router.Use(
		hlog.NewMiddleware(logger),
		xstats.NewHandler(stats, nil),
	)

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

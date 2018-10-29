package main

import (
	"bitbucket.org/atlassian/gosrv"
	"bitbucket.org/atlassian/httplog"
	"bitbucket.org/atlassian/logevent"
	"bitbucket.org/atlassian/vpcflow-digesterd/pkg/handlers/v1"
)

// Service is our RouteBinder implementation required by gosrv
type Service struct {
	DigesterHandler *v1.DigesterHandler
}

// BindRoutes binds our HTTP handlers to the gosrv router. It is called when calling gosrv.NewServer()
func (s Service) BindRoutes(router gosrv.Router) error {
	router.Post("/", s.DigesterHandler.Post)
	return nil
}

func main() {
	digesterHandler := &v1.DigesterHandler{
		LogProvider:      logevent.FromContext,
		LogEventProvider: httplog.NewEvent,
	}
	service := Service{digesterHandler}
	config := gosrv.Config{}
	server, err := gosrv.NewServer(&config, service)
	if err != nil {
		panic(err.Error())
	}

	_ = server.ListenAndServe()
}

package app

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/guionardo/gs-rproxy/internal/docker"
	"github.com/guionardo/gs-rproxy/internal/logger"
	"github.com/guionardo/gs-rproxy/internal/telemetry"
	"go.uber.org/zap"
)

func (s *Server) setupAPI(r *mux.Router) {
	api := r.PathPrefix("/api").Host(s.config.HostName).Subrouter()
	api.Path("/containers").HandlerFunc(s.apiContainersHandler)
	api.Path("/events").HandlerFunc(s.apiEventsHandler)
	api.Path("/version").HandlerFunc(s.apiVersionHandler)
}

func (s *Server) apiVersionHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusTeapot)
	version := fmt.Sprintf(`{"version":"%s"}`, Version)
	w.Write([]byte(version))
}

func (s *Server) apiContainersHandler(w http.ResponseWriter, r *http.Request) {
	conteiners := make([]*docker.ContainerInfo, 0, 20)
	for cnt := range s.containerGetter.GetAllContainers() {
		conteiners = append(conteiners, cnt)
	}
	content, err := json.Marshal(conteiners)
	if err != nil {
		writeError(w, err)
	}
	w.WriteHeader(http.StatusOK)
	w.Write(content)
}

func (s *Server) apiEventsHandler(w http.ResponseWriter, r *http.Request) {
	// Set CORS headers to allow all origins. You may want to restrict this to specific origins in a production environment.
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Expose-Headers", "Content-Type")

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	changeChan := make(chan []*docker.ContainerInfo)
	listenerId := r.RemoteAddr
	logger.Log.Info("connected on event handler", zap.String("remote_addr", listenerId))
	s.containerGetter.Subscribe(listenerId, changeChan)
	defer s.containerGetter.Unsubscribe(listenerId)
	defer logger.Log.Info("disconnect on event handler", zap.String("remote_addr", listenerId))

	telemetry.EnableFrontend(r)
	defer telemetry.DisableFrontend(r)
	// First event:
	cnts := make([]*docker.ContainerInfo, 0, 10)
	for cnt := range s.containerGetter.GetAllContainers() {
		cnts = append(cnts, cnt)
	}
	content, _ := json.Marshal(cnts)
	fmt.Fprintf(w, "data: %s\n\n", string(content))
	w.(http.Flusher).Flush()

	for {
		select {
		case change := <-changeChan:
			content, _ := json.Marshal(change)
			fmt.Fprintf(w, "data: %s\n\n", string(content))
			w.(http.Flusher).Flush()
		case <-r.Context().Done():
			return
		}
	}

}

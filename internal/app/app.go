package app

import (
	"context"
	"errors"
	"fmt"
	"iter"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/guionardo/gs-rproxy/internal/cache"
	"github.com/guionardo/gs-rproxy/internal/docker"
	http_errors "github.com/guionardo/gs-rproxy/internal/errors"
	"github.com/guionardo/gs-rproxy/internal/frontend"
	"github.com/guionardo/gs-rproxy/internal/httpdata"
	"github.com/guionardo/gs-rproxy/internal/logger"
	"github.com/guionardo/gs-rproxy/internal/telemetry"
	"github.com/guionardo/gs-rproxy/internal/utils"
	"go.uber.org/zap"
)

var (
	AppName          = "GS-RProxy"
	Version          = "0.0.1"
	AppHeaderVersion = AppName + " v" + Version
)

type (
	Server struct {
		config          Config
		containerGetter ContainerGetter
		cache           cache.Cache
	}

	ContainerGetter interface {
		GetContainerBySubdomain(subdomain string) *docker.ContainerInfo
		GetAllContainers() iter.Seq[*docker.ContainerInfo]
		Subscribe(listenerId string, eventChan chan []*docker.ContainerInfo)
		Unsubscribe(listenerId string)
	}
)

func New(config Config, containerGetter ContainerGetter) (*Server, error) {
	return &Server{
		config:          config,
		containerGetter: containerGetter,
		cache:           cache.NewMemoryCache(1024),
	}, nil
}

func (s *Server) Run(ctx context.Context) {
	r := mux.NewRouter()

	s.setupAPI(r)

	r.PathPrefix("/metrics").Host(s.config.HostName).Handler(telemetry.PromHandler())
	r.PathPrefix("/").Host(s.config.HostName).Handler(frontend.StaticHandler())
	r.PathPrefix("/").HandlerFunc(s.defaultHandler)

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", s.config.Port),
		Handler: r,
	}

	go func() {
		logger.Log.Info(AppName,
			zap.String("version", Version),
			zap.String("hostname", s.config.HostName),
			zap.Bool("dockerized", s.config.Dockerized),
			zap.Int("default_rps", s.config.DefaultRPS))
		logger.Log.Info("Starting listening", zap.String("address", server.Addr))
		if err := server.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			logger.Log.Fatal("HTTP Server error", zap.Error(err))
		}
		logger.Log.Info("Stopped serving new connections")
	}()

	<-ctx.Done()
	shutdownCtx, shutdownRelease := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownRelease()
	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Log.Fatal("HTTP shutdown error", zap.Error(err))
	}
	logger.Log.Info("Graceful shutdown complete")

}

func (s *Server) getContainer(r *http.Request) (container *docker.ContainerInfo, err error) {
	subDomain := getSubdomain(r, s.config.HostName)
	if len(subDomain) == 0 {
		return nil, http_errors.NewHttpError(fmt.Errorf("missing subdomain on request host %s", r.Host))
	}
	container = s.containerGetter.GetContainerBySubdomain(subDomain)
	if container == nil {
		return nil, http_errors.NewHttpError(fmt.Errorf("no container available for request host %s", r.Host))
	}
	if !container.Active {
		return nil, http_errors.NewHttpError(fmt.Errorf("container %s is inactive for request host %s", container.Name, r.Host))
	}
	return container, nil
}

func (s *Server) cloneRequest(r *http.Request, container *docker.ContainerInfo) *http.Request {
	var url string
	if s.config.Dockerized {
		url = fmt.Sprintf("http://%s:%d", container.Name, container.PrivatePort)
	} else {
		url = fmt.Sprintf("http://localhost:%d", container.Port)
	}
	query := ""
	if len(r.URL.RawQuery) > 0 {
		query = "?"
	}
	nr, _ := http.NewRequest(r.Method, fmt.Sprintf("%s%s%s%s", url, r.URL.Path, query, r.URL.RawQuery), r.Body)
	nr.Header = r.Header.Clone()
	for _, k := range r.Cookies() {
		nr.AddCookie(k)
	}

	return telemetry.SetContainer(nr, container.Name)
}

func (s *Server) defaultHandler(w http.ResponseWriter, r *http.Request) {

	log := logger.Log.With(
		zap.String("method", r.Method),
		zap.String("url", r.URL.String()))
	err := s.cache.WriteCachedItem(r, w)
	if err == nil {
		log.Info("Cache Hit")
		return
	}

	if traceId := utils.GetTraceId(r); len(traceId) > 0 {
		log = log.With(zap.String("trace-id", traceId))
	}
	container, err := s.getContainer(r)
	if err != nil {
		writeError(w, err)
		return
	}
	log = log.With(zap.String("container", container.Name)).With(zap.String("subdomain", container.Subdomain))

	startTime := time.Now()
	r = s.cloneRequest(r, container)
	var resp *http.Response
	if err = container.CanRequest(); err == nil {
		resp, err = http.DefaultClient.Do(r)
		// container.RequestDone()
	} else {
		err = http_errors.NewHttpError(err, http.StatusTooManyRequests)
		resp = &http.Response{
			StatusCode: http.StatusTooManyRequests,
		}
	}
	if err != nil {
		telemetry.ReqTime(r, time.Since(startTime), 0)
		telemetry.RequestError(r, err)
		writeError(w, err)
		if resp == nil {
			resp = &http.Response{
				StatusCode: 0,
			}
		}
		log.Warn("Rate limit", zap.Int("status", resp.StatusCode), zap.Error(err))

		return
	}

	respData := httpdata.ResponseFromHttp(resp)
	w.WriteHeader(int(respData.StatusCode))
	for header, values := range respData.Headers {
		for _, value := range values {
			w.Header().Add(header, value)
		}
	}
	w.Header().Add("x-proxy", AppHeaderVersion)
	w.Write(respData.Body)
	telemetry.ReqTime(r, time.Since(startTime), int(respData.StatusCode))
	fields := []zap.Field{zap.Int("status", int(respData.StatusCode)), zap.Int("content_length", int(resp.ContentLength))}
	if validUntil := s.cache.SaveCachedItem(r, respData); !validUntil.IsZero() {
		fields = append(fields, zap.Time("cached-until", validUntil))
	}

	log.Info("Req", fields...)
}

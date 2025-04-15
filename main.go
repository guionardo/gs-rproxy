package main

import (
	"context"
	"flag"
	"os"
	"os/signal"

	"github.com/guionardo/gs-rproxy/internal/app"
	"github.com/guionardo/gs-rproxy/internal/docker"
	"github.com/guionardo/gs-rproxy/internal/logger"
)

var (
	dockerized = flag.Bool("docker", false, "use when running the proxy inside a docker container")
	port       = flag.Int("port", 8080, "HTTP port")
	hostname   = flag.String("hostname", "localhost", "base host name")
	defaultRPS = flag.Int("default_rps", 100, "default Request Per Second limit")
)

func main() {
	flag.Parse()
	defer logger.Flush()
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()
	cntGetter, err := docker.NewContainersClient(ctx, *defaultRPS)
	if err != nil {
		panic(err)
	}
	app, err := app.New(app.Config{
		HostName:   *hostname,
		Port:       *port,
		Dockerized: *dockerized,
		DefaultRPS: *defaultRPS,
	}, cntGetter)
	if err != nil {
		panic(err)
	}
	app.Run(ctx)
}

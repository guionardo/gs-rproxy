package docker

import (
	"fmt"
	"hash/crc64"
	"strconv"
	"strings"

	"github.com/docker/docker/api/types/container"
	ratelimit "github.com/guionardo/gs-rproxy/internal/rate_limit"
)

type ContainerInfo struct {
	Id           string
	Name         string
	Subdomain    string
	Port         int
	PrivatePort  int
	Image        string
	Tag          string
	Active       bool
	IP           string
	Stats        *SummaryStats
	RateLimit    *ratelimit.RateLimit `json:"-"`
	RateLimitStr string               `json:"rate_limit"`
}

func (ci *ContainerInfo) String() string {
	return fmt.Sprintf("%s:%d -> %s [%v]", ci.Subdomain, ci.Port, ci.Name, ci.Active)
}

var crcTable = crc64.MakeTable(crc64.ECMA)

func (ci *ContainerInfo) Hash() uint64 {
	return crc64.Checksum(fmt.Appendf(nil, "%v", ci), crcTable)
}

func (ci *ContainerInfo) Validate() *ContainerInfo {
	if ci.Stats == nil {
		ci.Stats = &SummaryStats{}
	}
	ci.RateLimitStr = ci.RateLimit.String()
	return ci
}

func (ci *ContainerInfo) CanRequest() error {
	ci.RateLimitStr = ci.RateLimit.String()
	return ci.RateLimit.CanRequest()
}

func (ci *ContainerInfo) RequestDone() {
	ci.RateLimit.Release()
}

func fromSummary(summary container.Summary, inspect container.InspectResponse, defaultRps int) *ContainerInfo {
	var publicPort, privatePort int
	for _, p := range summary.Ports {
		publicPort = int(p.PublicPort)
		privatePort = int(p.PrivatePort)
		break
	}

	p, err := strconv.Atoi(summary.Labels["gsrp.port"])
	if err == nil && p > 0 {
		publicPort = p
	}
	p, err = strconv.Atoi(summary.Labels["gsrp.private.port"])
	if err == nil && p > 0 {
		privatePort = p
	}
	if publicPort == 0 {
		publicPort = 80
	}
	if privatePort == 0 {
		privatePort = 80
	}

	rateLimit, err := strconv.Atoi(summary.Labels["gsrp.ratelimit"])
	if err != nil {
		rateLimit = defaultRps
	}
	ip := summary.Names[0]
	for _, nw := range summary.NetworkSettings.Networks {
		ip = nw.IPAddress
	}
	cntName, _ := strings.CutPrefix(summary.Names[0], "/")
	ci := &ContainerInfo{
		Id:          summary.ID,
		Name:        cntName,
		Subdomain:   summary.Labels["gsrp.subdomain"],
		Port:        publicPort,
		PrivatePort: privatePort,
		Image:       summary.Image,
		Active:      inspect.State.Running,
		IP:          ip,
		RateLimit:   ratelimit.NewRateLimit(rateLimit),
	}

	return ci.Validate()
}

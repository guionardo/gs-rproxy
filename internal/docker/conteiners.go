package docker

import (
	"context"
	"iter"
	"sort"
	"sync"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/events"
	"github.com/docker/docker/client"
	"github.com/guionardo/gs-rproxy/internal/logger"
	"github.com/guionardo/gs-rproxy/internal/telemetry"
	"github.com/guionardo/gs-rproxy/internal/utils"
	"go.uber.org/zap"
)

type (
	ContainersClient struct {
		cli              *client.Client
		eventChan        <-chan events.Message
		eventErr         <-chan error
		containers       map[string]*ContainerInfo
		subdomains       map[string]*ContainerInfo
		lock             sync.RWMutex
		changed          bool
		eventSubscribers map[string]chan []*ContainerInfo
		defaultRps       int
	}
)

func NewContainersClient(ctx context.Context, defaultRps int) (*ContainersClient, error) {
	client, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		return nil, err
	}
	info, err := client.Info(ctx)
	if err != nil {
		return nil, err
	}
	logger.Log.Info("Docker containers client", zap.Any("containers_running", info.ContainersRunning))
	eventChan, errChan := client.Events(ctx, events.ListOptions{Since: time.Now().Format(time.RFC3339)})
	c := &ContainersClient{
		cli:              client,
		eventChan:        eventChan,
		eventErr:         errChan,
		changed:          true,
		eventSubscribers: make(map[string]chan []*ContainerInfo),
		defaultRps:       defaultRps,
	}
	if err = c.readContainers(ctx); err != nil {
		return nil, err
	}
	go c.eventListen(ctx)
	return c, nil
}

func (c *ContainersClient) GetContainers() []*ContainerInfo {
	containers := make([]*ContainerInfo, 0, len(c.containers))
	for _, cnt := range c.containers {
		containers = append(containers, cnt)
	}
	sort.Slice(containers, func(i, j int) bool {
		return containers[i].Name < containers[j].Name
	})
	return containers
}

func (c *ContainersClient) Subscribe(listenerId string, eventChan chan []*ContainerInfo) {
	c.lock.Lock()
	c.eventSubscribers[listenerId] = eventChan
	c.lock.Unlock()
}

func (c *ContainersClient) Unsubscribe(listenerId string) {
	c.lock.Lock()
	delete(c.eventSubscribers, listenerId)
	c.lock.Unlock()
}

func (c *ContainersClient) eventListen(ctx context.Context) {
	logger.Log.Info("starting events listening")
	defer logger.Log.Info("stopped events listening")
	statTicker := time.Tick(time.Second * 5)
	containers := c.GetContainers()
	var iteration int
	for {
		select {
		case err := <-c.eventErr:
			logger.Log.Error("Event", zap.Any("error", err))
			return
		case message := <-c.eventChan:
			if message.Type == events.ContainerEventType && utils.IsOneOf(message.Action, events.ActionStart, events.ActionStop, events.ActionPause, events.ActionUnPause, events.ActionDie) {
				logger.Log.Info("Event", zap.String("action", string(message.Action)))
				c.readContainers(ctx)
				containers = c.GetContainers()
				c.notifyListenersForChange()
			} else {
				logger.Log.Info("Event unprocessed", zap.String("action", string(message.Action)))
			}
		case <-statTicker:
			if len(containers) == 0 {
				continue
			}
			cnt := containers[iteration%len(containers)]
			if stats, err := GetStatSummary(c.cli, ctx, cnt.Id); err == nil {
				cnt.Stats = stats
				c.notifyListenersForChange()
			}
			iteration++
		case <-ctx.Done():
			logger.Log.Info("end listening")
			return
		}
	}
}

func (c *ContainersClient) notifyListenersForChange() {
	cnts := c.GetContainers()
	for _, ec := range c.eventSubscribers {
		ec <- cnts
	}
}

func (c *ContainersClient) readContainers(ctx context.Context) error {
	c.lock.Lock()
	defer c.lock.Unlock()

	containers, err := c.cli.ContainerList(ctx, container.ListOptions{})
	if err != nil {
		return err
	}

	c.containers = make(map[string]*ContainerInfo)
	c.subdomains = make(map[string]*ContainerInfo)
	for _, cnt := range containers {
		if inspect, err := c.cli.ContainerInspect(ctx, cnt.ID); err == nil {
			cntInfo := fromSummary(cnt, inspect, c.defaultRps)

			c.containers[cnt.ID] = cntInfo
			c.subdomains[cntInfo.Subdomain] = cntInfo
			logger.Log.Info("container",
				zap.String("info", cntInfo.String()),
				zap.Int("rate limit", cntInfo.RateLimit.RPS()))
			telemetry.SetContainerRPS(cntInfo.Name, cntInfo.RateLimit.RPS())
		}

	}
	return nil
}

func (c *ContainersClient) GetContainerBySubdomain(subdomain string) *ContainerInfo {
	return c.subdomains[subdomain]
}

func (c *ContainersClient) GetAllContainers() iter.Seq[*ContainerInfo] {
	return func(yield func(*ContainerInfo) bool) {
		for _, cnt := range c.containers {
			if !yield(cnt) {
				return
			}
		}
	}
}

package service

import (
	"sync"
	"time"

	"github.com/DataDog/datadog-agent/pkg/proto/pbgo"
	"github.com/benbjohnson/clock"
)

type client struct {
	expireAt time.Time
	pbClient *pbgo.Client
}

func (c *client) expired(clock clock.Clock) bool {
	return clock.Now().After(c.expireAt)
}

type clients struct {
	mutex sync.Mutex
	clock clock.Clock

	clientsTTL time.Duration
	clients    map[string]*client
}

func newClients(clock clock.Clock, clientsTTL time.Duration) *clients {
	return &clients{
		clock:      clock,
		clientsTTL: clientsTTL,
		clients:    make(map[string]*client),
	}
}

// seen marks the given client as active
func (c *clients) seen(pbClient *pbgo.Client) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.clients[pbClient.Id] = &client{
		expireAt: c.clock.Now().Add(c.clientsTTL),
		pbClient: pbClient,
	}
}

// activeClients returns the list of active clients
func (c *clients) activeClients() []*pbgo.Client {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	var activeClients []*pbgo.Client
	for id, client := range c.clients {
		if client.expired(c.clock) {
			delete(c.clients, id)
			continue
		}
		activeClients = append(activeClients, client.pbClient)
	}
	return activeClients
}

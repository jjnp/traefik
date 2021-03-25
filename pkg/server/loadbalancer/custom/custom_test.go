package custom

import (
	"context"
	"github.com/containous/traefik/v2/pkg/config/dynamic"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/url"
	"testing"
)

type mockedHttpHandler struct {}

func (m mockedHttpHandler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	panic("implement me")
}

func TestAddRemoveServer(t *testing.T) {
	c, err := New(&mockedHttpHandler{}, &dynamic.Custom{}, context.TODO())
	if err != nil {
		assert.Fail(t, "error creating custom instance", err)
	}
	s1, s2 := &url.URL{}, &url.URL{}
	c.UpsertServer(s1)
	c.UpsertServer(s2)
	servers := c.Servers()
	assert.Equal(t, 2, len(servers))
	for _, p := range c.metricsProviders {
		w, _ := p.GetWeights()
		assert.Equal(t, 2, len(w))
	}
	c.RemoveServer(s2)
	servers = c.Servers()
	assert.Equal(t, 1, len(servers))
	for _, p := range c.metricsProviders {
		w, _ := p.GetWeights()
		assert.Equal(t, 1, len(w))
	}
}

package docker

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewContainersClient(t *testing.T) {
	ctx := t.Context()
	cli, err := NewContainersClient(ctx, 1)
	assert.NoError(t, err)
	assert.NotNil(t, cli)
	time.Sleep(time.Second * 30)
	assert.NotNil(t, cli)
	for id, cnt := range cli.containers {
		t.Logf("%s = %s", id, cnt)
	}
}

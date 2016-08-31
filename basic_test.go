package dockere2e

import (
	// basic testing
	"flag"
	"os"
	"testing"

	"context"
	"time"

	// assertions are nice, let's do more of those
	"github.com/stretchr/testify/assert"

	// Engine API imports for talking to the docker engine
	"github.com/docker/engine-api/client"
	"github.com/docker/engine-api/types"
)

func TestMain(m *testing.M) {
	// gotta call this at the start or NONE of the flags work
	flag.Parse()

	// we need a client
	defaultHeaders := map[string]string{"User-Agent": "engine-api-cli-1.0"}
	cli, err := client.NewClient("unix:///var/run/docker.sock", "v1.22", nil, defaultHeaders)
	if err != nil {
		os.Exit(1)
	}

	exit := m.Run()
	// clean up the testing services we create before we wrap up
	CleanTestServices(context.Background(), cli)
	// and then bow out
	os.Exit(exit)
}

func TestServiceList(t *testing.T) {
	defaultHeaders := map[string]string{"User-Agent": "engine-api-cli-1.0"}
	cli, err := client.NewClient("unix:///var/run/docker.sock", "v1.22", nil, defaultHeaders)

	assert.NoError(t, err, "Client creation failed")

	// first, be sure that we have no services
	services, err := cli.ServiceList(context.Background(), types.ServiceListOptions{})
	assert.NoError(t, err, "error listing service")
	assert.Empty(t, services)
}

func TestServiceCreate(t *testing.T) {
	defaultHeaders := map[string]string{"User-Agent": "engine-api-cli-1.0"}
	cli, err := client.NewClient("unix:///var/run/docker.sock", "v1.22", nil, defaultHeaders)

	assert.NoError(t, err, "Client creation failed")
	serviceSpec := CannedServiceSpec("test", 3)

	// first, verify that the server responds as expected
	resp, err := cli.ServiceCreate(context.Background(), serviceSpec, types.ServiceCreateOptions{})
	assert.NoError(t, err, "Error creating service")
	assert.NotNil(t, resp, "Resp is nil for some reason")
	assert.NotZero(t, resp.ID, "response ID shouldn't be zero")

	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	err = waitForConverge(ctx, 2*time.Second, func() error {
		_, _, err := cli.ServiceInspectWithRaw(ctx, resp.ID)
		if err != nil {
			return err
		}
		return nil
	})

	assert.NoError(t, err)
}

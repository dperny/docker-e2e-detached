package dockere2e

import (
	// basic testing
	"testing"

	"context"
	"fmt"
	"time"

	"github.com/pkg/errors"

	// assertions are nice, let's do more of those
	"github.com/stretchr/testify/assert"

	// Engine API imports for talking to the docker engine
	"github.com/docker/engine-api/client"
	"github.com/docker/engine-api/types"
	"github.com/docker/engine-api/types/swarm"
)

// waitForConverge does test every poll
// returns nothing if test returns nothing, or test's error after context is done
//
// make sure that context is either canceled or given a timeout; if it isn't,
// test will run until half life 3 is released.
func waitForConverge(ctx context.Context, poll time.Duration, test func() error) error {
	var err error
	// create a ticker and a timer
	r := time.NewTicker(poll)
	// don't forget to close this thing
	// do we have to close this thing? idk
	defer r.Stop()

	for {
		select {
		case <-r.C:
			// do test, save the error
			fmt.Println("polling")
			err = test()
		case <-ctx.Done():
			// if the timer fires, just return whatever our last error was
			return errors.Wrap(err, "failed to converge")
		}
		// if there is no error, we're done
		if err == nil {
			return nil
		}
	}

	return err
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
	var replicas uint64 = 3
	serviceSpec := swarm.ServiceSpec{
		TaskTemplate: swarm.TaskSpec{
			ContainerSpec: swarm.ContainerSpec{
				Image: "nginx",
			},
		},
		Mode: swarm.ServiceMode{Replicated: &swarm.ReplicatedService{Replicas: &replicas}},
	}

	// first, verify that the server responds as expected
	resp, err := cli.ServiceCreate(context.Background(), serviceSpec, types.ServiceCreateOptions{})
	assert.NoError(t, err, "Error creating service")
	assert.NotNil(t, resp, "Resp is nil for some reason")
	assert.NotZero(t, resp.ID, "response ID shouldn't be zero")

	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	err = waitForConverge(ctx, 2*time.Second, func() error {
		fmt.Println("succeed")
		return nil
	})

	assert.NoError(t, err)

	ctx, _ = context.WithTimeout(context.Background(), 10*time.Second)
	err = waitForConverge(ctx, 2*time.Second, func() error {
		fmt.Println("just wait")
		return errors.New("just fail")
	})

	assert.Error(t, err)
}

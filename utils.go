package dockere2e

import (
	"context"
	"time"

	"github.com/pkg/errors"

	"github.com/docker/engine-api/client"
	"github.com/docker/engine-api/types"
	"github.com/docker/engine-api/types/filters"
	"github.com/docker/engine-api/types/swarm"
)

const E2EServiceLabel = "e2etesting"

// CleanTestServices removes all services with the E2EServiceLabel
func CleanTestServices(ctx context.Context, cli *client.Client) error {
	f := filters.NewArgs()
	f.Add("label", E2EServiceLabel)
	opts := types.ServiceListOptions{
		Filter: f,
	}
	services, err := cli.ServiceList(ctx, opts)
	if err != nil {
		return err
	}

	for _, service := range services {
		cli.ServiceRemove(ctx, service.ID)
	}

	return nil
}

// CannedServiceSpec returns a ready-to-go service spec with name and replicas
func CannedServiceSpec(name string, replicas uint64) swarm.ServiceSpec {
	return swarm.ServiceSpec{
		Annotations: swarm.Annotations{
			Name:   "name",
			Labels: map[string]string{E2EServiceLabel: "true"},
		},
		TaskTemplate: swarm.TaskSpec{
			ContainerSpec: swarm.ContainerSpec{
				Image: "nginx",
			},
		},
		Mode: swarm.ServiceMode{Replicated: &swarm.ReplicatedService{Replicas: &replicas}},
	}
}

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

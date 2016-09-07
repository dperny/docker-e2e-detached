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

// CleanTestServices removes all e2etesting services with the specified labels
func CleanTestServices(ctx context.Context, cli *client.Client, labels ...string) error {
	// create a new filter for our test label
	f := GetTestFilter(labels...)
	opts := types.ServiceListOptions{
		Filter: f,
	}
	// get the services with that label
	services, err := cli.ServiceList(ctx, opts)
	if err != nil {
		return err
	}

	// delete all of them
	for _, service := range services {
		cli.ServiceRemove(ctx, service.ID)
	}

	return nil
}

// CannedServiceSpec returns a ready-to-go service spec with name and replicas
func CannedServiceSpec(name string, replicas uint64, labels ...string) swarm.ServiceSpec {
	// first create the canned spec
	spec := swarm.ServiceSpec{
		Annotations: swarm.Annotations{
			Name:   name,
			Labels: map[string]string{E2EServiceLabel: "true"},
		},
		TaskTemplate: swarm.TaskSpec{
			ContainerSpec: swarm.ContainerSpec{
				Image:   "alpine",
				Command: []string{"ping", "localhost"},
			},
		},
		Mode: swarm.ServiceMode{Replicated: &swarm.ReplicatedService{Replicas: &replicas}},
	}

	// then, add labels
	for _, label := range labels {
		spec.Annotations.Labels[label] = ""
	}

	return spec
}

// WaitForConverge does test every poll
// returns nothing if test returns nothing, or test's error after context is done
//
// make sure that context is either canceled or given a timeout; if it isn't,
// test will run until half life 3 is released.
//
// if an irrecoverable error is noticed during the test function, calling the
// context's cancel func from inside the test can be used to abort the test
// before the timeout
func WaitForConverge(ctx context.Context, poll time.Duration, test func() error) error {
	// create a ticker and a timer
	r := time.NewTicker(poll)
	// don't forget to close this thing
	// do we have to close this thing? idk
	defer r.Stop()

	var err error
	for {
		select {
		case <-ctx.Done():
			// if the context is done, just return whatever our last saved error was
			return errors.Wrap(err, "failed to converge")
		case <-r.C:
			// do test, save the error
			err = test()
		}
		// if there is no error, we're done
		if err == nil {
			return nil
		}
	}
}

// GetServiceTasks returns all of the tasks associated with a the service
func GetServiceTasks(ctx context.Context, cli *client.Client, serviceID string) ([]swarm.Task, error) {
	// get the default filter
	filterArgs := GetTestFilter()
	// all of the tasks that we want to be running
	filterArgs.Add("desired-state", "running")
	// on the service we're requesting
	filterArgs.Add("service", serviceID)
	return cli.TaskList(ctx, types.TaskListOptions{Filter: filterArgs})
}

// GetTestFilter creates a default filter for labels
// Always adds the E2EServiceLabel, plus some user-defined labels.
// if you need more fitlers, add them to the returned value.
func GetTestFilter(labels ...string) filters.Args {
	filterArgs := filters.NewArgs()
	filterArgs.Add("label", E2EServiceLabel)
	for _, l := range labels {
		filterArgs.Add("label", l)
	}
	return filterArgs
}

// ServiceScale scales a service to the provided number
/*
func ServiceScale(ctx context.Context, cli *client.Client, serviceID string, replicas uint64) (serviceID, error) {
	service, _, err = cli.ServiceInspectWithRaw(ctx, serviceID)
	spec := service.Spec
	spec.Mode.Replicated.Replicas = &replicas
}
*/

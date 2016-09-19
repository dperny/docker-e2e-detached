package dockere2e

import (
	// basic imports
	"context"
	"flag"
	"fmt"
	"os"
	"testing"
	"time"

	// assertions are nice, let's do more of those
	"github.com/stretchr/testify/assert"
	// errors gives us Wrap which is really useful
	"github.com/pkg/errors"

	// Engine API imports for talking to the docker engine
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/swarm"
	"github.com/docker/docker/client"
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

	// clean up the testing services that might have existed before we start
	CleanTestServices(context.Background(), cli)

	// run the tests, save the exit code
	exit := m.Run()
	// and then bow out
	os.Exit(exit)
}

func TestServicesList(t *testing.T) {
	t.Parallel()
	defaultHeaders := map[string]string{"User-Agent": "engine-api-cli-1.0"}
	cli, err := client.NewClient("unix:///var/run/docker.sock", "v1.22", nil, defaultHeaders)

	assert.NoError(t, err, "Client creation failed")

	// list all services with the "TestServiceList" label
	opts := types.ServiceListOptions{Filter: GetTestFilter("TestServiceList")}
	services, err := cli.ServiceList(context.Background(), opts)
	// there shouldn't be any services with that label
	assert.NoError(t, err, "error listing service")
	assert.Empty(t, services)
}

func TestServicesCreate(t *testing.T) {
	t.Parallel()
	// label name for cleanup later
	name := "TestServicesCreate"

	defaultHeaders := map[string]string{"User-Agent": "engine-api-cli-1.0"}
	cli, err := client.NewClient("unix:///var/run/docker.sock", "v1.22", nil, defaultHeaders)

	assert.NoError(t, err, "Client creation failed")
	// create a spec for a task named TestServicesCreate labeled the same, with 3 replicas
	serviceSpec := CannedServiceSpec(name, 3, name)

	// first, verify that the server responds as expected
	resp, err := cli.ServiceCreate(context.Background(), serviceSpec, types.ServiceCreateOptions{})
	assert.NoError(t, err, "Error creating service")
	assert.NotNil(t, resp, "Resp is nil for some reason")
	assert.NotZero(t, resp.ID, "response ID shouldn't be zero")

	// now make sure we can call back and get the service
	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	err = WaitForConverge(ctx, 2*time.Second, func() error {
		_, _, err := cli.ServiceInspectWithRaw(ctx, resp.ID)
		if err != nil {
			return err
		}
		return nil
	})
	assert.NoError(t, err)

	// clean up our service
	CleanTestServices(context.Background(), cli, name)
}

func TestServicesScale(t *testing.T) {
	t.Parallel()
	name := "TestServicesScale"

	defaultHeaders := map[string]string{"User-Agent": "engine-api-cli-1.0"}
	cli, err := client.NewClient("unix:///var/run/docker.sock", "v1.22", nil, defaultHeaders)

	// create a new service
	serviceSpec := CannedServiceSpec(name, 1, name)
	service, err := cli.ServiceCreate(context.Background(), serviceSpec, types.ServiceCreateOptions{})
	assert.NoError(t, err, "error creating service")

	// little generator to create scaling tests
	// welcome to closure hell
	// TODO(dperny) abstract this into a standalone function?
	scaleCheck := func(ctx context.Context, replicas int) func() error {
		return func() error {
			// get all of the tasks for the service
			tasks, err := GetServiceTasks(ctx, cli, service.ID)
			if err != nil {
				return err
			}
			// check for correct number of tasks
			if t := len(tasks); t != replicas {
				return fmt.Errorf("wrong number of tasks, got %v expected %v", t, replicas)
			}
			// verify that all tasks are in the RUNNING state
			for _, task := range tasks {
				if task.Status.State != swarm.TaskStateRunning {
					return errors.New("a task is not yet running")
				}
			}
			// if all of the above checks out, service has converged
			return nil
		}
	}

	// check that it converges to 1 replica
	ctx, _ := context.WithTimeout(context.Background(), 30*time.Second)
	err = WaitForConverge(ctx, 2*time.Second, scaleCheck(ctx, 1))
	assert.NoError(t, err)

	// get the full spec to make changes
	full, _, err := cli.ServiceInspectWithRaw(context.Background(), service.ID)
	// more replicas
	var replicas uint64 = 3
	full.Spec.Mode.Replicated.Replicas = &replicas
	// send the update
	version := full.Meta.Version
	err = cli.ServiceUpdate(context.Background(), service.ID, version, full.Spec, types.ServiceUpdateOptions{})
	assert.NoError(t, err)

	// check that it converges to 3 replicas
	ctx, _ = context.WithTimeout(context.Background(), 30*time.Second)
	err = WaitForConverge(ctx, 2*time.Second, scaleCheck(ctx, 3))
	assert.NoError(t, err)

	// clean up after
	CleanTestServices(context.Background(), cli, name)
}

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
	// errors gives us some dead simple bonus error handling
	"github.com/pkg/errors"

	// Engine API imports for talking to the docker engine
	"github.com/docker/engine-api/client"
	"github.com/docker/engine-api/types"
	"github.com/docker/engine-api/types/swarm"
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

	// run the tests, save the exit code
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

	// now make sure we can call back and get the service
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

func TestServiceScale(t *testing.T) {
	defaultHeaders := map[string]string{"User-Agent": "engine-api-cli-1.0"}
	cli, err := client.NewClient("unix:///var/run/docker.sock", "v1.22", nil, defaultHeaders)

	// create a new service
	serviceSpec := CannedServiceSpec("scaletest", 1)
	service, err := cli.ServiceCreate(context.Background(), serviceSpec, types.ServiceCreateOptions{})
	assert.NoError(t, err, "error creating service")

	ctx, _ := context.WithTimeout(context.Background(), 2*time.Minute)
	err = waitForConverge(ctx, 2*time.Second, func() error {
		// TODO(dperny): this whole anonymous function can probably be abstracted
		// get all of the tasks for the service
		tasks, err := GetServiceTasks(ctx, cli, service.ID)
		if err != nil {
			return err
		}
		// make sure we have the correct number of tasks
		if len(tasks) != 1 {
			return errors.New("expected 1 task")
		}

		// make sure they're all running
		for _, task := range tasks {
			if task.Status.State != swarm.TaskStateRunning {
				return errors.New("a task is not yet running")
			}
		}
		return nil
	})

	assert.NoError(t, err)
}

package dockere2e

import (
	// basic imports
	"context"
	"fmt"
	"io/ioutil"
	"sync"
	"testing"
	"time"

	// testify
	"github.com/stretchr/testify/assert"

	// http is used to test network endpoints
	"net/http"

	// docker api
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/swarm"
	"github.com/docker/docker/client"
)

// tests the load balancer for services with public endpoints
func TestNetworkExternalLb(t *testing.T) {
	// TODO(dperny): there are debugging statements commented out. remove them.
	t.Parallel()
	name := "TestNetworkExternalLb"
	// create a client
	defaultHeaders := map[string]string{"User-Agent": "engine-api-cli-1.0"}
	cli, err := client.NewClient("unix:///var/run/docker.sock", "v1.22", nil, defaultHeaders)
	assert.NoError(t, err, "Client creation failed")

	replicas := 3
	spec := CannedServiceSpec(name, 3, name)
	// use nginx
	spec.TaskTemplate.ContainerSpec.Image = "zembutsu/docker-sample-nginx"
	spec.TaskTemplate.ContainerSpec.Command = nil
	// expose a port
	spec.EndpointSpec = &swarm.EndpointSpec{
		Mode: swarm.ResolutionModeVIP,
		Ports: []swarm.PortConfig{
			{
				Protocol:      swarm.PortConfigProtocolTCP,
				TargetPort:    80,
				PublishedPort: 8080,
			},
		},
	}

	// create the service
	service, err := cli.ServiceCreate(context.Background(), spec, types.ServiceCreateOptions{})
	assert.NoError(t, err, "Error creating service")
	assert.NotNil(t, service, "Resp is nil for some reason")
	assert.NotZero(t, service.ID, "serviceonse ID is zero, something is amiss")

	// now make sure the service comes up
	ctx, _ := context.WithTimeout(context.Background(), 30*time.Second)
	err = WaitForConverge(ctx, 1*time.Second, func() error {
		// TODO(dperny): we use a similar function in basic_test/TestServicesScale.
		// should we break it out into a helper function?
		tasks, err := GetServiceTasks(ctx, cli, service.ID)
		if err != nil {
			return err
		}

		// check for correct number of tasks
		if t := len(tasks); t != replicas {
			return fmt.Errorf("wrong number of tasks, got %v expected %v", t, replicas)
		}
		// verify all tasks in running state
		for _, task := range tasks {
			if task.Status.State != swarm.TaskStateRunning {
				return fmt.Errorf("a task is not yet running")
			}
		}
		return nil
	})
	assert.NoError(t, err)

	// create a context, and also grab the cancelfunc
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)

	// alright now comes the tricky part. we're gonna hit the endpoint
	// repeatedly until we get 3 different container ids, twice each.
	// if we hit twice each, we know that we've been LB'd around to each
	// instance. why twice? seems like a good number, idk. when i test LB
	// manually i just hit the endpoint a few times until i've seen each
	// container a couple of times

	// create a map to store all the containers we've seen
	containers := make(map[string]int)
	// create a mutex to synchronize access to this map
	mu := new(sync.Mutex)

	// first we need a function to poll containers, and let it run
	go func() {
		for {
			select {
			case <-ctx.Done():
				// stop polling when ctx is done
				return
			default:
				// anonymous func to leverage defers
				func() {
					// TODO(dperny) consider breaking out into separate function
					// lock the mutex to synchronize access to the map
					mu.Lock()
					defer mu.Unlock()

					// poll the endpoint
					resp, err := http.Get("http://127.0.0.1:8080")
					if err != nil {
						// TODO(dperny) properly handle error
						// fmt.Printf("error: %v", err)
						return
					}

					// body text should just be the container id
					namebytes, err := ioutil.ReadAll(resp.Body)
					// docs say we have to close the body. defer doing so
					defer resp.Body.Close()
					if err != nil {
						// TODO(dperny) properly handle error
						return
					}
					name := string(namebytes)
					// fmt.Printf("saw container: %v\n", name)

					// if the container has already been seen, increment its count
					if count, ok := containers[name]; ok {
						containers[name] = count + 1
						// if not, add it as a new record with count 1
					} else {
						containers[name] = 1
					}
				}()
				time.Sleep(5 * time.Millisecond)
			}
		}
	}()

	// function to check if we've been LB'd to all containers
	checkComplete := func() error {
		mu.Lock()
		defer mu.Unlock()
		c := len(containers)
		// fmt.Printf("saw %v containers\n", c)
		// check if we have too many containers (unlikely but possible)
		if c > replicas {
			// cancel the context, we have overshot and will never converge
			cancel()
			return fmt.Errorf("expected %v different container IDs, got %v", replicas, c)
		}
		// now check if we have too few
		if c < replicas {
			return fmt.Errorf("haven't seen enough different containers, expected %v got %v", replicas, c)
		}
		// now check that we've hit each container at least 2 times
		for name, count := range containers {
			if count < 2 {
				return fmt.Errorf("haven't seen container %v twice", name)
			}
		}
		// if everything so far passes, we're golden
		return nil
	}

	err = WaitForConverge(ctx, time.Second, checkComplete)
	// cancel the context to stop polling
	cancel()

	// fmt.Printf("saw these containers like this: %v", containers)

	assert.NoError(t, err)

	CleanTestServices(context.Background(), cli, name)
}

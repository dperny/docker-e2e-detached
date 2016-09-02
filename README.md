docker-e2e
==========
`docker-e2e` is a project to end-to-end test Docker Engine, especially 
orchestration features, in real deployment environments. Unlike integration 
tests, the goal of `docker-e2e` is to issue commands strictly through public 
apis to real Docker Engines running real containers. 

Building the Tests
------------------
The tests require go version 1.7+

Running the Tests
-----------------
Tests are built with the go test framework, and as such, running `go test` in 
the project root will run the tests.

The tests must be run on a Docker Swarm Mode manager node.

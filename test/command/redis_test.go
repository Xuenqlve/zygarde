package command

import (
	"strconv"
	"testing"
	"time"
)

func TestRedisSingleUpDoctorDown(t *testing.T) {
	tc := newLifecycleTestContext(t)

	redisPort := freePort(t)
	redisImage := localImageOrSkip(
		t,
		"redis:6.2",
		"localhost/redis:6.2",
		"redis:6.2",
		"docker.io/library/redis:6.2",
	)

	t.Logf("redis port: %d", redisPort)
	t.Logf("redis image: %s", redisImage)
	blueprintPath := tc.writeBlueprint(redisBlueprint(redisPort, redisImage))

	upResult := tc.up(blueprintPath)
	tc.verifyCurrentEnvironment(upResult)
	_, artifact := tc.verifyRunningEnvironment(upResult)
	tc.verifyRuntimeFiles(artifact)
	tc.verifyStatusRunning()
	tc.waitForDoctorPass(2 * time.Minute)
	tc.downAndVerify(upResult)
}

func TestRedisMasterSlaveUpDoctorDown(t *testing.T) {
	tc := newLifecycleTestContext(t)

	masterPort := freePort(t)
	slavePort := freePort(t)
	for slavePort == masterPort {
		slavePort = freePort(t)
	}
	redisImage := localImageOrSkip(
		t,
		"redis:6.2",
		"localhost/redis:6.2",
		"redis:6.2",
		"docker.io/library/redis:6.2",
	)

	t.Logf("redis master port: %d", masterPort)
	t.Logf("redis slave port: %d", slavePort)
	t.Logf("redis image: %s", redisImage)
	blueprintPath := tc.writeBlueprint(redisMasterSlaveBlueprint(masterPort, slavePort, redisImage))

	upResult := tc.up(blueprintPath)
	tc.verifyCurrentEnvironment(upResult)
	_, artifact := tc.verifyRunningEnvironment(upResult)
	tc.verifyRuntimeFiles(artifact)
	tc.verifyStatusRunning()
	tc.waitForDoctorPass(2 * time.Minute)
	tc.downAndVerify(upResult)
}

func TestRedisClusterUpDoctorDown(t *testing.T) {
	tc := newLifecycleTestContext(t)

	node1Port := freePort(t)
	node2Port := freePort(t)
	node3Port := freePort(t)
	node1BusPort := freePort(t)
	node2BusPort := freePort(t)
	node3BusPort := freePort(t)
	redisImage := localImageOrSkip(
		t,
		"redis:6.2",
		"localhost/redis:6.2",
		"redis:6.2",
		"docker.io/library/redis:6.2",
	)

	t.Logf("redis cluster ports: %d, %d, %d", node1Port, node2Port, node3Port)
	t.Logf("redis cluster bus ports: %d, %d, %d", node1BusPort, node2BusPort, node3BusPort)
	t.Logf("redis image: %s", redisImage)
	blueprintPath := tc.writeBlueprint(redisClusterBlueprint(node1Port, node2Port, node3Port, node1BusPort, node2BusPort, node3BusPort, redisImage))

	upResult := tc.up(blueprintPath)
	tc.verifyCurrentEnvironment(upResult)
	_, artifact := tc.verifyRunningEnvironment(upResult)
	tc.verifyRuntimeFiles(artifact)
	tc.verifyStatusRunning()
	tc.waitForDoctorPass(2 * time.Minute)
	tc.downAndVerify(upResult)
}

func redisBlueprint(port int, image string) string {
	blueprint := "name: redis-test\n" +
		"version: \"v1\"\n" +
		"description: redis single lifecycle integration test\n\n" +
		"runtime:\n" +
		"  project-name: redis-test\n\n" +
		"services:\n" +
		"  - name: redis-1\n" +
		"    middleware: redis\n" +
		"    template: single\n" +
		"    values:\n" +
		"      version: v6.2\n" +
		"      port: " + strconv.Itoa(port) + "\n" +
		"      data_dir: ./data/redis-1\n"
	if image != "" {
		blueprint += "      image: " + image + "\n"
	}
	return blueprint
}

func redisMasterSlaveBlueprint(masterPort, slavePort int, image string) string {
	blueprint := "name: redis-master-slave-test\n" +
		"version: \"v1\"\n" +
		"description: redis master-slave lifecycle integration test\n\n" +
		"runtime:\n" +
		"  project-name: redis-master-slave-test\n\n" +
		"services:\n" +
		"  - name: redis-ms-1\n" +
		"    middleware: redis\n" +
		"    template: master-slave\n" +
		"    values:\n" +
		"      version: v6.2\n" +
		"      master_port: " + strconv.Itoa(masterPort) + "\n" +
		"      slave_port: " + strconv.Itoa(slavePort) + "\n" +
		"      master_data_dir: ./data/redis-master\n" +
		"      slave_data_dir: ./data/redis-slave\n"
	if image != "" {
		blueprint += "      master_image: " + image + "\n"
		blueprint += "      slave_image: " + image + "\n"
	}
	return blueprint
}

func redisClusterBlueprint(node1Port, node2Port, node3Port, node1BusPort, node2BusPort, node3BusPort int, image string) string {
	blueprint := "name: redis-cluster-test\n" +
		"version: \"v1\"\n" +
		"description: redis cluster lifecycle integration test\n\n" +
		"runtime:\n" +
		"  project-name: redis-cluster-test\n\n" +
		"services:\n" +
		"  - name: redis-cluster-1\n" +
		"    middleware: redis\n" +
		"    template: cluster\n" +
		"    values:\n" +
		"      version: v6.2\n" +
		"      node_1_port: " + strconv.Itoa(node1Port) + "\n" +
		"      node_1_bus_port: " + strconv.Itoa(node1BusPort) + "\n" +
		"      node_2_port: " + strconv.Itoa(node2Port) + "\n" +
		"      node_2_bus_port: " + strconv.Itoa(node2BusPort) + "\n" +
		"      node_3_port: " + strconv.Itoa(node3Port) + "\n" +
		"      node_3_bus_port: " + strconv.Itoa(node3BusPort) + "\n" +
		"      node_1_data_dir: ./data/redis-node-1\n" +
		"      node_2_data_dir: ./data/redis-node-2\n" +
		"      node_3_data_dir: ./data/redis-node-3\n"
	if image != "" {
		blueprint += "      image: " + image + "\n"
	}
	return blueprint
}

package command

import (
	"strconv"
	"testing"
	"time"
)

func TestZooKeeperSingleUpDoctorDown(t *testing.T) {
	tc := newLifecycleTestContext(t)

	clientPort := freePort(t)
	followerPort := freePort(t)
	electionPort := freePort(t)
	image := localImageOrSkip(
		t,
		"zookeeper:3.9",
		"localhost/zookeeper:3.9",
		"zookeeper:3.9",
		"docker.io/library/zookeeper:3.9",
	)

	t.Logf("zookeeper ports: client=%d follower=%d election=%d", clientPort, followerPort, electionPort)
	t.Logf("zookeeper image: %s", image)
	blueprintPath := tc.writeBlueprint(zookeeperBlueprint(clientPort, followerPort, electionPort, image))

	upResult := tc.up(blueprintPath)
	tc.verifyCurrentEnvironment(upResult)
	_, artifact := tc.verifyRunningEnvironment(upResult)
	tc.verifyRuntimeFiles(artifact)
	tc.verifyStatusRunning()
	tc.waitForDoctorPass(4 * time.Minute)
	tc.downAndVerify(upResult)
}

func TestZooKeeperClusterUpDoctorDown(t *testing.T) {
	tc := newLifecycleTestContext(t)

	zk1ClientPort := freePort(t)
	zk2ClientPort := freePort(t)
	zk3ClientPort := freePort(t)
	image := localImageOrSkip(
		t,
		"zookeeper:3.9",
		"localhost/zookeeper:3.9",
		"zookeeper:3.9",
		"docker.io/library/zookeeper:3.9",
	)

	t.Logf("zookeeper cluster ports: zk1=%d zk2=%d zk3=%d", zk1ClientPort, zk2ClientPort, zk3ClientPort)
	t.Logf("zookeeper image: %s", image)
	blueprintPath := tc.writeBlueprint(zookeeperClusterBlueprint(zk1ClientPort, zk2ClientPort, zk3ClientPort, image))

	upResult := tc.up(blueprintPath)
	tc.verifyCurrentEnvironment(upResult)
	_, artifact := tc.verifyRunningEnvironment(upResult)
	tc.verifyRuntimeFiles(artifact)
	tc.verifyStatusRunning()
	tc.waitForDoctorPass(4 * time.Minute)
	tc.downAndVerify(upResult)
}

func zookeeperBlueprint(clientPort, followerPort, electionPort int, image string) string {
	blueprint := "name: zookeeper-test\n" +
		"version: \"v1\"\n" +
		"description: zookeeper single lifecycle integration test\n\n" +
		"runtime:\n" +
		"  project-name: zookeeper-test\n\n" +
		"services:\n" +
		"  - name: zk-1\n" +
		"    middleware: zookeeper\n" +
		"    template: single\n" +
		"    values:\n" +
		"      version: v3.9\n" +
		"      client_port: " + strconv.Itoa(clientPort) + "\n" +
		"      follower_port: " + strconv.Itoa(followerPort) + "\n" +
		"      election_port: " + strconv.Itoa(electionPort) + "\n" +
		"      data_dir: ./data/zk\n" +
		"      datalog_dir: ./datalog/zk\n"
	if image != "" {
		blueprint += "      image: " + image + "\n"
	}
	return blueprint
}

func zookeeperClusterBlueprint(zk1ClientPort, zk2ClientPort, zk3ClientPort int, image string) string {
	blueprint := "name: zookeeper-cluster-test\n" +
		"version: \"v1\"\n" +
		"description: zookeeper cluster lifecycle integration test\n\n" +
		"runtime:\n" +
		"  project-name: zookeeper-cluster-test\n\n" +
		"services:\n" +
		"  - name: zk-cluster-1\n" +
		"    middleware: zookeeper\n" +
		"    template: cluster\n" +
		"    values:\n" +
		"      version: v3.9\n" +
		"      zk1_client_port: " + strconv.Itoa(zk1ClientPort) + "\n" +
		"      zk2_client_port: " + strconv.Itoa(zk2ClientPort) + "\n" +
		"      zk3_client_port: " + strconv.Itoa(zk3ClientPort) + "\n" +
		"      zk1_data_dir: ./data/zk1\n" +
		"      zk2_data_dir: ./data/zk2\n" +
		"      zk3_data_dir: ./data/zk3\n" +
		"      zk1_datalog_dir: ./datalog/zk1\n" +
		"      zk2_datalog_dir: ./datalog/zk2\n" +
		"      zk3_datalog_dir: ./datalog/zk3\n"
	if image != "" {
		blueprint += "      image: " + image + "\n"
	}
	return blueprint
}

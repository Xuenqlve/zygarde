package create

import (
	"strconv"
	"testing"
	"time"
)

func TestMongoDBSingleUpDoctorDown(t *testing.T) {
	tc := newLifecycleTestContext(t)

	mongoPort := freePort(t)
	mongoImage := localImageOrSkip(
		t,
		"mongo:6.0",
		"localhost/mongo:6.0",
		"mongo:6.0",
		"docker.io/library/mongo:6.0",
	)

	t.Logf("mongodb port: %d", mongoPort)
	t.Logf("mongodb image: %s", mongoImage)
	blueprintPath := tc.writeBlueprint(mongodbBlueprint(mongoPort, mongoImage))

	upResult := tc.up(blueprintPath)
	tc.verifyCurrentEnvironment(upResult)
	_, artifact := tc.verifyRunningEnvironment(upResult)
	tc.verifyRuntimeFiles(artifact)
	tc.verifyStatusRunning()
	tc.waitForDoctorPass(2 * time.Minute)
	tc.downAndVerify(upResult)
}

func TestMongoDBReplicaSetUpDoctorDown(t *testing.T) {
	tc := newLifecycleTestContext(t)

	rs1Port := freePort(t)
	rs2Port := freePort(t)
	rs3Port := freePort(t)
	mongoImage := localImageOrSkip(
		t,
		"mongo:6.0",
		"localhost/mongo:6.0",
		"mongo:6.0",
		"docker.io/library/mongo:6.0",
	)

	t.Logf("mongodb replica-set ports: %d, %d, %d", rs1Port, rs2Port, rs3Port)
	t.Logf("mongodb image: %s", mongoImage)
	blueprintPath := tc.writeBlueprint(mongodbReplicaSetBlueprint(rs1Port, rs2Port, rs3Port, mongoImage))

	upResult := tc.up(blueprintPath)
	tc.verifyCurrentEnvironment(upResult)
	_, artifact := tc.verifyRunningEnvironment(upResult)
	tc.verifyRuntimeFiles(artifact)
	tc.verifyStatusRunning()
	tc.waitForDoctorPass(3 * time.Minute)
	tc.downAndVerify(upResult)
}

func TestMongoDBShardedUpDoctorDown(t *testing.T) {
	tc := newLifecycleTestContext(t)

	mongosPort := freePort(t)
	mongoImage := localImageOrSkip(
		t,
		"mongo:6.0",
		"localhost/mongo:6.0",
		"mongo:6.0",
		"docker.io/library/mongo:6.0",
	)

	t.Logf("mongodb mongos port: %d", mongosPort)
	t.Logf("mongodb image: %s", mongoImage)
	blueprintPath := tc.writeBlueprint(mongodbShardedBlueprint(mongosPort, mongoImage))

	upResult := tc.up(blueprintPath)
	tc.verifyCurrentEnvironment(upResult)
	_, artifact := tc.verifyRunningEnvironment(upResult)
	tc.verifyRuntimeFiles(artifact)
	tc.verifyStatusRunning()
	tc.waitForDoctorPass(4 * time.Minute)
	tc.downAndVerify(upResult)
}

func mongodbBlueprint(port int, image string) string {
	blueprint := "name: mongodb-test\n" +
		"version: \"v1\"\n" +
		"description: mongodb single lifecycle integration test\n\n" +
		"runtime:\n" +
		"  project-name: mongodb-test\n\n" +
		"services:\n" +
		"  - name: mongodb-1\n" +
		"    middleware: mongodb\n" +
		"    template: single\n" +
		"    values:\n" +
		"      version: v6.0\n" +
		"      port: " + strconv.Itoa(port) + "\n" +
		"      data_dir: ./data/mongodb-1\n"
	if image != "" {
		blueprint += "      image: " + image + "\n"
	}
	return blueprint
}

func mongodbReplicaSetBlueprint(rs1Port, rs2Port, rs3Port int, image string) string {
	blueprint := "name: mongodb-rs-test\n" +
		"version: \"v1\"\n" +
		"description: mongodb replica-set lifecycle integration test\n\n" +
		"runtime:\n" +
		"  project-name: mongodb-rs-test\n\n" +
		"services:\n" +
		"  - name: mongodb-rs-1\n" +
		"    middleware: mongodb\n" +
		"    template: replica-set\n" +
		"    values:\n" +
		"      version: v6.0\n" +
		"      rs1_port: " + strconv.Itoa(rs1Port) + "\n" +
		"      rs2_port: " + strconv.Itoa(rs2Port) + "\n" +
		"      rs3_port: " + strconv.Itoa(rs3Port) + "\n" +
		"      rs1_data_dir: ./data/mongo-rs1\n" +
		"      rs2_data_dir: ./data/mongo-rs2\n" +
		"      rs3_data_dir: ./data/mongo-rs3\n"
	if image != "" {
		blueprint += "      image: " + image + "\n"
	}
	return blueprint
}

func mongodbShardedBlueprint(mongosPort int, image string) string {
	blueprint := "name: mongodb-sharded-test\n" +
		"version: \"v1\"\n" +
		"description: mongodb sharded lifecycle integration test\n\n" +
		"runtime:\n" +
		"  project-name: mongodb-sharded-test\n\n" +
		"services:\n" +
		"  - name: mongodb-sharded-1\n" +
		"    middleware: mongodb\n" +
		"    template: sharded\n" +
		"    values:\n" +
		"      version: v6.0\n" +
		"      mongos_port: " + strconv.Itoa(mongosPort) + "\n" +
		"      cfg1_data_dir: ./data/cfg1\n" +
		"      cfg2_data_dir: ./data/cfg2\n" +
		"      cfg3_data_dir: ./data/cfg3\n" +
		"      shard1_data_dir: ./data/shard1\n" +
		"      shard2_data_dir: ./data/shard2\n"
	if image != "" {
		blueprint += "      image: " + image + "\n"
	}
	return blueprint
}

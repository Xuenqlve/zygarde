package command

import (
	"strconv"
	"testing"
	"time"
)

func TestPostgreSQLSingleUpDoctorDown(t *testing.T) {
	tc := newLifecycleTestContext(t)

	postgresPort := freePort(t)
	postgresImage := localImageOrSkip(
		t,
		"postgres:16",
		"localhost/postgres:16",
		"postgres:16",
		"docker.io/library/postgres:16",
	)

	t.Logf("postgresql port: %d", postgresPort)
	t.Logf("postgresql image: %s", postgresImage)
	blueprintPath := tc.writeBlueprint(postgresqlBlueprint(postgresPort, postgresImage))

	upResult := tc.up(blueprintPath)
	tc.verifyCurrentEnvironment(upResult)
	_, artifact := tc.verifyRunningEnvironment(upResult)
	tc.verifyRuntimeFiles(artifact)
	tc.verifyStatusRunning()
	tc.waitForDoctorPass(2 * time.Minute)
	tc.downAndVerify(upResult)
}

func TestPostgreSQLMasterSlaveUpDoctorDown(t *testing.T) {
	tc := newLifecycleTestContext(t)

	masterPort := freePort(t)
	slavePort := freePort(t)
	for slavePort == masterPort {
		slavePort = freePort(t)
	}
	postgresImage := localImageOrSkip(
		t,
		"postgres:16",
		"localhost/postgres:16",
		"postgres:16",
		"docker.io/library/postgres:16",
	)

	t.Logf("postgresql master port: %d", masterPort)
	t.Logf("postgresql slave port: %d", slavePort)
	t.Logf("postgresql image: %s", postgresImage)
	blueprintPath := tc.writeBlueprint(postgresqlMasterSlaveBlueprint(masterPort, slavePort, postgresImage))

	upResult := tc.up(blueprintPath)
	tc.verifyCurrentEnvironment(upResult)
	_, artifact := tc.verifyRunningEnvironment(upResult)
	tc.verifyRuntimeFiles(artifact)
	tc.verifyStatusRunning()
	tc.waitForDoctorPass(3 * time.Minute)
	tc.downAndVerify(upResult)
}

func postgresqlBlueprint(port int, image string) string {
	blueprint := "name: postgresql-test\n" +
		"version: \"v1\"\n" +
		"description: postgresql single lifecycle integration test\n\n" +
		"runtime:\n" +
		"  project-name: postgresql-test\n\n" +
		"services:\n" +
		"  - name: postgres-1\n" +
		"    middleware: postgresql\n" +
		"    template: single\n" +
		"    values:\n" +
		"      version: v16\n" +
		"      port: " + strconv.Itoa(port) + "\n" +
		"      user: postgres\n" +
		"      password: postgres123\n" +
		"      database: app\n" +
		"      data_dir: ./data/postgres-1\n"
	if image != "" {
		blueprint += "      image: " + image + "\n"
	}
	return blueprint
}

func postgresqlMasterSlaveBlueprint(masterPort, slavePort int, image string) string {
	blueprint := "name: postgresql-master-slave-test\n" +
		"version: \"v1\"\n" +
		"description: postgresql master-slave lifecycle integration test\n\n" +
		"runtime:\n" +
		"  project-name: postgresql-master-slave-test\n\n" +
		"services:\n" +
		"  - name: postgres-ms-1\n" +
		"    middleware: postgresql\n" +
		"    template: master-slave\n" +
		"    values:\n" +
		"      version: v16\n" +
		"      master_port: " + strconv.Itoa(masterPort) + "\n" +
		"      slave_port: " + strconv.Itoa(slavePort) + "\n" +
		"      user: postgres\n" +
		"      password: postgres123\n" +
		"      database: app\n" +
		"      replication_user: repl_user\n" +
		"      replication_password: repl_pass\n" +
		"      master_data_dir: ./data/postgres-master\n" +
		"      slave_data_dir: ./data/postgres-slave\n"
	if image != "" {
		blueprint += "      master_image: " + image + "\n"
		blueprint += "      slave_image: " + image + "\n"
	}
	return blueprint
}

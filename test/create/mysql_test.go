package create

import (
	"strconv"
	"testing"
	"time"
)

func TestMySQLSingleUpDoctorDown(t *testing.T) {
	tc := newLifecycleTestContext(t)

	mysqlPort := freePort(t)
	t.Logf("mysql port: %d", mysqlPort)
	blueprintPath := tc.writeBlueprint(mysqlBlueprint(mysqlPort))

	upResult := tc.up(blueprintPath)
	tc.verifyCurrentEnvironment(upResult)
	_, artifact := tc.verifyRunningEnvironment(upResult)
	tc.verifyRuntimeFiles(artifact)
	tc.verifyStatusRunning()
	tc.waitForDoctorPass(2 * time.Minute)
	tc.downAndVerify(upResult)
}

func TestMySQLMasterSlaveUpDoctorDown(t *testing.T) {
	tc := newLifecycleTestContext(t)

	masterPort := freePort(t)
	slavePort := freePort(t)
	for slavePort == masterPort {
		slavePort = freePort(t)
	}

	t.Logf("mysql master port: %d", masterPort)
	t.Logf("mysql slave port: %d", slavePort)
	blueprintPath := tc.writeBlueprint(mysqlMasterSlaveBlueprint(masterPort, slavePort))

	upResult := tc.up(blueprintPath)
	tc.verifyCurrentEnvironment(upResult)
	_, artifact := tc.verifyRunningEnvironment(upResult)
	tc.verifyRuntimeFiles(artifact)
	tc.verifyStatusRunning()
	tc.waitForDoctorPass(3 * time.Minute)
	tc.downAndVerify(upResult)
}

func mysqlBlueprint(port int) string {
	return "name: mysql-test\n" +
		"version: \"v1\"\n" +
		"description: mysql single lifecycle integration test\n\n" +
		"runtime:\n" +
		"  project-name: mysql-test\n\n" +
		"services:\n" +
		"  - name: mysql-1\n" +
		"    middleware: mysql\n" +
		"    template: single\n" +
		"    values:\n" +
		"      version: v5.7\n" +
		"      port: " + intString(port) + "\n" +
		"      root_password: root123\n" +
		"      data_dir: ./data/mysql-1\n"
}

func mysqlMasterSlaveBlueprint(masterPort, slavePort int) string {
	return "name: mysql-master-slave-test\n" +
		"version: \"v1\"\n" +
		"description: mysql master-slave lifecycle integration test\n\n" +
		"runtime:\n" +
		"  project-name: mysql-master-slave-test\n\n" +
		"services:\n" +
		"  - name: mysql-ms-1\n" +
		"    middleware: mysql\n" +
		"    template: master-slave\n" +
		"    values:\n" +
		"      version: v5.7\n" +
		"      master_port: " + intString(masterPort) + "\n" +
		"      slave_port: " + intString(slavePort) + "\n" +
		"      root_password: root123\n" +
		"      replication_user: repl\n" +
		"      replication_password: repl123\n" +
		"      master_data_dir: ./data/mysql-master\n" +
		"      slave_data_dir: ./data/mysql-slave\n"
}

func intString(value int) string {
	return strconv.Itoa(value)
}

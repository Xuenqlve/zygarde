package create

import (
	"strconv"
	"testing"
	"time"
)

func TestRabbitMQSingleUpDoctorDown(t *testing.T) {
	tc := newLifecycleTestContext(t)

	amqpPort := freePort(t)
	managementPort := freePort(t)
	for managementPort == amqpPort {
		managementPort = freePort(t)
	}
	rabbitmqImage := localImageOrSkip(
		t,
		"rabbitmq:4.2-management",
		"localhost/rabbitmq:4.2-management",
		"rabbitmq:4.2-management",
		"docker.io/library/rabbitmq:4.2-management",
	)

	t.Logf("rabbitmq amqp port: %d", amqpPort)
	t.Logf("rabbitmq management port: %d", managementPort)
	t.Logf("rabbitmq image: %s", rabbitmqImage)
	blueprintPath := tc.writeBlueprint(rabbitmqBlueprint(amqpPort, managementPort, rabbitmqImage))

	upResult := tc.up(blueprintPath)
	tc.verifyCurrentEnvironment(upResult)
	_, artifact := tc.verifyRunningEnvironment(upResult)
	tc.verifyRuntimeFiles(artifact)
	tc.verifyStatusRunning()
	tc.waitForDoctorPass(2 * time.Minute)
	tc.downAndVerify(upResult)
}

func TestRabbitMQClusterUpDoctorDown(t *testing.T) {
	tc := newLifecycleTestContext(t)

	rabbit1AMQPPort := freePort(t)
	rabbit2AMQPPort := freePort(t)
	rabbit3AMQPPort := freePort(t)
	rabbit1ManagementPort := freePort(t)
	rabbit2ManagementPort := freePort(t)
	rabbit3ManagementPort := freePort(t)
	rabbitmqImage := localImageOrSkip(
		t,
		"rabbitmq:4.2-management",
		"localhost/rabbitmq:4.2-management",
		"rabbitmq:4.2-management",
		"docker.io/library/rabbitmq:4.2-management",
	)

	t.Logf("rabbitmq cluster amqp ports: %d, %d, %d", rabbit1AMQPPort, rabbit2AMQPPort, rabbit3AMQPPort)
	t.Logf("rabbitmq cluster management ports: %d, %d, %d", rabbit1ManagementPort, rabbit2ManagementPort, rabbit3ManagementPort)
	t.Logf("rabbitmq image: %s", rabbitmqImage)
	blueprintPath := tc.writeBlueprint(rabbitmqClusterBlueprint(
		rabbit1AMQPPort,
		rabbit2AMQPPort,
		rabbit3AMQPPort,
		rabbit1ManagementPort,
		rabbit2ManagementPort,
		rabbit3ManagementPort,
		rabbitmqImage,
	))

	upResult := tc.up(blueprintPath)
	tc.verifyCurrentEnvironment(upResult)
	_, artifact := tc.verifyRunningEnvironment(upResult)
	tc.verifyRuntimeFiles(artifact)
	tc.verifyStatusRunning()
	tc.waitForDoctorPass(3 * time.Minute)
	tc.downAndVerify(upResult)
}

func rabbitmqBlueprint(amqpPort, managementPort int, image string) string {
	blueprint := "name: rabbitmq-test\n" +
		"version: \"v1\"\n" +
		"description: rabbitmq single lifecycle integration test\n\n" +
		"runtime:\n" +
		"  project-name: rabbitmq-test\n\n" +
		"services:\n" +
		"  - name: rabbitmq-1\n" +
		"    middleware: rabbitmq\n" +
		"    template: single\n" +
		"    values:\n" +
		"      version: v4.2\n" +
		"      amqp_port: " + strconv.Itoa(amqpPort) + "\n" +
		"      management_port: " + strconv.Itoa(managementPort) + "\n" +
		"      default_user: admin\n" +
		"      default_pass: admin123\n" +
		"      erlang_cookie: rabbitmq-cookie\n" +
		"      data_dir: ./data/rabbitmq-1\n"
	if image != "" {
		blueprint += "      image: " + image + "\n"
	}
	return blueprint
}

func rabbitmqClusterBlueprint(rabbit1AMQPPort, rabbit2AMQPPort, rabbit3AMQPPort, rabbit1ManagementPort, rabbit2ManagementPort, rabbit3ManagementPort int, image string) string {
	blueprint := "name: rabbitmq-cluster-test\n" +
		"version: \"v1\"\n" +
		"description: rabbitmq cluster lifecycle integration test\n\n" +
		"runtime:\n" +
		"  project-name: rabbitmq-cluster-test\n\n" +
		"services:\n" +
		"  - name: rabbitmq-cluster-1\n" +
		"    middleware: rabbitmq\n" +
		"    template: cluster\n" +
		"    values:\n" +
		"      version: v4.2\n" +
		"      rabbit1_amqp_port: " + strconv.Itoa(rabbit1AMQPPort) + "\n" +
		"      rabbit2_amqp_port: " + strconv.Itoa(rabbit2AMQPPort) + "\n" +
		"      rabbit3_amqp_port: " + strconv.Itoa(rabbit3AMQPPort) + "\n" +
		"      rabbit1_management_port: " + strconv.Itoa(rabbit1ManagementPort) + "\n" +
		"      rabbit2_management_port: " + strconv.Itoa(rabbit2ManagementPort) + "\n" +
		"      rabbit3_management_port: " + strconv.Itoa(rabbit3ManagementPort) + "\n" +
		"      default_user: admin\n" +
		"      default_pass: admin123\n" +
		"      erlang_cookie: rabbitmq-cookie\n" +
		"      rabbit1_data_dir: ./data/rabbit1\n" +
		"      rabbit2_data_dir: ./data/rabbit2\n" +
		"      rabbit3_data_dir: ./data/rabbit3\n"
	if image != "" {
		blueprint += "      image: " + image + "\n"
	}
	return blueprint
}

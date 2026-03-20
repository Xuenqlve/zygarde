package command

import (
	"strconv"
	"testing"
	"time"
)

func TestKafkaSingleUpDoctorDown(t *testing.T) {
	tc := newLifecycleTestContext(t)

	kafkaPort := freePort(t)
	kafkaImage := localImageOrSkip(
		t,
		"apache/kafka:4.2.0",
		"localhost/apache/kafka:4.2.0",
		"apache/kafka:4.2.0",
		"docker.io/apache/kafka:4.2.0",
	)

	t.Logf("kafka port: %d", kafkaPort)
	t.Logf("kafka image: %s", kafkaImage)
	blueprintPath := tc.writeBlueprint(kafkaBlueprint(kafkaPort, kafkaImage))

	upResult := tc.up(blueprintPath)
	tc.verifyCurrentEnvironment(upResult)
	_, artifact := tc.verifyRunningEnvironment(upResult)
	tc.verifyRuntimeFiles(artifact)
	tc.verifyStatusRunning()
	tc.waitForDoctorPass(3 * time.Minute)
	tc.downAndVerify(upResult)
}

func TestKafkaClusterUpDoctorDown(t *testing.T) {
	tc := newLifecycleTestContext(t)

	kafka1Port := freePort(t)
	kafka2Port := freePort(t)
	kafka3Port := freePort(t)
	kafkaImage := localImageOrSkip(
		t,
		"apache/kafka:4.2.0",
		"localhost/apache/kafka:4.2.0",
		"apache/kafka:4.2.0",
		"docker.io/apache/kafka:4.2.0",
	)

	t.Logf("kafka cluster ports: %d, %d, %d", kafka1Port, kafka2Port, kafka3Port)
	t.Logf("kafka image: %s", kafkaImage)
	blueprintPath := tc.writeBlueprint(kafkaClusterBlueprint(kafka1Port, kafka2Port, kafka3Port, kafkaImage))

	upResult := tc.up(blueprintPath)
	tc.verifyCurrentEnvironment(upResult)
	_, artifact := tc.verifyRunningEnvironment(upResult)
	tc.verifyRuntimeFiles(artifact)
	tc.verifyStatusRunning()
	tc.waitForDoctorPass(4 * time.Minute)
	tc.downAndVerify(upResult)
}

func kafkaBlueprint(port int, image string) string {
	blueprint := "name: kafka-test\n" +
		"version: \"v1\"\n" +
		"description: kafka single lifecycle integration test\n\n" +
		"runtime:\n" +
		"  project-name: kafka-test\n\n" +
		"services:\n" +
		"  - name: kafka-1\n" +
		"    middleware: kafka\n" +
		"    template: single\n" +
		"    values:\n" +
		"      version: v4.2\n" +
		"      port: " + strconv.Itoa(port) + "\n" +
		"      cluster_id: MkU3OEVBNTcwNTJENDM2Qk\n" +
		"      data_dir: ./data/kafka-1\n"
	if image != "" {
		blueprint += "      image: " + image + "\n"
	}
	return blueprint
}

func kafkaClusterBlueprint(kafka1Port, kafka2Port, kafka3Port int, image string) string {
	blueprint := "name: kafka-cluster-test\n" +
		"version: \"v1\"\n" +
		"description: kafka cluster lifecycle integration test\n\n" +
		"runtime:\n" +
		"  project-name: kafka-cluster-test\n\n" +
		"services:\n" +
		"  - name: kafka-cluster-1\n" +
		"    middleware: kafka\n" +
		"    template: cluster\n" +
		"    values:\n" +
		"      version: v4.2\n" +
		"      cluster_id: MkU3OEVBNTcwNTJENDM2Qk\n" +
		"      kafka1_port: " + strconv.Itoa(kafka1Port) + "\n" +
		"      kafka2_port: " + strconv.Itoa(kafka2Port) + "\n" +
		"      kafka3_port: " + strconv.Itoa(kafka3Port) + "\n" +
		"      kafka1_data_dir: ./data/kafka1\n" +
		"      kafka2_data_dir: ./data/kafka2\n" +
		"      kafka3_data_dir: ./data/kafka3\n"
	if image != "" {
		blueprint += "      image: " + image + "\n"
	}
	return blueprint
}

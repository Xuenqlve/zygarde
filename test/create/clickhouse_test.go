package create

import (
	"strconv"
	"testing"
	"time"
)

func TestClickHouseSingleUpDoctorDown(t *testing.T) {
	tc := newLifecycleTestContext(t)

	httpPort := freePort(t)
	tcpPort := freePort(t)
	image := localImageOrSkip(
		t,
		"clickhouse/clickhouse-server:25.8",
		"localhost/clickhouse/clickhouse-server:25.8",
		"clickhouse/clickhouse-server:25.8",
		"docker.io/clickhouse/clickhouse-server:25.8",
	)

	t.Logf("clickhouse ports: http=%d tcp=%d", httpPort, tcpPort)
	t.Logf("clickhouse image: %s", image)
	blueprintPath := tc.writeBlueprint(clickhouseBlueprint(httpPort, tcpPort, image))

	upResult := tc.up(blueprintPath)
	tc.verifyCurrentEnvironment(upResult)
	_, artifact := tc.verifyRunningEnvironment(upResult)
	tc.verifyRuntimeFiles(artifact)
	tc.verifyStatusRunning()
	tc.waitForDoctorPass(4 * time.Minute)
	tc.downAndVerify(upResult)
}

func TestClickHouseClusterUpDoctorDown(t *testing.T) {
	tc := newLifecycleTestContext(t)

	ch1HTTPPort := freePort(t)
	ch2HTTPPort := freePort(t)
	ch3HTTPPort := freePort(t)
	ch1TCPPort := freePort(t)
	ch2TCPPort := freePort(t)
	ch3TCPPort := freePort(t)
	image := localImageOrSkip(
		t,
		"clickhouse/clickhouse-server:25.8",
		"localhost/clickhouse/clickhouse-server:25.8",
		"clickhouse/clickhouse-server:25.8",
		"docker.io/clickhouse/clickhouse-server:25.8",
	)

	t.Logf("clickhouse cluster ports: ch1_http=%d ch2_http=%d ch3_http=%d ch1_tcp=%d ch2_tcp=%d ch3_tcp=%d", ch1HTTPPort, ch2HTTPPort, ch3HTTPPort, ch1TCPPort, ch2TCPPort, ch3TCPPort)
	t.Logf("clickhouse image: %s", image)
	blueprintPath := tc.writeBlueprint(clickhouseClusterBlueprint(ch1HTTPPort, ch2HTTPPort, ch3HTTPPort, ch1TCPPort, ch2TCPPort, ch3TCPPort, image))

	upResult := tc.up(blueprintPath)
	tc.verifyCurrentEnvironment(upResult)
	_, artifact := tc.verifyRunningEnvironment(upResult)
	tc.verifyRuntimeFiles(artifact)
	tc.verifyStatusRunning()
	tc.waitForDoctorPass(4 * time.Minute)
	tc.downAndVerify(upResult)
}

func clickhouseBlueprint(httpPort, tcpPort int, image string) string {
	blueprint := "name: clickhouse-test\n" +
		"version: \"v1\"\n" +
		"description: clickhouse single lifecycle integration test\n\n" +
		"runtime:\n" +
		"  project-name: clickhouse-test\n\n" +
		"services:\n" +
		"  - name: clickhouse-1\n" +
		"    middleware: clickhouse\n" +
		"    template: single\n" +
		"    values:\n" +
		"      version: v25\n" +
		"      http_port: " + strconv.Itoa(httpPort) + "\n" +
		"      tcp_port: " + strconv.Itoa(tcpPort) + "\n" +
		"      data_dir: ./data/clickhouse\n"
	if image != "" {
		blueprint += "      image: " + image + "\n"
	}
	return blueprint
}

func clickhouseClusterBlueprint(ch1HTTPPort, ch2HTTPPort, ch3HTTPPort, ch1TCPPort, ch2TCPPort, ch3TCPPort int, image string) string {
	blueprint := "name: clickhouse-cluster-test\n" +
		"version: \"v1\"\n" +
		"description: clickhouse cluster lifecycle integration test\n\n" +
		"runtime:\n" +
		"  project-name: clickhouse-cluster-test\n\n" +
		"services:\n" +
		"  - name: clickhouse-cluster-1\n" +
		"    middleware: clickhouse\n" +
		"    template: cluster\n" +
		"    values:\n" +
		"      version: v25\n" +
		"      ch1_http_port: " + strconv.Itoa(ch1HTTPPort) + "\n" +
		"      ch2_http_port: " + strconv.Itoa(ch2HTTPPort) + "\n" +
		"      ch3_http_port: " + strconv.Itoa(ch3HTTPPort) + "\n" +
		"      ch1_tcp_port: " + strconv.Itoa(ch1TCPPort) + "\n" +
		"      ch2_tcp_port: " + strconv.Itoa(ch2TCPPort) + "\n" +
		"      ch3_tcp_port: " + strconv.Itoa(ch3TCPPort) + "\n" +
		"      ch1_data_dir: ./data/ch1\n" +
		"      ch2_data_dir: ./data/ch2\n" +
		"      ch3_data_dir: ./data/ch3\n"
	if image != "" {
		blueprint += "      image: " + image + "\n"
	}
	return blueprint
}

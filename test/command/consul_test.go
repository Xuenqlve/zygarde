package command

import (
	"strconv"
	"testing"
	"time"
)

func TestConsulSingleUpDoctorDown(t *testing.T) {
	tc := newLifecycleTestContext(t)

	httpPort := freePort(t)
	dnsPort := freePort(t)
	serverPort := freePort(t)
	consulImage := localImageOrSkip(
		t,
		"hashicorp/consul:1.20",
		"localhost/hashicorp/consul:1.20",
		"hashicorp/consul:1.20",
		"docker.io/hashicorp/consul:1.20",
	)

	t.Logf("consul ports: http=%d dns=%d server=%d", httpPort, dnsPort, serverPort)
	t.Logf("consul image: %s", consulImage)
	blueprintPath := tc.writeBlueprint(consulBlueprint(httpPort, dnsPort, serverPort, consulImage))

	upResult := tc.up(blueprintPath)
	tc.verifyCurrentEnvironment(upResult)
	_, artifact := tc.verifyRunningEnvironment(upResult)
	tc.verifyRuntimeFiles(artifact)
	tc.verifyStatusRunning()
	tc.waitForDoctorPass(3 * time.Minute)
	tc.downAndVerify(upResult)
}

func TestConsulClusterUpDoctorDown(t *testing.T) {
	tc := newLifecycleTestContext(t)

	consul1HTTPPort := freePort(t)
	consul1DNSPort := freePort(t)
	consul2HTTPPort := freePort(t)
	consul3HTTPPort := freePort(t)
	consulImage := localImageOrSkip(
		t,
		"hashicorp/consul:1.20",
		"localhost/hashicorp/consul:1.20",
		"hashicorp/consul:1.20",
		"docker.io/hashicorp/consul:1.20",
	)

	t.Logf("consul cluster ports: http1=%d dns1=%d http2=%d http3=%d", consul1HTTPPort, consul1DNSPort, consul2HTTPPort, consul3HTTPPort)
	t.Logf("consul image: %s", consulImage)
	blueprintPath := tc.writeBlueprint(consulClusterBlueprint(consul1HTTPPort, consul1DNSPort, consul2HTTPPort, consul3HTTPPort, consulImage))

	upResult := tc.up(blueprintPath)
	tc.verifyCurrentEnvironment(upResult)
	_, artifact := tc.verifyRunningEnvironment(upResult)
	tc.verifyRuntimeFiles(artifact)
	tc.verifyStatusRunning()
	tc.waitForDoctorPass(4 * time.Minute)
	tc.downAndVerify(upResult)
}

func consulBlueprint(httpPort, dnsPort, serverPort int, image string) string {
	blueprint := "name: consul-test\n" +
		"version: \"v1\"\n" +
		"description: consul single lifecycle integration test\n\n" +
		"runtime:\n" +
		"  project-name: consul-test\n\n" +
		"services:\n" +
		"  - name: consul-1\n" +
		"    middleware: consul\n" +
		"    template: single\n" +
		"    values:\n" +
		"      version: v1.20\n" +
		"      http_port: " + strconv.Itoa(httpPort) + "\n" +
		"      dns_port: " + strconv.Itoa(dnsPort) + "\n" +
		"      server_port: " + strconv.Itoa(serverPort) + "\n" +
		"      data_dir: ./data/consul\n"
	if image != "" {
		blueprint += "      image: " + image + "\n"
	}
	return blueprint
}

func consulClusterBlueprint(consul1HTTPPort, consul1DNSPort, consul2HTTPPort, consul3HTTPPort int, image string) string {
	blueprint := "name: consul-cluster-test\n" +
		"version: \"v1\"\n" +
		"description: consul cluster lifecycle integration test\n\n" +
		"runtime:\n" +
		"  project-name: consul-cluster-test\n\n" +
		"services:\n" +
		"  - name: consul-cluster-1\n" +
		"    middleware: consul\n" +
		"    template: cluster\n" +
		"    values:\n" +
		"      version: v1.20\n" +
		"      consul1_http_port: " + strconv.Itoa(consul1HTTPPort) + "\n" +
		"      consul1_dns_port: " + strconv.Itoa(consul1DNSPort) + "\n" +
		"      consul2_http_port: " + strconv.Itoa(consul2HTTPPort) + "\n" +
		"      consul3_http_port: " + strconv.Itoa(consul3HTTPPort) + "\n" +
		"      consul1_data_dir: ./data/consul1\n" +
		"      consul2_data_dir: ./data/consul2\n" +
		"      consul3_data_dir: ./data/consul3\n"
	if image != "" {
		blueprint += "      image: " + image + "\n"
	}
	return blueprint
}

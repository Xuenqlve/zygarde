package command

import (
	"strconv"
	"testing"
	"time"
)

func TestEtcdSingleUpDoctorDown(t *testing.T) {
	tc := newLifecycleTestContext(t)

	clientPort := freePort(t)
	peerPort := freePort(t)
	for peerPort == clientPort {
		peerPort = freePort(t)
	}
	etcdImage := localImageOrSkip(
		t,
		"quay.io/coreos/etcd:v3.6.0",
		"localhost/quay.io/coreos/etcd:v3.6.0",
		"quay.io/coreos/etcd:v3.6.0",
	)

	t.Logf("etcd client port: %d", clientPort)
	t.Logf("etcd peer port: %d", peerPort)
	t.Logf("etcd image: %s", etcdImage)
	blueprintPath := tc.writeBlueprint(etcdBlueprint(clientPort, peerPort, etcdImage))

	upResult := tc.up(blueprintPath)
	tc.verifyCurrentEnvironment(upResult)
	_, artifact := tc.verifyRunningEnvironment(upResult)
	tc.verifyRuntimeFiles(artifact)
	tc.verifyStatusRunning()
	tc.waitForDoctorPass(3 * time.Minute)
	tc.downAndVerify(upResult)
}

func TestEtcdClusterUpDoctorDown(t *testing.T) {
	tc := newLifecycleTestContext(t)

	etcd1ClientPort := freePort(t)
	etcd2ClientPort := freePort(t)
	etcd3ClientPort := freePort(t)
	etcdImage := localImageOrSkip(
		t,
		"quay.io/coreos/etcd:v3.6.0",
		"localhost/quay.io/coreos/etcd:v3.6.0",
		"quay.io/coreos/etcd:v3.6.0",
	)

	t.Logf("etcd cluster client ports: %d, %d, %d", etcd1ClientPort, etcd2ClientPort, etcd3ClientPort)
	t.Logf("etcd image: %s", etcdImage)
	blueprintPath := tc.writeBlueprint(etcdClusterBlueprint(etcd1ClientPort, etcd2ClientPort, etcd3ClientPort, etcdImage))

	upResult := tc.up(blueprintPath)
	tc.verifyCurrentEnvironment(upResult)
	_, artifact := tc.verifyRunningEnvironment(upResult)
	tc.verifyRuntimeFiles(artifact)
	tc.verifyStatusRunning()
	tc.waitForDoctorPass(4 * time.Minute)
	tc.downAndVerify(upResult)
}

func etcdBlueprint(clientPort, peerPort int, image string) string {
	blueprint := "name: etcd-test\n" +
		"version: \"v1\"\n" +
		"description: etcd single lifecycle integration test\n\n" +
		"runtime:\n" +
		"  project-name: etcd-test\n\n" +
		"services:\n" +
		"  - name: etcd-1\n" +
		"    middleware: etcd\n" +
		"    template: single\n" +
		"    values:\n" +
		"      version: v3.6\n" +
		"      client_port: " + strconv.Itoa(clientPort) + "\n" +
		"      peer_port: " + strconv.Itoa(peerPort) + "\n" +
		"      cluster_token: zygarde-etcd-single\n" +
		"      data_dir: ./data/etcd\n"
	if image != "" {
		blueprint += "      image: " + image + "\n"
	}
	return blueprint
}

func etcdClusterBlueprint(etcd1ClientPort, etcd2ClientPort, etcd3ClientPort int, image string) string {
	blueprint := "name: etcd-cluster-test\n" +
		"version: \"v1\"\n" +
		"description: etcd cluster lifecycle integration test\n\n" +
		"runtime:\n" +
		"  project-name: etcd-cluster-test\n\n" +
		"services:\n" +
		"  - name: etcd-cluster-1\n" +
		"    middleware: etcd\n" +
		"    template: cluster\n" +
		"    values:\n" +
		"      version: v3.6\n" +
		"      etcd1_client_port: " + strconv.Itoa(etcd1ClientPort) + "\n" +
		"      etcd2_client_port: " + strconv.Itoa(etcd2ClientPort) + "\n" +
		"      etcd3_client_port: " + strconv.Itoa(etcd3ClientPort) + "\n" +
		"      cluster_token: zygarde-etcd-cluster\n" +
		"      etcd1_data_dir: ./data/etcd1\n" +
		"      etcd2_data_dir: ./data/etcd2\n" +
		"      etcd3_data_dir: ./data/etcd3\n"
	if image != "" {
		blueprint += "      image: " + image + "\n"
	}
	return blueprint
}

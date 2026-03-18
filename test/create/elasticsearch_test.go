package create

import (
	"strconv"
	"testing"
	"time"
)

func TestElasticsearchSingleUpDoctorDown(t *testing.T) {
	tc := newLifecycleTestContext(t)

	httpPort := freePort(t)
	transportPort := freePort(t)
	image := localImageOrSkip(
		t,
		"docker.elastic.co/elasticsearch/elasticsearch:8.19.0",
		"localhost/docker.elastic.co/elasticsearch/elasticsearch:8.19.0",
		"docker.elastic.co/elasticsearch/elasticsearch:8.19.0",
	)

	t.Logf("elasticsearch ports: http=%d transport=%d", httpPort, transportPort)
	t.Logf("elasticsearch image: %s", image)
	blueprintPath := tc.writeBlueprint(elasticsearchBlueprint(httpPort, transportPort, image))

	upResult := tc.up(blueprintPath)
	tc.verifyCurrentEnvironment(upResult)
	_, artifact := tc.verifyRunningEnvironment(upResult)
	tc.verifyRuntimeFiles(artifact)
	tc.verifyStatusRunning()
	tc.waitForDoctorPass(5 * time.Minute)
	tc.downAndVerify(upResult)
}

func TestElasticsearchClusterUpDoctorDown(t *testing.T) {
	tc := newLifecycleTestContext(t)

	es1HTTPPort := freePort(t)
	es2HTTPPort := freePort(t)
	es3HTTPPort := freePort(t)
	image := localImageOrSkip(
		t,
		"docker.elastic.co/elasticsearch/elasticsearch:8.19.0",
		"localhost/docker.elastic.co/elasticsearch/elasticsearch:8.19.0",
		"docker.elastic.co/elasticsearch/elasticsearch:8.19.0",
	)

	t.Logf("elasticsearch cluster ports: es1=%d es2=%d es3=%d", es1HTTPPort, es2HTTPPort, es3HTTPPort)
	t.Logf("elasticsearch image: %s", image)
	blueprintPath := tc.writeBlueprint(elasticsearchClusterBlueprint(es1HTTPPort, es2HTTPPort, es3HTTPPort, image))

	upResult := tc.up(blueprintPath)
	tc.verifyCurrentEnvironment(upResult)
	_, artifact := tc.verifyRunningEnvironment(upResult)
	tc.verifyRuntimeFiles(artifact)
	tc.verifyStatusRunning()
	tc.waitForDoctorPass(5 * time.Minute)
	tc.downAndVerify(upResult)
}

func elasticsearchBlueprint(httpPort, transportPort int, image string) string {
	blueprint := "name: elasticsearch-test\n" +
		"version: \"v1\"\n" +
		"description: elasticsearch single lifecycle integration test\n\n" +
		"runtime:\n" +
		"  project-name: elasticsearch-test\n\n" +
		"services:\n" +
		"  - name: es-1\n" +
		"    middleware: elasticsearch\n" +
		"    template: single\n" +
		"    values:\n" +
		"      version: v8.19\n" +
		"      http_port: " + strconv.Itoa(httpPort) + "\n" +
		"      transport_port: " + strconv.Itoa(transportPort) + "\n" +
		"      data_dir: ./data/es\n"
	if image != "" {
		blueprint += "      image: " + image + "\n"
	}
	return blueprint
}

func elasticsearchClusterBlueprint(es1HTTPPort, es2HTTPPort, es3HTTPPort int, image string) string {
	blueprint := "name: elasticsearch-cluster-test\n" +
		"version: \"v1\"\n" +
		"description: elasticsearch cluster lifecycle integration test\n\n" +
		"runtime:\n" +
		"  project-name: elasticsearch-cluster-test\n\n" +
		"services:\n" +
		"  - name: es-cluster-1\n" +
		"    middleware: elasticsearch\n" +
		"    template: cluster\n" +
		"    values:\n" +
		"      version: v8.19\n" +
		"      es1_http_port: " + strconv.Itoa(es1HTTPPort) + "\n" +
		"      es2_http_port: " + strconv.Itoa(es2HTTPPort) + "\n" +
		"      es3_http_port: " + strconv.Itoa(es3HTTPPort) + "\n" +
		"      es1_data_dir: ./data/es1\n" +
		"      es2_data_dir: ./data/es2\n" +
		"      es3_data_dir: ./data/es3\n"
	if image != "" {
		blueprint += "      image: " + image + "\n"
	}
	return blueprint
}

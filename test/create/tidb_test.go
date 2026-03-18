package create

import (
	"strconv"
	"testing"
	"time"
)

func TestTiDBSingleUpDoctorDown(t *testing.T) {
	tc := newLifecycleTestContext(t)

	pdPort := freePort(t)
	tikvPort := freePort(t)
	tidbPort := freePort(t)
	tidbStatusPort := freePort(t)

	pdImage := localImageOrSkip(
		t,
		"pingcap/pd:v6.5.12",
		"localhost/pingcap/pd:v6.5.12",
		"pingcap/pd:v6.5.12",
		"docker.io/pingcap/pd:v6.5.12",
	)
	tikvImage := localImageOrSkip(
		t,
		"pingcap/tikv:v6.5.12",
		"localhost/pingcap/tikv:v6.5.12",
		"pingcap/tikv:v6.5.12",
		"docker.io/pingcap/tikv:v6.5.12",
	)
	tidbImage := localImageOrSkip(
		t,
		"pingcap/tidb:v6.5.12",
		"localhost/pingcap/tidb:v6.5.12",
		"pingcap/tidb:v6.5.12",
		"docker.io/pingcap/tidb:v6.5.12",
	)

	t.Logf("tidb ports: pd=%d tikv=%d tidb=%d status=%d", pdPort, tikvPort, tidbPort, tidbStatusPort)
	t.Logf("tidb images: pd=%s tikv=%s tidb=%s", pdImage, tikvImage, tidbImage)
	blueprintPath := tc.writeBlueprint(tidbBlueprint(pdPort, tikvPort, tidbPort, tidbStatusPort, pdImage, tikvImage, tidbImage))

	upResult := tc.up(blueprintPath)
	tc.verifyCurrentEnvironment(upResult)
	_, artifact := tc.verifyRunningEnvironment(upResult)
	tc.verifyRuntimeFiles(artifact)
	tc.verifyStatusRunning()
	tc.waitForDoctorPass(4 * time.Minute)
	tc.downAndVerify(upResult)
}

func TestTiDBClusterUpDoctorDown(t *testing.T) {
	tc := newLifecycleTestContext(t)

	pd1Port := freePort(t)
	pd2Port := freePort(t)
	pd3Port := freePort(t)
	tidb1Port := freePort(t)
	tidb2Port := freePort(t)
	tidb1StatusPort := freePort(t)
	tidb2StatusPort := freePort(t)

	pdImage := localImageOrSkip(
		t,
		"pingcap/pd:v6.5.12",
		"localhost/pingcap/pd:v6.5.12",
		"pingcap/pd:v6.5.12",
		"docker.io/pingcap/pd:v6.5.12",
	)
	tikvImage := localImageOrSkip(
		t,
		"pingcap/tikv:v6.5.12",
		"localhost/pingcap/tikv:v6.5.12",
		"pingcap/tikv:v6.5.12",
		"docker.io/pingcap/tikv:v6.5.12",
	)
	tidbImage := localImageOrSkip(
		t,
		"pingcap/tidb:v6.5.12",
		"localhost/pingcap/tidb:v6.5.12",
		"pingcap/tidb:v6.5.12",
		"docker.io/pingcap/tidb:v6.5.12",
	)

	t.Logf("tidb cluster ports: pd1=%d pd2=%d pd3=%d tidb1=%d tidb2=%d status1=%d status2=%d", pd1Port, pd2Port, pd3Port, tidb1Port, tidb2Port, tidb1StatusPort, tidb2StatusPort)
	t.Logf("tidb images: pd=%s tikv=%s tidb=%s", pdImage, tikvImage, tidbImage)
	blueprintPath := tc.writeBlueprint(tidbClusterBlueprint(pd1Port, pd2Port, pd3Port, tidb1Port, tidb2Port, tidb1StatusPort, tidb2StatusPort, pdImage, tikvImage, tidbImage))

	upResult := tc.up(blueprintPath)
	tc.verifyCurrentEnvironment(upResult)
	_, artifact := tc.verifyRunningEnvironment(upResult)
	tc.verifyRuntimeFiles(artifact)
	tc.verifyStatusRunning()
	tc.waitForDoctorPass(5 * time.Minute)
	tc.downAndVerify(upResult)
}

func tidbBlueprint(pdPort, tikvPort, tidbPort, tidbStatusPort int, pdImage, tikvImage, tidbImage string) string {
	blueprint := "name: tidb-test\n" +
		"version: \"v1\"\n" +
		"description: tidb single lifecycle integration test\n\n" +
		"runtime:\n" +
		"  project-name: tidb-test\n\n" +
		"services:\n" +
		"  - name: tidb-1\n" +
		"    middleware: tidb\n" +
		"    template: single\n" +
		"    values:\n" +
		"      version: v6.7\n" +
		"      pd_port: " + strconv.Itoa(pdPort) + "\n" +
		"      tikv_port: " + strconv.Itoa(tikvPort) + "\n" +
		"      tidb_port: " + strconv.Itoa(tidbPort) + "\n" +
		"      tidb_status_port: " + strconv.Itoa(tidbStatusPort) + "\n" +
		"      pd_data_dir: ./data/pd\n" +
		"      tikv_data_dir: ./data/tikv\n"
	if pdImage != "" {
		blueprint += "      pd_image: " + pdImage + "\n"
	}
	if tikvImage != "" {
		blueprint += "      tikv_image: " + tikvImage + "\n"
	}
	if tidbImage != "" {
		blueprint += "      tidb_image: " + tidbImage + "\n"
	}
	return blueprint
}

func tidbClusterBlueprint(pd1Port, pd2Port, pd3Port, tidb1Port, tidb2Port, tidb1StatusPort, tidb2StatusPort int, pdImage, tikvImage, tidbImage string) string {
	blueprint := "name: tidb-cluster-test\n" +
		"version: \"v1\"\n" +
		"description: tidb cluster lifecycle integration test\n\n" +
		"runtime:\n" +
		"  project-name: tidb-cluster-test\n\n" +
		"services:\n" +
		"  - name: tidb-cluster-1\n" +
		"    middleware: tidb\n" +
		"    template: cluster\n" +
		"    values:\n" +
		"      version: v6.7\n" +
		"      pd1_port: " + strconv.Itoa(pd1Port) + "\n" +
		"      pd2_port: " + strconv.Itoa(pd2Port) + "\n" +
		"      pd3_port: " + strconv.Itoa(pd3Port) + "\n" +
		"      tidb1_port: " + strconv.Itoa(tidb1Port) + "\n" +
		"      tidb2_port: " + strconv.Itoa(tidb2Port) + "\n" +
		"      tidb1_status_port: " + strconv.Itoa(tidb1StatusPort) + "\n" +
		"      tidb2_status_port: " + strconv.Itoa(tidb2StatusPort) + "\n" +
		"      pd1_data_dir: ./data/pd1\n" +
		"      pd2_data_dir: ./data/pd2\n" +
		"      pd3_data_dir: ./data/pd3\n" +
		"      tikv1_data_dir: ./data/tikv1\n" +
		"      tikv2_data_dir: ./data/tikv2\n" +
		"      tikv3_data_dir: ./data/tikv3\n"
	if pdImage != "" {
		blueprint += "      pd_image: " + pdImage + "\n"
	}
	if tikvImage != "" {
		blueprint += "      tikv_image: " + tikvImage + "\n"
	}
	if tidbImage != "" {
		blueprint += "      tidb_image: " + tidbImage + "\n"
	}
	return blueprint
}

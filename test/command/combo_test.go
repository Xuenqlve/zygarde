package command

import (
	"strconv"
	"testing"
	"time"
)

func TestComboMySQLRedisUpDoctorDown(t *testing.T) {
	tc := newLifecycleTestContext(t)

	mysqlPort := freePort(t)
	redisPort := freePort(t)
	for redisPort == mysqlPort {
		redisPort = freePort(t)
	}

	mysqlImage := localImageOrSkip(
		t,
		"mysql:5.7",
		"localhost/mysql:5.7",
		"mysql:5.7",
		"docker.io/library/mysql:5.7",
	)
	redisImage := localImageOrSkip(
		t,
		"redis:6.2",
		"localhost/redis:6.2",
		"redis:6.2",
		"docker.io/library/redis:6.2",
	)

	t.Logf("combo mysql port: %d", mysqlPort)
	t.Logf("combo redis port: %d", redisPort)
	t.Logf("combo mysql image: %s", mysqlImage)
	t.Logf("combo redis image: %s", redisImage)

	blueprintPath := tc.writeBlueprint(comboMySQLRedisBlueprint(mysqlPort, redisPort, mysqlImage, redisImage))

	upResult := tc.up(blueprintPath)
	tc.verifyCurrentEnvironment(upResult)
	env, artifact := tc.verifyRunningEnvironment(upResult)
	tc.verifyRuntimeFiles(artifact)
	if len(env.Endpoints) < 2 {
		t.Fatalf("expected at least 2 endpoints for combo environment, got %+v", env.Endpoints)
	}
	tc.verifyStatusRunning()
	tc.waitForDoctorPass(3 * time.Minute)
	tc.downAndVerify(upResult)
}

func TestComboMySQLRedisCreateStartDoctorDown(t *testing.T) {
	tc := newLifecycleTestContext(t)

	mysqlPort := freePort(t)
	redisPort := freePort(t)
	for redisPort == mysqlPort {
		redisPort = freePort(t)
	}

	mysqlImage := localImageOrSkip(
		t,
		"mysql:5.7",
		"localhost/mysql:5.7",
		"mysql:5.7",
		"docker.io/library/mysql:5.7",
	)
	redisImage := localImageOrSkip(
		t,
		"redis:6.2",
		"localhost/redis:6.2",
		"redis:6.2",
		"docker.io/library/redis:6.2",
	)

	t.Logf("combo mysql port: %d", mysqlPort)
	t.Logf("combo redis port: %d", redisPort)
	t.Logf("combo mysql image: %s", mysqlImage)
	t.Logf("combo redis image: %s", redisImage)

	blueprintPath := tc.writeBlueprint(comboMySQLRedisBlueprint(mysqlPort, redisPort, mysqlImage, redisImage))

	createResult := tc.create(blueprintPath)
	tc.verifyCurrentEnvironment(createResult)
	_, artifact := tc.verifyStoppedEnvironment(createResult)
	tc.verifyRuntimeFiles(artifact)
	tc.verifyStatusStopped()
	tc.startAndVerify(createResult)
	env, _ := tc.verifyRunningEnvironment(createResult)
	if len(env.Endpoints) < 2 {
		t.Fatalf("expected at least 2 endpoints after combo start, got %+v", env.Endpoints)
	}
	tc.waitForDoctorPass(3 * time.Minute)
	tc.downAndVerify(createResult)
}

func TestComboMySQLRedisRabbitMQUpDoctorDown(t *testing.T) {
	tc := newLifecycleTestContext(t)

	mysqlPort := freePort(t)
	redisPort := freePort(t)
	for redisPort == mysqlPort {
		redisPort = freePort(t)
	}
	amqpPort := freePort(t)
	for amqpPort == mysqlPort || amqpPort == redisPort {
		amqpPort = freePort(t)
	}
	managementPort := freePort(t)
	for managementPort == mysqlPort || managementPort == redisPort || managementPort == amqpPort {
		managementPort = freePort(t)
	}

	mysqlImage := localImageOrSkip(
		t,
		"mysql:5.7",
		"localhost/mysql:5.7",
		"mysql:5.7",
		"docker.io/library/mysql:5.7",
	)
	redisImage := localImageOrSkip(
		t,
		"redis:6.2",
		"localhost/redis:6.2",
		"redis:6.2",
		"docker.io/library/redis:6.2",
	)
	rabbitmqImage := localImageOrSkip(
		t,
		"rabbitmq:4.2-management",
		"localhost/rabbitmq:4.2-management",
		"rabbitmq:4.2-management",
		"docker.io/library/rabbitmq:4.2-management",
	)

	t.Logf("combo mysql port: %d", mysqlPort)
	t.Logf("combo redis port: %d", redisPort)
	t.Logf("combo rabbitmq amqp port: %d", amqpPort)
	t.Logf("combo rabbitmq management port: %d", managementPort)
	t.Logf("combo mysql image: %s", mysqlImage)
	t.Logf("combo redis image: %s", redisImage)
	t.Logf("combo rabbitmq image: %s", rabbitmqImage)

	blueprintPath := tc.writeBlueprint(comboMySQLRedisRabbitMQBlueprint(
		mysqlPort,
		redisPort,
		amqpPort,
		managementPort,
		mysqlImage,
		redisImage,
		rabbitmqImage,
	))

	upResult := tc.up(blueprintPath)
	tc.verifyCurrentEnvironment(upResult)
	env, artifact := tc.verifyRunningEnvironment(upResult)
	tc.verifyRuntimeFiles(artifact)
	if len(env.Endpoints) < 4 {
		t.Fatalf("expected at least 4 endpoints for combo environment, got %+v", env.Endpoints)
	}
	tc.verifyStatusRunning()
	tc.waitForDoctorPass(4 * time.Minute)
	tc.downAndVerify(upResult)
}

func TestComboMySQLRedisRabbitMQCreateStartDoctorDown(t *testing.T) {
	tc := newLifecycleTestContext(t)

	mysqlPort := freePort(t)
	redisPort := freePort(t)
	for redisPort == mysqlPort {
		redisPort = freePort(t)
	}
	amqpPort := freePort(t)
	for amqpPort == mysqlPort || amqpPort == redisPort {
		amqpPort = freePort(t)
	}
	managementPort := freePort(t)
	for managementPort == mysqlPort || managementPort == redisPort || managementPort == amqpPort {
		managementPort = freePort(t)
	}

	mysqlImage := localImageOrSkip(
		t,
		"mysql:5.7",
		"localhost/mysql:5.7",
		"mysql:5.7",
		"docker.io/library/mysql:5.7",
	)
	redisImage := localImageOrSkip(
		t,
		"redis:6.2",
		"localhost/redis:6.2",
		"redis:6.2",
		"docker.io/library/redis:6.2",
	)
	rabbitmqImage := localImageOrSkip(
		t,
		"rabbitmq:4.2-management",
		"localhost/rabbitmq:4.2-management",
		"rabbitmq:4.2-management",
		"docker.io/library/rabbitmq:4.2-management",
	)

	t.Logf("combo mysql port: %d", mysqlPort)
	t.Logf("combo redis port: %d", redisPort)
	t.Logf("combo rabbitmq amqp port: %d", amqpPort)
	t.Logf("combo rabbitmq management port: %d", managementPort)
	t.Logf("combo mysql image: %s", mysqlImage)
	t.Logf("combo redis image: %s", redisImage)
	t.Logf("combo rabbitmq image: %s", rabbitmqImage)

	blueprintPath := tc.writeBlueprint(comboMySQLRedisRabbitMQBlueprint(
		mysqlPort,
		redisPort,
		amqpPort,
		managementPort,
		mysqlImage,
		redisImage,
		rabbitmqImage,
	))

	createResult := tc.create(blueprintPath)
	tc.verifyCurrentEnvironment(createResult)
	_, artifact := tc.verifyStoppedEnvironment(createResult)
	tc.verifyRuntimeFiles(artifact)
	tc.verifyStatusStopped()
	tc.startAndVerify(createResult)
	env, _ := tc.verifyRunningEnvironment(createResult)
	if len(env.Endpoints) < 4 {
		t.Fatalf("expected at least 4 endpoints after combo start, got %+v", env.Endpoints)
	}
	tc.waitForDoctorPass(4 * time.Minute)
	tc.downAndVerify(createResult)
}

func TestComboPostgreSQLKafkaUpDoctorDown(t *testing.T) {
	tc := newLifecycleTestContext(t)

	postgresPort := freePort(t)
	kafkaPort := freePort(t)
	for kafkaPort == postgresPort {
		kafkaPort = freePort(t)
	}

	postgresImage := localImageOrSkip(
		t,
		"postgres:16",
		"localhost/postgres:16",
		"postgres:16",
		"docker.io/library/postgres:16",
	)
	kafkaImage := localImageOrSkip(
		t,
		"apache/kafka:4.2.0",
		"localhost/apache/kafka:4.2.0",
		"apache/kafka:4.2.0",
		"docker.io/apache/kafka:4.2.0",
	)

	t.Logf("combo postgresql port: %d", postgresPort)
	t.Logf("combo kafka port: %d", kafkaPort)
	t.Logf("combo postgresql image: %s", postgresImage)
	t.Logf("combo kafka image: %s", kafkaImage)

	blueprintPath := tc.writeBlueprint(comboPostgreSQLKafkaBlueprint(postgresPort, kafkaPort, postgresImage, kafkaImage))

	upResult := tc.up(blueprintPath)
	tc.verifyCurrentEnvironment(upResult)
	env, artifact := tc.verifyRunningEnvironment(upResult)
	tc.verifyRuntimeFiles(artifact)
	if len(env.Endpoints) < 2 {
		t.Fatalf("expected at least 2 endpoints for combo environment, got %+v", env.Endpoints)
	}
	tc.verifyStatusRunning()
	tc.waitForDoctorPass(4 * time.Minute)
	tc.downAndVerify(upResult)
}

func comboMySQLRedisBlueprint(mysqlPort, redisPort int, mysqlImage, redisImage string) string {
	blueprint := "name: combo-mysql-redis-test\n" +
		"version: \"v1\"\n" +
		"description: mysql and redis combo lifecycle integration test\n\n" +
		"runtime:\n" +
		"  project-name: combo-mysql-redis-test\n\n" +
		"services:\n" +
		"  - name: mysql-1\n" +
		"    middleware: mysql\n" +
		"    template: single\n" +
		"    values:\n" +
		"      version: v5.7\n" +
		"      port: " + strconv.Itoa(mysqlPort) + "\n" +
		"      root_password: root123\n" +
		"      data_dir: ./data/mysql-1\n"
	if mysqlImage != "" {
		blueprint += "      image: " + mysqlImage + "\n"
	}
	blueprint +=
		"  - name: redis-1\n" +
			"    middleware: redis\n" +
			"    template: single\n" +
			"    values:\n" +
			"      version: v6.2\n" +
			"      port: " + strconv.Itoa(redisPort) + "\n" +
			"      data_dir: ./data/redis-1\n"
	if redisImage != "" {
		blueprint += "      image: " + redisImage + "\n"
	}
	return blueprint
}

func comboMySQLRedisRabbitMQBlueprint(mysqlPort, redisPort, amqpPort, managementPort int, mysqlImage, redisImage, rabbitmqImage string) string {
	blueprint := "name: combo-mysql-redis-rabbitmq-test\n" +
		"version: \"v1\"\n" +
		"description: mysql redis rabbitmq combo lifecycle integration test\n\n" +
		"runtime:\n" +
		"  project-name: combo-mysql-redis-rabbitmq-test\n\n" +
		"services:\n" +
		"  - name: mysql-1\n" +
		"    middleware: mysql\n" +
		"    template: single\n" +
		"    values:\n" +
		"      version: v5.7\n" +
		"      port: " + strconv.Itoa(mysqlPort) + "\n" +
		"      root_password: root123\n" +
		"      data_dir: ./data/mysql-1\n"
	if mysqlImage != "" {
		blueprint += "      image: " + mysqlImage + "\n"
	}
	blueprint +=
		"  - name: redis-1\n" +
			"    middleware: redis\n" +
			"    template: single\n" +
			"    values:\n" +
			"      version: v6.2\n" +
			"      port: " + strconv.Itoa(redisPort) + "\n" +
			"      data_dir: ./data/redis-1\n"
	if redisImage != "" {
		blueprint += "      image: " + redisImage + "\n"
	}
	blueprint +=
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
	if rabbitmqImage != "" {
		blueprint += "      image: " + rabbitmqImage + "\n"
	}
	return blueprint
}

func comboPostgreSQLKafkaBlueprint(postgresPort, kafkaPort int, postgresImage, kafkaImage string) string {
	blueprint := "name: combo-postgresql-kafka-test\n" +
		"version: \"v1\"\n" +
		"description: postgresql kafka combo lifecycle integration test\n\n" +
		"runtime:\n" +
		"  project-name: combo-postgresql-kafka-test\n\n" +
		"services:\n" +
		"  - name: postgres-1\n" +
		"    middleware: postgresql\n" +
		"    template: single\n" +
		"    values:\n" +
		"      version: v16\n" +
		"      port: " + strconv.Itoa(postgresPort) + "\n" +
		"      user: postgres\n" +
		"      password: postgres123\n" +
		"      database: app\n" +
		"      data_dir: ./data/postgres-1\n"
	if postgresImage != "" {
		blueprint += "      image: " + postgresImage + "\n"
	}
	blueprint +=
		"  - name: kafka-1\n" +
			"    middleware: kafka\n" +
			"    template: single\n" +
			"    values:\n" +
			"      version: v4.2\n" +
			"      port: " + strconv.Itoa(kafkaPort) + "\n" +
			"      cluster_id: MkU3OEVBNTcwNTJENDM2Qk\n" +
			"      data_dir: ./data/kafka-1\n"
	if kafkaImage != "" {
		blueprint += "      image: " + kafkaImage + "\n"
	}
	return blueprint
}

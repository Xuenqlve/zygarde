package register

import "github.com/xuenqlve/zygarde/pkg/catalog"

func init() {
	for _, info := range []catalog.TemplateInfo{
		{Middleware: "mysql", Template: "single", RuntimeType: "compose", Description: "single-node MySQL", Default: true, Versions: []string{"v5.7", "v8.0"}, DocPath: "docs/mysql.md"},
		{Middleware: "mysql", Template: "master-slave", RuntimeType: "compose", Description: "primary and replica MySQL", Versions: []string{"v5.7", "v8.0"}, DocPath: "docs/mysql.md"},
		{Middleware: "redis", Template: "single", RuntimeType: "compose", Description: "single-node Redis", Default: true, Versions: []string{"v6.2", "v7.4"}, DocPath: "docs/redis.md"},
		{Middleware: "redis", Template: "master-slave", RuntimeType: "compose", Description: "master and replica Redis", Versions: []string{"v6.2", "v7.4"}, DocPath: "docs/redis.md"},
		{Middleware: "redis", Template: "cluster", RuntimeType: "compose", Description: "three-node Redis cluster", Versions: []string{"v6.2", "v7.4"}, DocPath: "docs/redis.md"},
		{Middleware: "mongodb", Template: "single", RuntimeType: "compose", Description: "single-node MongoDB", Default: true, Versions: []string{"v6.0", "v7.0"}, DocPath: "docs/mongodb.md"},
		{Middleware: "mongodb", Template: "replica-set", RuntimeType: "compose", Description: "three-node MongoDB replica set", Versions: []string{"v6.0", "v7.0"}, DocPath: "docs/mongodb.md"},
		{Middleware: "mongodb", Template: "sharded", RuntimeType: "compose", Description: "MongoDB sharded cluster with mongos", Versions: []string{"v6.0", "v7.0"}, DocPath: "docs/mongodb.md"},
		{Middleware: "postgresql", Template: "single", RuntimeType: "compose", Description: "single-node PostgreSQL", Default: true, Versions: []string{"v16", "v17"}, DocPath: "docs/postgresql.md"},
		{Middleware: "postgresql", Template: "master-slave", RuntimeType: "compose", Description: "primary and replica PostgreSQL", Versions: []string{"v16", "v17"}, DocPath: "docs/postgresql.md"},
		{Middleware: "rabbitmq", Template: "single", RuntimeType: "compose", Description: "single-node RabbitMQ", Default: true, Versions: []string{"v4.2"}, DocPath: "docs/rabbitmq.md"},
		{Middleware: "rabbitmq", Template: "cluster", RuntimeType: "compose", Description: "three-node RabbitMQ cluster", Versions: []string{"v4.2"}, DocPath: "docs/rabbitmq.md"},
		{Middleware: "kafka", Template: "single", RuntimeType: "compose", Description: "single-node Kafka KRaft", Default: true, Versions: []string{"v4.2"}, DocPath: "docs/kafka.md"},
		{Middleware: "kafka", Template: "cluster", RuntimeType: "compose", Description: "three-node Kafka KRaft cluster", Versions: []string{"v4.2"}, DocPath: "docs/kafka.md"},
		{Middleware: "tidb", Template: "single", RuntimeType: "compose", Description: "single TiDB topology with PD and TiKV", Default: true, Versions: []string{"v6.7"}, DocPath: "docs/tidb.md"},
		{Middleware: "tidb", Template: "cluster", RuntimeType: "compose", Description: "multi-node TiDB cluster", Versions: []string{"v6.7"}, DocPath: "docs/tidb.md"},
		{Middleware: "etcd", Template: "single", RuntimeType: "compose", Description: "single-node etcd", Default: true, Versions: []string{"v3.6"}, DocPath: "docs/etcd.md"},
		{Middleware: "etcd", Template: "cluster", RuntimeType: "compose", Description: "three-node etcd cluster", Versions: []string{"v3.6"}, DocPath: "docs/etcd.md"},
		{Middleware: "consul", Template: "single", RuntimeType: "compose", Description: "single-node Consul server", Default: true, Versions: []string{"v1.20"}, DocPath: "docs/consul.md"},
		{Middleware: "consul", Template: "cluster", RuntimeType: "compose", Description: "three-node Consul server cluster", Versions: []string{"v1.20"}, DocPath: "docs/consul.md"},
		{Middleware: "clickhouse", Template: "single", RuntimeType: "compose", Description: "single-node ClickHouse", Default: true, Versions: []string{"v24", "v25"}, DocPath: "docs/clickhouse.md"},
		{Middleware: "clickhouse", Template: "cluster", RuntimeType: "compose", Description: "three-node ClickHouse cluster", Versions: []string{"v24", "v25"}, DocPath: "docs/clickhouse.md"},
		{Middleware: "zookeeper", Template: "single", RuntimeType: "compose", Description: "single-node ZooKeeper", Default: true, Versions: []string{"v3.8", "v3.9"}, DocPath: "docs/zookeeper.md"},
		{Middleware: "zookeeper", Template: "cluster", RuntimeType: "compose", Description: "three-node ZooKeeper cluster", Versions: []string{"v3.8", "v3.9"}, DocPath: "docs/zookeeper.md"},
		{Middleware: "elasticsearch", Template: "single", RuntimeType: "compose", Description: "single-node Elasticsearch", Default: true, Versions: []string{"v8.18", "v8.19"}, DocPath: "docs/elasticsearch.md"},
		{Middleware: "elasticsearch", Template: "cluster", RuntimeType: "compose", Description: "three-node Elasticsearch cluster", Versions: []string{"v8.18", "v8.19"}, DocPath: "docs/elasticsearch.md"},
	} {
		if err := catalog.RegisterTemplate(info); err != nil {
			panic(err)
		}
	}
}

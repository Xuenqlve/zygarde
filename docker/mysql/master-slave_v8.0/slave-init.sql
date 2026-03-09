-- 在 mysql-slave 执行，复制通道建立后设置 read-only
STOP REPLICA;
RESET REPLICA ALL;

CHANGE REPLICATION SOURCE TO
  SOURCE_HOST='mysql-master',
  SOURCE_PORT=3306,
  SOURCE_USER='repl',
  SOURCE_PASSWORD='repl123',
  SOURCE_AUTO_POSITION=1,
  GET_SOURCE_PUBLIC_KEY=1;

START REPLICA;

-- 复制通道建立后切换为只读
SET GLOBAL read_only = ON;
SET GLOBAL super_read_only = ON;

-- 在 mysql-slave 执行，复制通道建立后设置 read-only
STOP SLAVE;
RESET SLAVE ALL;

CHANGE MASTER TO
  MASTER_HOST='mysql-master',
  MASTER_PORT=3306,
  MASTER_USER='repl',
  MASTER_PASSWORD='repl123',
  MASTER_AUTO_POSITION=1;

START SLAVE;

-- 复制通道建立后切换为只读
SET GLOBAL read_only = ON;
SET GLOBAL super_read_only = ON;

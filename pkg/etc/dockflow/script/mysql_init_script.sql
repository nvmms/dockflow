-- 1. 禁止 root TCP 登录
ALTER USER 'root'@'%' ACCOUNT LOCK;

-- 2. root 改用 socket 认证（密码失效）
ALTER USER 'root'@'localhost'
IDENTIFIED WITH auth_socket;

-- 3. 创建 DockFlow 管理用户（非 root）
CREATE USER 'dockflow'@'localhost'
IDENTIFIED WITH auth_socket;

GRANT ALL PRIVILEGES ON *.* TO 'dockflow'@'localhost';
FLUSH PRIVILEGES;

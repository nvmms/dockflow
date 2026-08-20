INSTALL PLUGIN auth_socket SONAME 'auth_socket.so';

ALTER USER 'root'@'localhost'
IDENTIFIED WITH auth_socket AS 'mysql';

ALTER USER 'root'@'%' ACCOUNT LOCK;

CREATE USER 'dockflow'@'localhost'
IDENTIFIED WITH auth_socket AS 'mysql';

GRANT ALL PRIVILEGES ON *.* TO 'dockflow'@'localhost';
FLUSH PRIVILEGES;

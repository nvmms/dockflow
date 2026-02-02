-- =========================================================
-- DockFlow PostgreSQL Bootstrap Script
-- =========================================================

-- 1. 创建 dockflow 用户（无密码，peer 认证）
DO
$$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_roles WHERE rolname = 'dockflow'
    ) THEN
        CREATE ROLE dockflow
            LOGIN
            SUPERUSER
            CREATEDB
            CREATEROLE;
    END IF;
END
$$;

-- 2. 授权数据库所有权（默认 POSTGRES_DB）
ALTER DATABASE :"POSTGRES_DB" OWNER TO dockflow;

-- 3. 切换 schema 所有权
ALTER SCHEMA public OWNER TO dockflow;

-- 4. 允许 dockflow 使用 public schema
GRANT ALL ON SCHEMA public TO dockflow;

-- 5. 设置默认权限（未来对象自动授权）
ALTER DEFAULT PRIVILEGES IN SCHEMA public
GRANT ALL ON TABLES TO dockflow;

ALTER DEFAULT PRIVILEGES IN SCHEMA public
GRANT ALL ON SEQUENCES TO dockflow;

ALTER DEFAULT PRIVILEGES IN SCHEMA public
GRANT ALL ON FUNCTIONS TO dockflow;

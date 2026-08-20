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

SELECT format('ALTER DATABASE %I OWNER TO dockflow', current_database()) \gexec
ALTER SCHEMA public OWNER TO dockflow;
GRANT ALL ON SCHEMA public TO dockflow;
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL ON TABLES TO dockflow;
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL ON SEQUENCES TO dockflow;
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL ON FUNCTIONS TO dockflow;

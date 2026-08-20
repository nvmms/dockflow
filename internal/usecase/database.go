package usecase

import (
	"dockflow/internal/domain"
	"dockflow/internal/service/docker"
	"dockflow/internal/service/filesystem"
	"dockflow/internal/util"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/samber/lo"
)

var (
	ErrdatabaseNotFound   = errors.New("database not found")
	ErrdatabaseNotExist   = errors.New("database not exist")
	ErrdatabaseExist      = errors.New("database name is exist")
	ErrdatabaseNotSuppert = errors.New("database not suppert")
)

func Createdatabase(database domain.DatabaseSpec) error {
	ns, err := domain.NewNamespace(database.Namespace)
	if err != nil {
		return err
	}
	if ns == nil {
		return ErrNamespaceNotFound
	}

	_, found := lo.Find(ns.Database, func(d domain.DatabaseSpec) bool {
		return d.Name == database.Name
	})

	if found {
		return ErrdatabaseExist
	}

	name := strings.ToLower(database.DbType)

	if idx := strings.LastIndex(name, "/"); idx != -1 {
		name = name[idx+1:]
	}
	if idx := strings.Index(name, ":"); idx != -1 {
		name = name[:idx]
	}
	switch name {
	case "mysql", "postgres", "postgresql":
	default:
		return ErrdatabaseNotSuppert
	}

	databaseImageName := database.DbType
	if err := docker.PullImage(databaseImageName); err != nil {
		return err
	}

	opts := docker.NewRunOptions(database.Name, databaseImageName)
	opts.WithRestart(container.RestartPolicyOnFailure)
	opts.WithNetwork(ns.Network)
	opts.WithCpu(database.CPU)
	opts.WithMemory(database.Memory)

	err = detectDatabaseType(database, opts)
	if err != nil {
		return err
	}

	containerId, err := docker.RunContainer(opts)
	if err != nil {
		return err
	}

	inspect, err := docker.InspectContainer(containerId)
	if err != nil {
		return err
	}

	ips := []string{}
	for _, net := range inspect.NetworkSettings.Networks {
		ips = append(ips, net.IPAddress)
	}

	database.ContainerId = containerId
	database.Ip = ips
	ns.Database = append(ns.Database, database)
	ns.Save()

	return nil
}

func Listdatabase(namespaceName string) ([]domain.DatabaseSpec, error) {
	ns, err := domain.NewNamespace(namespaceName)
	if err != nil {
		return nil, err
	}
	if ns == nil {
		return nil, ErrNamespaceNotFound
	}

	return ns.Database, nil
}

func Removedatabase(namespaceName string, databaseContainerName string) error {
	ns, err := domain.NewNamespace(namespaceName)
	if err != nil {
		return err
	}
	if ns == nil {
		return ErrNamespaceNotFound
	}

	database, index, found := lo.FindIndexOf(ns.Database, func(d domain.DatabaseSpec) bool {
		return d.Name == databaseContainerName
	})
	if !found {
		return ErrdatabaseNotExist
	}

	containerId, err := docker.HasContainer(database.ContainerId)
	if err != nil {
		return err
	}

	if containerId != "" {
		containerInfo, err := docker.InspectContainer(database.ContainerId)
		if err != nil {
			return err
		}

		isRun, err := docker.ContainerRunning(database.ContainerId)
		if err != nil {
			return err
		}

		if isRun {
			err := docker.StopContainer(database.ContainerId, nil)
			if err != nil {
				return err
			}
		}

		err = docker.RemoveContainer(database.ContainerId, true)
		if err != nil {
			return err
		}

		for _, m := range containerInfo.Mounts {
			if m.Type == mount.TypeVolume {
				err := docker.RemoveVolume(m.Name)
				if err != nil {
					return err
				}
			}
		}
	}

	ns.Database = lo.Filter(ns.Database, func(item domain.DatabaseSpec, i int) bool {
		return index != i
	})
	ns.Save()

	return nil
}

func detectDatabaseType(database domain.DatabaseSpec, opt *docker.ContainerRunOptions) (err error) {
	name := strings.ToLower(database.DbType)

	if idx := strings.LastIndex(name, "/"); idx != -1 {
		name = name[idx+1:]
	}
	if idx := strings.Index(name, ":"); idx != -1 {
		name = name[:idx]
	}

	switch name {
	case "mysql":
		opt.WithVolume(filesystem.MySqlInitScript, "/docker-entrypoint-initdb.d/001-dockflow.sql", "ro")
		opt.WithVolume(fmt.Sprintf("dockflow-dbvolume-%s-%s-%s", database.Namespace, database.Name, database.DbName), "/var/lib/mysql")

		opt.WithEnv("MYSQL_ALLOW_EMPTY_PASSWORD", "yes")
		opt.WithEnv("MYSQL_DATABASE", database.DbName)
		opt.WithEnv("MYSQL_USER", database.Username)
		opt.WithEnv("MYSQL_PASSWORD", database.Password)

		if database.Remote {
			hostPort := util.GenerateRandomPort()
			opt.WithPort(hostPort, 3306)
		}

	case "postgres", "postgresql":
		opt.WithVolume(filesystem.PgSqlInitScript, "/docker-entrypoint-initdb.d/001-dockflow.sql", "ro")
		opt.WithVolume(fmt.Sprintf("dockflow-dbvolume-%s-%s-%s", database.Namespace, database.Name, database.DbName), "/var/lib/postgresql")

		opt.WithEnv("POSTGRES_DB", database.DbName)
		opt.WithEnv("POSTGRES_USER", database.Username)
		opt.WithEnv("POSTGRES_PASSWORD", database.Password)

		if database.Remote {
			hostPort := util.GenerateRandomPort()
			opt.WithPort(hostPort, 3306)
		}
	default:
		return ErrdatabaseNotSuppert
	}
	return nil
}

func ExportSQL(namespace, name string) error {
	out, err := ExportSQLData(namespace, name)
	if err != nil {
		return err
	}
	return os.WriteFile("/tmp/asdfasdf.sql", out, 0644)
}

// ExportSQLData returns a dump so non-CLI transports can stream it directly.
func ExportSQLData(namespace, name string) ([]byte, error) {
	ns, err := domain.NewNamespace(namespace)
	if err != nil {
		return nil, err
	}

	database, found := lo.Find(ns.Database, func(d domain.DatabaseSpec) bool {
		return d.Name == name
	})
	if !found {
		return nil, fmt.Errorf("database [%s] not exist", name)
	}

	command, env, err := databaseExportCommand(database)
	if err != nil {
		return nil, err
	}
	out, err := docker.ExecContainer(database.ContainerId, command, docker.ContainerExecOptions{Env: env, User: databaseExecUser(database)})
	if err != nil {
		return nil, err
	}
	return []byte(out), nil
}

func ImportSQL(namespace, name, sqlPath string) error {
	file, err := os.Open(sqlPath)
	if err != nil {
		return err
	}
	defer file.Close()
	return ImportSQLReader(namespace, name, file)
}

// ImportSQLReader imports SQL from a stream, avoiding transport-specific temp files.
func ImportSQLReader(namespace, name string, input io.Reader) error {
	ns, err := domain.NewNamespace(namespace)
	if err != nil {
		return err
	}

	database, found := lo.Find(ns.Database, func(d domain.DatabaseSpec) bool {
		return d.Name == name
	})
	if !found {
		return fmt.Errorf("database [%s] not exist", name)
	}

	command, env, err := databaseImportCommand(database)
	if err != nil {
		return err
	}
	_, err = docker.ExecContainer(database.ContainerId, command, docker.ContainerExecOptions{Stdin: input, Env: env, User: databaseExecUser(database)})

	return err
}

func databaseExportCommand(database domain.DatabaseSpec) ([]string, []string, error) {
	switch databaseEngine(database.DbType) {
	case "mysql":
		return []string{
			"mysqldump",
			"--user", "dockflow",
			"--single-transaction",
			"--routines",
			"--triggers",
			database.DbName,
		}, nil, nil
	case "postgres":
		return []string{
			"pg_dump",
			"--username", "dockflow",
			"--dbname", database.DbName,
			"--no-owner",
			"--no-privileges",
		}, nil, nil
	default:
		return nil, nil, ErrdatabaseNotSuppert
	}
}

func databaseImportCommand(database domain.DatabaseSpec) ([]string, []string, error) {
	switch databaseEngine(database.DbType) {
	case "mysql":
		return []string{
			"mysql",
			"--user", "dockflow",
			database.DbName,
		}, nil, nil
	case "postgres":
		return []string{
			"psql",
			"--username", "dockflow",
			"--dbname", database.DbName,
			"--set", "ON_ERROR_STOP=1",
		}, nil, nil
	default:
		return nil, nil, ErrdatabaseNotSuppert
	}
}

func databaseEngine(dbType string) string {
	name := strings.ToLower(dbType)
	if idx := strings.LastIndex(name, "/"); idx != -1 {
		name = name[idx+1:]
	}
	if idx := strings.Index(name, ":"); idx != -1 {
		name = name[:idx]
	}
	if name == "postgresql" {
		return "postgres"
	}
	return name
}

func databaseExecUser(database domain.DatabaseSpec) string {
	if databaseEngine(database.DbType) == "postgres" {
		return "postgres"
	}
	return "mysql"
}

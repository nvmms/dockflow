package usecase

import (
	"dockflow/internal/domain"
	"dockflow/internal/service/docker"
	"dockflow/internal/service/filesystem"
	"errors"
	"strings"

	"github.com/docker/docker/api/types/container"
	"github.com/samber/lo"
)

var (
	ErrdatabaseNotFound = errors.New("database not found")
	ErrdatabaseNotExist = errors.New("database not exist")
	ErrdatabaseExist    = errors.New("database name is exist")
	ErrdatabaseNotSuppert = errors.New("database not suppert")
)

func Createdatabase(database domain.DatabaseSpec) error {

	ns, err := filesystem.LoadNamespace(database.Namespace)
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
	case "mysql","postgres", "postgresql":
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

	env ,err:= detectDatabaseType(database)
	if err != nil{
		return err
	}
	for key, value := range env {
		opts.WithEnv(key, value)
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
	filesystem.SaveNamespace(*ns)

	return nil
}

func Listdatabase(namespaceName string) ([]domain.DatabaseSpec, error) {
	ns, err := filesystem.LoadNamespace(namespaceName)
	if err != nil {
		return nil, err
	}
	if ns == nil {
		return nil, ErrNamespaceNotFound
	}

	return ns.Database, nil
}

func Removedatabase(namespaceName string, databaseContainerName string) error {
	ns, err := filesystem.LoadNamespace(namespaceName)
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

	isExist, err := docker.HasContainer(database.ContainerId)
	if err != nil {
		return err
	}
	if !isExist {
		return ErrdatabaseNotExist
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

	ns.Database = lo.Filter(ns.Database, func(item domain.DatabaseSpec, i int) bool {
		return index != i
	})
	filesystem.SaveNamespace(*ns)

	return nil

}

func detectDatabaseType(database domain.DatabaseSpec) (env map[string]string,err error) {
	name := strings.ToLower(database.DbType)

	if idx := strings.LastIndex(name, "/"); idx != -1 {
		name = name[idx+1:]
	}
	if idx := strings.Index(name, ":"); idx != -1 {
		name = name[:idx]
	}

	switch name {
	case "mysql":
		return createMysqlEnv(database),nil
	case "postgres", "postgresql":
		return createPostgresEnv(database),nil
	default:
		return nil ,ErrdatabaseNotSuppert
	}
}

func createMysqlEnv(database domain.DatabaseSpec) (env map[string]string) {
	env = make(map[string]string)
	env["MYSQL_DATABASE"] = database.DbName
	env["MYSQL_USER"] = database.Username
	env["MYSQL_PASSWORD"] = database.Password
	return env
}

func createPostgresEnv(database domain.DatabaseSpec) (env map[string]string) {
	env = make(map[string]string)
	env["POSTGRES_DB"] = database.DbName
	env["POSTGRES_USER"] = database.Username
	env["POSTGRES_PASSWORD"] = database.Password
	return env
}

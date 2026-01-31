package filesystem

import (
	"dockflow/internal/domain"
	"fmt"
)

func SaveApp(app domain.AppSpec) error {
	ns, err := LoadNamespace(app.Namespace)
	if err != nil {
		return err
	}
	if ns == nil {
		return fmt.Errorf("namespace [%s] not exist", app.Namespace)
	}
	var appIndex = -1
	for index, _app := range ns.App {
		if app.Name == _app.Name {
			appIndex = index
		}
	}
	if appIndex == -1 {
		return fmt.Errorf("app [%s] not exist", app.Name)
	}
	ns.App[appIndex] = app
	SaveNamespace(*ns)
	return nil
}

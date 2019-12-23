package helm

import (
	"fmt"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/helmpath"
	"helm.sh/helm/v3/pkg/repo"
	"log"
	"os"
)

var (
	settings = cli.New()
)

// GetChartsForRepo retrieve charts info from a repo cache index
// Check: can we use the generated time to do compare?
func GetChartsForRepo(name string) (*repo.IndexFile, error) {
	path := helmpath.CachePath("repository") + fmt.Sprintf("/%s-index.yaml", name)
	return repo.LoadIndexFile(path)
}

// GetActionConfig returns action configuration based on Helm env
func GetActionConfig(namespace string) (*action.Configuration, error) {
	actionConfig := new(action.Configuration)
	err := actionConfig.Init(settings.RESTClientGetter(), namespace, os.Getenv("HELM_DRIVER"), debug)
	if err != nil {
		return nil, err
	}

	return actionConfig, err
}

func debug(format string, v ...interface{}) {
	if settings.Debug {
		format = fmt.Sprintf("[debug] %s\n", format)
		log.Output(2, fmt.Sprintf(format, v...))
	}
}

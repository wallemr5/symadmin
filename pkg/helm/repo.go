package helm

import (
	"github.com/pkg/errors"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/getter"
	"helm.sh/helm/v3/pkg/helmpath"
	"helm.sh/helm/v3/pkg/repo"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/klog"
	"sync"
	"time"
)

var (
	defaultInterval = 30
)

var lock sync.Mutex

// ChartDetails describes a chart details
type ChartDetails struct {
	Name     string          `json:"name"`
	Repo     string          `json:"repo"`
	Versions []*ChartVersion `json:"versions"`
}

// ChartVersion describes a chart verion
type ChartVersion struct {
	Chart  *repo.ChartVersion `json:"chart"`
	Values string             `json:"values"`
	Readme string             `json:"readme"`
}

// IndexSyncer sync helm repo index repeatedly
type IndexSyncer struct {
	// interval is the interval of the sync process
	interval int
}

// NewDefaultIndexSyncer create a new IndexSyncer
func NewDefaultIndexSyncer(repos map[string]string) *IndexSyncer {
	err := ensureDefaultRepos(repos)
	if err != nil {
		klog.Fatal("repo err:%v", err)
	}
	return &IndexSyncer{
		interval: defaultInterval,
	}
}

// Start will refresh the repo index periodically
func (i *IndexSyncer) Start(stop <-chan struct{}) error {
	return wait.PollUntil(time.Second*time.Duration(i.interval),
		func() (done bool, err error) {
			klog.V(4).Info("update helm repo index")
			err = initReposIndex()
			if err != nil {
				klog.Error("update helm repo index error: ", err)
			}
			return false, nil
		},
		stop,
	)
}

// initReposIndex update index for all the known repos. This happens when captain starts.
func initReposIndex() error {
	f, err := repo.LoadFile(helmRepositoryFile())
	if err != nil {
		return err
	}
	if len(f.Repositories) == 0 {
		return nil
	}
	for _, re := range f.Repositories {
		err := addRepository(re.Name, re.URL, re.Username, re.Password, "", "", "", false)
		if err != nil {
			klog.Warningf("repo index update error for %s: %s", re.Name, err.Error())
			continue
		} else {
			klog.Infof("update index done for repo: %s", re.Name)
		}
	}
	return nil
}

// init default config repo
func ensureDefaultRepos(repos map[string]string) error {
	klog.Infof("Setting up default helm repos.")
	for repoName, repoUrl := range repos {
		if err := AddBasicAuthRepository(repoName, repoUrl, "", ""); err != nil {
			return errors.Wrapf(err, "cannot init repo: %s", repoName)
		}
	}

	return nil
}

func helmRepositoryFile() string {
	return helmpath.ConfigPath("repositories.yaml")
}

// AddBasicAuthRepository add a repo with basic auth
func AddBasicAuthRepository(name, url, username, password string) error {
	return addRepository(name, url, username, password, "", "", "", false)
}

// RemoveRepository remove a repo from helm
func RemoveRepository(name string) error {
	lock.Lock()
	defer lock.Unlock()

	f, err := repo.LoadFile(helmRepositoryFile())
	if err != nil {
		return err
	}

	found := f.Remove(name)
	if found {
		return f.WriteFile(helmRepositoryFile(), 0644)
	}

	return nil
}

// addRepository add a repo and update index ( the repo already exist, we only need to update-index part)
func addRepository(name, url, username, password string, certFile, keyFile, caFile string, noUpdate bool) error {
	lock.Lock()
	defer lock.Unlock()

	f, err := repo.LoadFile(helmRepositoryFile())
	if err != nil {
		return err
	}

	if noUpdate && f.Has(name) {
		return errors.Errorf("repository name (%s) already exists, please specify a different name", name)
	}

	c := repo.Entry{
		Name:     name,
		URL:      url,
		Username: username,
		Password: password,
		CertFile: certFile,
		KeyFile:  keyFile,
		CAFile:   caFile,
	}

	settings := cli.New()
	settings.Debug = true

	r, err := repo.NewChartRepository(&c, getter.All(settings))
	if err != nil {
		return err
	}

	if _, err := r.DownloadIndexFile(); err != nil {
		return errors.Wrapf(err, "looks like %q is not a valid chart repository or cannot be reached", url)
	}

	f.Update(&c)

	return f.WriteFile(helmpath.ConfigPath("repositories.yaml"), 0644)
}

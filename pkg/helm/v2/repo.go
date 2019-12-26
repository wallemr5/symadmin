package v2

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"k8s.io/helm/pkg/downloader"
	"k8s.io/helm/pkg/getter"
	helmenv "k8s.io/helm/pkg/helm/environment"
	"k8s.io/helm/pkg/helm/helmpath"
	"k8s.io/helm/pkg/repo"
	"k8s.io/klog"
)

// ErrRepoNotFound describe an error if helm repository not found
// nolint: gochecknoglobals
var ErrRepoNotFound = errors.New("helm repository not found!")

// InitHelmRepoEnv Generate helm path based on orgName
func InitHelmRepoEnv(organizationName string, repoMap map[string]string) (*helmenv.EnvSettings, error) {
	settings := &helmenv.EnvSettings{}

	helmRepoHome := fmt.Sprintf("%s/%s", "./cache", organizationName)
	settings.Home = helmpath.Home(helmRepoHome)

	// check local helm
	_, err := os.Stat(helmRepoHome)
	if os.IsNotExist(err) {
		klog.Infof("Helm directories [%s] not exists", helmRepoHome)
		err := InstallLocalHelm(settings, repoMap)
		if err != nil {
			klog.Errorf("InstallLocalHelm err: %+v", err)
			return nil, err
		}
	} else {
		entries, err := ReposGet(settings)
		if err != nil {
			klog.Errorf("get all repo err: %+v", err)
			return nil, err
		}

		for _, e := range entries {
			err := ReposUpdate(settings, e.Name)
			if err != nil {
				klog.Errorf("update repo: %s err: %+v", e.Name, err)
			}

			// lists, err := ChartsGet(env, "", e.Name, "", "")
			// if err == nil {
			// 	klog.Infof("charts len:%d", len(lists[0].Charts))
			// }
		}
	}

	return settings, nil
}

// InstallLocalHelm install helm into the given path
func InstallLocalHelm(env *helmenv.EnvSettings, repoMap map[string]string) error {
	if err := InstallHelmClient(env); err != nil {
		return err
	}

	klog.Info("Helm client install succeeded")
	if err := ensureDefaultRepos(env, repoMap); err != nil {
		return errors.Wrap(err, "Setting up default repos failed!")
	}
	return nil
}

// DownloadChartFromRepo download a given chart
func DownloadChartFromRepo(name, version string, env helmenv.EnvSettings) (string, error) {
	dl := downloader.ChartDownloader{
		HelmHome: env.Home,
		Getters:  getter.All(env),
	}
	if _, err := os.Stat(env.Home.Archive()); os.IsNotExist(err) {
		klog.Infof("Creating '%s' directory.", env.Home.Archive())
		_ = os.MkdirAll(env.Home.Archive(), 0744)
	}

	klog.Infof("Downloading helm chart %q, version %q to %q", name, version, env.Home.Archive())
	filename, _, err := dl.DownloadTo(name, version, env.Home.Archive())
	if err == nil {
		lname, err := filepath.Abs(filename)
		if err != nil {
			return filename, errors.Wrapf(err, "Could not create absolute path from %s", filename)
		}
		klog.Infof("Fetched helm chart %q, version %q to %q", name, version, filename)
		return lname, nil
	}

	return filename, errors.Wrapf(err, "Failed to download chart %q, version %q", name, version)
}

// InstallHelmClient Installs helm client on a given path
func InstallHelmClient(env *helmenv.EnvSettings) error {
	if err := EnsureDirectories(env); err != nil {
		return errors.Wrap(err, "Initializing helm directories failed!")
	}

	klog.Info("Initializing helm client succeeded, happy helming!")
	return nil
}

// EnsureDirectories for helm repo local install
func EnsureDirectories(env *helmenv.EnvSettings) error {
	home := env.Home
	configDirectories := []string{
		home.String(),
		home.Repository(),
		home.Cache(),
		home.LocalRepository(),
		home.Plugins(),
		home.Starters(),
		home.Archive(),
	}

	klog.Info("Setting up helm directories.")

	for _, p := range configDirectories {
		if fi, err := os.Stat(p); err != nil {
			klog.Infof("Creating '%s'", p)
			if err := os.MkdirAll(p, 0755); err != nil {
				return errors.Wrapf(err, "Could not create '%s'", p)
			}
		} else if !fi.IsDir() {
			return errors.Errorf("'%s' must be a directory", p)
		}
	}
	return nil
}

func ensureDefaultRepos(env *helmenv.EnvSettings, repoMap map[string]string) error {
	klog.Infof("Setting up default helm repos.")

	for repoName, repoUrl := range repoMap {
		_, err := ReposAdd(
			env,
			&repo.Entry{
				Name:  repoName,
				URL:   repoUrl,
				Cache: env.Home.CacheIndex(repoName),
			})
		if err != nil {
			return errors.Wrapf(err, "cannot init repo: %s", repoName)
		}
	}

	return nil
}

// ReposGet returns repo
func ReposGet(env *helmenv.EnvSettings) ([]*repo.Entry, error) {
	repoPath := env.Home.RepositoryFile()
	klog.V(3).Infof("Helm repo path: %s", repoPath)

	f, err := repo.LoadRepositoriesFile(repoPath)
	if err != nil {
		return nil, err
	}
	if len(f.Repositories) == 0 {
		return make([]*repo.Entry, 0), nil
	}

	return f.Repositories, nil
}

// ReposAdd adds repo(s)
func ReposAdd(env *helmenv.EnvSettings, Hrepo *repo.Entry) (bool, error) {
	repoFile := env.Home.RepositoryFile()
	var f *repo.RepoFile
	if _, err := os.Stat(repoFile); err != nil {
		klog.Infof("Creating %s", repoFile)
		f = repo.NewRepoFile()
	} else {
		f, err = repo.LoadRepositoriesFile(repoFile)
		if err != nil {
			return false, errors.Wrap(err, "Cannot create a new ChartRepo")
		}
		klog.Infof("Profile file %q loaded.", repoFile)
	}

	for _, n := range f.Repositories {
		klog.Infof("repo: %s", n.Name)
		if n.Name == Hrepo.Name {
			return false, nil
		}
	}

	c := repo.Entry{
		Name:  Hrepo.Name,
		URL:   Hrepo.URL,
		Cache: env.Home.CacheIndex(Hrepo.Name),
	}
	r, err := repo.NewChartRepository(&c, getter.All(*env))
	if err != nil {
		return false, errors.Wrap(err, "Cannot create a new ChartRepo")
	}
	klog.Infof("New repo added: %s", Hrepo.Name)

	errIdx := r.DownloadIndexFile("")
	if errIdx != nil {
		return false, errors.Wrap(errIdx, "Repo index download failed")
	}
	f.Add(&c)
	if errW := f.WriteFile(repoFile, 0644); errW != nil {
		return false, errors.Wrap(errW, "Cannot write helm repo profile file")
	}
	return true, nil
}

// ReposDelete deletes repo(s)
func ReposDelete(env helmenv.EnvSettings, repoName string) error {
	repoFile := env.Home.RepositoryFile()
	klog.Infof("Repo File: %s", repoFile)

	r, err := repo.LoadRepositoriesFile(repoFile)
	if err != nil {
		return err
	}

	if !r.Remove(repoName) {
		return ErrRepoNotFound
	}
	if err := r.WriteFile(repoFile, 0644); err != nil {
		return err
	}

	if _, err := os.Stat(env.Home.CacheIndex(repoName)); err == nil {
		err = os.Remove(env.Home.CacheIndex(repoName))
		if err != nil {
			return err
		}
	}
	return nil
}

// ReposModify modifies repo(s)
func ReposModify(env helmenv.EnvSettings, repoName string, newRepo *repo.Entry) error {

	klog.Info("ReposModify")
	repoFile := env.Home.RepositoryFile()
	klog.Infof("Repo File: %s", repoFile)
	klog.Infof("New repo content: %#v", newRepo)

	f, err := repo.LoadRepositoriesFile(repoFile)
	if err != nil {
		return err
	}

	if !f.Has(repoName) {
		return ErrRepoNotFound
	}

	var formerRepo *repo.Entry
	repos := f.Repositories
	for _, r := range repos {
		if r.Name == repoName {
			formerRepo = r
		}
	}

	if formerRepo != nil {
		if len(newRepo.Name) == 0 {
			newRepo.Name = formerRepo.Name
			klog.Infof("new repo name field is empty, replaced with: %s", formerRepo.Name)
		}

		if len(newRepo.URL) == 0 {
			newRepo.URL = formerRepo.URL
			klog.Infof("new repo url field is empty, replaced with: %s", formerRepo.URL)
		}

		if len(newRepo.Cache) == 0 {
			newRepo.Cache = formerRepo.Cache
			klog.Infof("new repo cache field is empty, replaced with: %s", formerRepo.Cache)
		}
	}

	f.Update(newRepo)

	if errW := f.WriteFile(repoFile, 0644); errW != nil {
		return errors.Wrap(errW, "Cannot write helm repo profile file")
	}
	return nil
}

// ReposUpdate updates a repo(s)
func ReposUpdate(env *helmenv.EnvSettings, repoName string) error {
	repoFile := env.Home.RepositoryFile()
	klog.Infof("start update Repo:%s File: %s", repoName, repoFile)

	f, err := repo.LoadRepositoriesFile(repoFile)
	if err != nil {
		return errors.Wrap(err, "Load ChartRepo")
	}

	for _, cfg := range f.Repositories {
		if cfg.Name == repoName {
			c, err := repo.NewChartRepository(cfg, getter.All(*env))
			if err != nil {
				return errors.Wrap(err, "Cannot get ChartRepo")
			}
			errIdx := c.DownloadIndexFile("")
			if errIdx != nil {
				return errors.Wrap(errIdx, "Repo index download failed")
			}
			return nil
		}
	}

	return ErrRepoNotFound
}

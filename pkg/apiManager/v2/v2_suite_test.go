package v2

import (
	"testing"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	k8sclient "gitlab.dmall.com/arch/sym-admin/pkg/k8s/client"
	k8smanager "gitlab.dmall.com/arch/sym-admin/pkg/k8s/manager"
	"gitlab.dmall.com/arch/sym-admin/pkg/labels"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/klog"
	ctrlmanager "sigs.k8s.io/controller-runtime/pkg/manager"
)

var (
	manager *Manager
)

const (
	timeout     = 5.0
	emptyResult = `{
    "message": null,
    "resultMap": {
        "data": []
    },
    "success": true
}`
)

func TestV2(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "V2 Suite")
}

var _ = BeforeSuite(func() {
	opt := DefaultRootOption()
	opt.Kubeconfig = "../../../manifests/kubeconfig.yaml"
	opt.ConfigContext = "test-tke-gz-bj5-bus-01"
	cli := NewDksCli(opt)
	cfg, err := cli.GetK8sConfig()
	Expect(err).NotTo(HaveOccurred())

	rp := time.Second * 120
	mgr, err := ctrlmanager.New(cfg, ctrlmanager.Options{
		Scheme:                 k8sclient.GetScheme(),
		MetricsBindAddress:     "0",
		HealthProbeBindAddress: "0",
		LeaderElection:         false,
		SyncPeriod:             &rp,
	})
	Expect(err).NotTo(HaveOccurred())

	k8sCli := k8smanager.MasterClient{
		KubeCli: cli.GetKubeInterfaceOrDie(),
		Manager: mgr,
	}

	k8sMgr, err := k8smanager.NewManager(
		k8sCli,
		k8smanager.DefaultClusterManagerOption(true, labels.GetClusterLs()),
	)
	Expect(err).NotTo(HaveOccurred())

	manager = &Manager{K8sMgr: k8sMgr, Cluster: k8sMgr}
}, timeout)

type RootOption struct {
	Kubeconfig       string
	ConfigContext    string
	Namespace        string
	DefaultNamespace string
	DevelopmentMode  bool
}

func DefaultRootOption() *RootOption {
	return &RootOption{
		Namespace:       corev1.NamespaceAll,
		DevelopmentMode: true,
	}
}

type DksCli struct {
	RootCmd *cobra.Command
	Opt     *RootOption
}

func NewDksCli(opt *RootOption) *DksCli {
	return &DksCli{
		Opt: opt,
	}
}

func (c *DksCli) GetK8sConfig() (*rest.Config, error) {
	config, err := k8sclient.GetConfigWithContext(c.Opt.Kubeconfig, c.Opt.ConfigContext)
	if err != nil {
		return nil, errors.Wrap(err, "could not get k8s config")
	}

	return config, nil
}

func (c *DksCli) GetKubeInterface() (kubernetes.Interface, error) {
	cfg, err := c.GetK8sConfig()
	if err != nil {
		return nil, errors.Wrap(err, "could not get k8s config")
	}

	kubeCli, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		klog.Fatalf("failed to get kubernetes Clientset: %v", err)
	}

	return kubeCli, nil
}

func (c *DksCli) GetKubeInterfaceOrDie() kubernetes.Interface {
	kubeCli, err := c.GetKubeInterface()
	if err != nil {
		klog.Fatalf("unable to get kube interface err: %v", err)
	}

	return kubeCli
}

package v2

import (
	"strings"
	"time"

	"fmt"

	"github.com/pkg/errors"
	"gitlab.dmall.com/arch/sym-admin/pkg/backoff"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	k8sapierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/helm/cmd/helm/installer"
	"k8s.io/klog"
)

const releaseNameMaxLen = 53

// Install describes an Helm tiller install request
type Install struct {
	// Namespace of Tiller
	Namespace string `json:"namespace"` // "kube-system"

	// Upgrade if Tiller is already installed
	Upgrade bool `json:"upgrade"`

	// Name of service account
	ServiceAccount string `json:"service_account"`

	ClusterRoleName string `json:"ClusterRoleName,omitempty"`

	// Use the canary Tiller image
	Canary bool `json:"canary_image"`

	// Override Tiller image
	ImageSpec string `json:"tiller_image"`

	// Limit the maximum number of revisions saved per release. Use 0 for no limit.
	MaxHistory int `json:"history_max"`

	Tolerations  []corev1.Toleration  `json:"tolerations,omitempty"`
	NodeAffinity *corev1.NodeAffinity `json:"nodeAffinity,omitempty"`
	NodeSelector string               `json:"nodeSelector,omitempty"`
}

// PreInstallTiller create's serviceAccount and AccountRoleBinding
func PreInstallTiller(helmInstall *Install, kubeClient kubernetes.Interface) error {
	klog.Infof("start pre-install")

	var backoffConfig = backoff.ConstantBackoffConfig{
		Delay:      10 * time.Second,
		MaxRetries: 5,
	}
	var backoffPolicy = backoff.NewConstantBackoffPolicy(&backoffConfig)

	v1MetaData := metav1.ObjectMeta{
		Name: helmInstall.ServiceAccount, // "tiller",
	}

	serviceAccount := &corev1.ServiceAccount{
		ObjectMeta: v1MetaData,
	}

	klog.Info("create serviceaccount")
	err := backoff.Retry(func() error {
		if _, err := kubeClient.CoreV1().ServiceAccounts(helmInstall.Namespace).Create(serviceAccount); err != nil {
			if k8sapierrors.IsAlreadyExists(err) {
				return backoff.MarkErrorPermanent(err)
			}
		}
		return nil
	}, backoffPolicy)
	if err != nil {
		return errors.Wrapf(err, "could not create serviceaccount serviceaccount: %q  namespace: %s", serviceAccount, helmInstall.Namespace)
	}

	clusterRoleName := "cluster-admin"
	if helmInstall.ClusterRoleName != "" && helmInstall.ClusterRoleName == helmInstall.ServiceAccount {
		clusterRole := &rbacv1.ClusterRole{
			ObjectMeta: v1MetaData,
			Rules: []rbacv1.PolicyRule{{
				APIGroups: []string{
					"*",
				},
				Resources: []string{
					"*",
				},
				Verbs: []string{
					"*",
				},
			},
				{
					NonResourceURLs: []string{
						"*",
					},
					Verbs: []string{
						"*",
					},
				}},
		}
		klog.Info("create clusterroles")
		clusterRoleName = helmInstall.ServiceAccount
		err = backoff.Retry(func() error {
			if _, err := kubeClient.RbacV1().ClusterRoles().Create(clusterRole); err != nil {
				if k8sapierrors.IsAlreadyExists(err) {
					return backoff.MarkErrorPermanent(err)
				}
			}
			return nil
		}, backoffPolicy)
		//if err != nil && strings.Contains(err.Error(), "is forbidden") {
		if err != nil {
			return errors.Wrapf(err, "could not create clusterrole: %s", clusterRoleName)
		}
	} else {
		_, errGet := kubeClient.RbacV1().ClusterRoles().Get(clusterRoleName, metav1.GetOptions{})
		if errGet != nil {
			return errors.Wrapf(errGet, "clusterrole:%s not found", clusterRoleName)
		}
	}

	klog.Infof("ClusterRole Name: %s serviceAccount Name: %s", clusterRoleName, helmInstall.ServiceAccount)
	clusterRoleBinding := &rbacv1.ClusterRoleBinding{
		ObjectMeta: v1MetaData,
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     clusterRoleName, // "tiller",
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      helmInstall.ServiceAccount, // "tiller",
				Namespace: helmInstall.Namespace,
			}},
	}
	klog.Info("create clusterrolebinding")
	err = backoff.Retry(func() error {
		if _, err := kubeClient.RbacV1().ClusterRoleBindings().Create(clusterRoleBinding); err != nil {
			if k8sapierrors.IsAlreadyExists(err) {
				return backoff.MarkErrorPermanent(err)
			}
		}
		return nil
	}, backoffPolicy)

	if err != nil {
		return errors.Wrapf(err, "could not create clusterrolebinding: %s", clusterRoleBinding.Name)
	}

	return nil
}

// RetryHelmInstall retries for a configurable time/interval
func RetryHelmInstall(helmInstall *Install, kubeClient kubernetes.Interface) error {
	retryAttempts := 30
	retrySleepSeconds := 15
	for i := 0; i <= retryAttempts; i++ {
		klog.Infof("Waiting %d/%d", i, retryAttempts)
		err := InstallTiller(helmInstall, kubeClient)
		if err != nil {
			if strings.Contains(err.Error(), "net/http: TLS handshake timeout") {
				time.Sleep(time.Duration(retrySleepSeconds) * time.Second)
				continue
			}
		}
		return nil
	}
	return fmt.Errorf("timeout during helm install")
}

// InstallTiller uses Kubernetes client to install Tiller.
func InstallTiller(helmInstall *Install, kubeClient kubernetes.Interface) error {
	opts := installer.Options{
		Namespace:                    helmInstall.Namespace,
		ServiceAccount:               helmInstall.ServiceAccount,
		UseCanary:                    helmInstall.Canary,
		ImageSpec:                    helmInstall.ImageSpec,
		MaxHistory:                   helmInstall.MaxHistory,
		AutoMountServiceAccountToken: true,
	}

	for i := range helmInstall.Tolerations {
		if helmInstall.Tolerations[i].Key != "" {
			opts.Values = append(opts.Values, fmt.Sprintf("spec.template.spec.tolerations[%d].key=%s", i, helmInstall.Tolerations[i].Key))
		}

		if helmInstall.Tolerations[i].Operator != "" {
			opts.Values = append(opts.Values, fmt.Sprintf("spec.template.spec.tolerations[%d].operator=%s", i, helmInstall.Tolerations[i].Operator))
		}

		if helmInstall.Tolerations[i].Value != "" {
			opts.Values = append(opts.Values, fmt.Sprintf("spec.template.spec.tolerations[%d].value=%s", i, helmInstall.Tolerations[i].Value))
		}

		if helmInstall.Tolerations[i].Effect != "" {
			opts.Values = append(opts.Values, fmt.Sprintf("spec.template.spec.tolerations[%d].effect=%s", i, helmInstall.Tolerations[i].Effect))
		}
	}

	if helmInstall.NodeAffinity != nil {
		for i := range helmInstall.NodeAffinity.PreferredDuringSchedulingIgnoredDuringExecution {
			preferredSchedulingTerm := helmInstall.NodeAffinity.PreferredDuringSchedulingIgnoredDuringExecution[i]

			schedulingTermString := fmt.Sprintf("spec.template.spec.affinity.nodeAffinity.preferredDuringSchedulingIgnoredDuringExecution[%d]", i)
			opts.Values = append(opts.Values, fmt.Sprintf("%s.weight=%d", schedulingTermString, preferredSchedulingTerm.Weight))

			for j := range preferredSchedulingTerm.Preference.MatchExpressions {
				matchExpression := preferredSchedulingTerm.Preference.MatchExpressions[j]
				matchExpressionString := fmt.Sprintf("%s.preference.matchExpressions[%d]", schedulingTermString, j)
				opts.Values = append(opts.Values, fmt.Sprintf("%s.key=%s", matchExpressionString, matchExpression.Key))
				opts.Values = append(opts.Values, fmt.Sprintf("%s.operator=%s", matchExpressionString, matchExpression.Operator))
				for k := range matchExpression.Values {
					opts.Values = append(opts.Values, fmt.Sprintf("%s.values[%d]=%v", matchExpressionString, k, matchExpression.Values[i]))
				}
			}
		}
	}

	if helmInstall.NodeSelector != "" {
		opts.NodeSelectors = helmInstall.NodeSelector
	}

	if err := installer.Install(kubeClient, &opts); err != nil {
		if !k8sapierrors.IsAlreadyExists(err) {
			//TODO shouldn'T we just skipp?
			return err
		}
		if helmInstall.Upgrade {
			if err := installer.Upgrade(kubeClient, &opts); err != nil {
				return errors.Wrap(err, "error when upgrading")
			}
			klog.Info("Tiller (the Helm server-side component) has been upgraded to the current version.")
		} else {
			klog.Info("Warning: Tiller is already installed in the cluster.")
		}
	} else {
		klog.Info("Tiller (the Helm server-side component) has been installed into your Kubernetes Cluster.")
	}
	klog.Info("Helm install finished")
	return nil
}

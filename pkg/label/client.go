package label

import (
	"fmt"

	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth" // register cloud auth providers for out-of-cluster use
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// NewClient builds a Kubernetes clientset, preferring in-cluster configuration
// and falling back to the given kubeconfig path.
func NewClient(kubeconfig, userAgent string) (kubernetes.Interface, error) {
	cfg, err := rest.InClusterConfig()
	if err != nil {
		cfg, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			return nil, fmt.Errorf("load kubeconfig %q: %w", kubeconfig, err)
		}
	}
	cfg.UserAgent = userAgent
	return kubernetes.NewForConfig(cfg)
}

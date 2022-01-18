package kubernetes

import (
	"fmt"

	scapi "github.com/fluxcd/source-controller/api/v1beta1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func NewClient() (client.Client, error) {
	clientConfig := genericclioptions.ConfigFlags{
		Timeout:    pointer.String("0"),
		KubeConfig: pointer.String(""),
		Context:    pointer.String(""),
	}

	restConfig, err := clientConfig.ToRESTConfig()
	if err != nil {
		return nil, fmt.Errorf("could not create Kubernetes client config: %w", err)
	}

	scheme := runtime.NewScheme()
	if err := scapi.AddToScheme(scheme); err != nil {
		return nil, fmt.Errorf("could not register Flux scheme: %w", err)
	}

	return client.New(restConfig, client.Options{Scheme: scheme})
}

package helm

import (
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/discovery/cached/memory"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
	"k8s.io/client-go/tools/clientcmd"

	_ "k8s.io/client-go/plugin/pkg/client/auth" // load all auth plugins
)

// restClientGetter implements genericclioptions.RESTClientGetter for helm's action.Configuration.
type restClientGetter struct {
	restCfg   *rest.Config
	namespace string
}

func newRESTClientGetter(restCfg *rest.Config, namespace string) *restClientGetter {
	return &restClientGetter{restCfg: restCfg, namespace: namespace}
}

func (r *restClientGetter) ToRESTConfig() (*rest.Config, error) {
	return r.restCfg, nil
}

func (r *restClientGetter) ToDiscoveryClient() (discovery.CachedDiscoveryInterface, error) {
	dc, err := discovery.NewDiscoveryClientForConfig(r.restCfg)
	if err != nil {
		return nil, err
	}
	return memory.NewMemCacheClient(dc), nil
}

func (r *restClientGetter) ToRESTMapper() (meta.RESTMapper, error) {
	dc, err := r.ToDiscoveryClient()
	if err != nil {
		return nil, err
	}
	mapper := restmapper.NewDeferredDiscoveryRESTMapper(dc)
	return mapper, nil
}

func (r *restClientGetter) ToRawKubeConfigLoader() clientcmd.ClientConfig {
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	overrides := &clientcmd.ConfigOverrides{}
	overrides.Context.Namespace = r.namespace
	return clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, overrides)
}

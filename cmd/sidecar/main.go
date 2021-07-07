package main

import (
	"context"
	"flag"
	restful "github.com/emicklei/go-restful"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/klog"
	tenantv1alpha1 "kubesphere.io/api/tenant/v1alpha1"
	"net/http"
	resourcesv1alpha1 "secrets/api/v1alpha1"
	ctrl "sigs.k8s.io/controller-runtime"
	runtimecache "sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
	cache runtimecache.Cache
)

func init() {
	_ = clientgoscheme.AddToScheme(scheme)

	_ = resourcesv1alpha1.AddToScheme(scheme)
	// +kubebuilder:scaffold:scheme
}

func init() {
	klog.InitFlags(flag.CommandLine)
}

func main() {
	flag.Parse()
	cache, _ = runtimecache.New(ctrl.GetConfigOrDie(),runtimecache.Options{
		Scheme:    scheme,
	})


	ws := new(restful.WebService)
	ws.Produces(restful.MIME_JSON)
	ws.Route(ws.GET("/kapis/resources.demo.io/v1alpha1/workspaces/{workspace}/secrets").To(listSecrets))
	ws.Route(ws.GET("/demo.io/metrics").To(metrics))

	restful.Add(ws)

	go cache.Start(ctrl.SetupSignalHandler())

	klog.V(0).Infof("Start listening on %d", 1090)
	klog.Fatal(http.ListenAndServe(":1090", nil))
}

func metrics(req *restful.Request, resp *restful.Response) {
	secrets := &resourcesv1alpha1.SecretList{}
	if err := cache.List(context.Background(),secrets, &client.ListOptions{}); err != nil {
		resp.WriteHeaderAndEntity(http.StatusInternalServerError,errors.NewInternalError(err))
		return
	}

	resp.WriteEntity(struct {
		SecretsCount int `json:"secrets_count"`
	}{
		SecretsCount: len(secrets.Items),
	})
}

func listSecrets(req *restful.Request, resp *restful.Response) {
	workspaceName := req.PathParameter("workspace")

	secrets := &resourcesv1alpha1.SecretList{}
	if err := cache.List(context.Background(),secrets, &client.ListOptions{LabelSelector: labels.SelectorFromSet(labels.Set{tenantv1alpha1.WorkspaceLabel: workspaceName})}); err != nil {
		resp.WriteHeaderAndEntity(http.StatusInternalServerError,errors.NewInternalError(err))
		return
	}

	resp.WriteEntity(secrets)
}

/*


Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	resourcesv1alpha1 "secrets/api/v1alpha1"
)

// SecretReconciler reconciles a Secret object
type SecretReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=resources.demo.io,resources=secrets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=resources.demo.io,resources=secrets/status,verbs=get;update;patch

func (r *SecretReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	rootCtx := context.Background()

	secret := &resourcesv1alpha1.Secret{}
	if err := r.Get(rootCtx, req.NamespacedName, secret); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	//secrets := &corev1.SecretList{}
	//if err := r.List(rootCtx, secrets, &client.ListOptions{LabelSelector: labels.SelectorFromSet(secret.Spec.LabelSelector)}); err != nil {
	//	return ctrl.Result{}, client.IgnoreNotFound(err)
	//}

	//for _, obj := range secrets.Items {
	//	namespace := &corev1.Namespace{}
	//	if err := r.Get(rootCtx, types.NamespacedName{Name: obj.Namespace}, namespace); err != nil {
	//		return ctrl.Result{}, err
	//	}
	//	if v1.IsControlledBy(obj.DeepCopy(), secret) &&  !labels.SelectorFromSet(secret.Spec.LabelSelector).Matches(labels.Set(namespace.GetLabels())) {
	//
	//	}
	//}

	namespaces := &corev1.NamespaceList{}
	if err := r.List(rootCtx, namespaces, &client.ListOptions{LabelSelector: labels.SelectorFromSet(secret.Spec.LabelSelector)}); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	for _, namespace := range namespaces.Items {
		obj := secret.Spec.Secret.DeepCopy()
		obj.Namespace = namespace.Name
		_ = controllerutil.SetControllerReference(secret, obj, r.Scheme)
		if _, err := ctrl.CreateOrUpdate(rootCtx, r.Client, obj, func() error {
			return nil
		}); err != nil {
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

func (r *SecretReconciler) SetupWithManager(mgr ctrl.Manager) error {
	_, err := ctrl.NewControllerManagedBy(mgr).
		WithOptions(controller.Options{
			MaxConcurrentReconciles: 3,
		}).
		For(&resourcesv1alpha1.Secret{}).
		Watches(&source.Kind{Type: &corev1.Namespace{}}, &handler.EnqueueRequestsFromMapFunc{
			ToRequests: handler.ToRequestsFunc(
				func(object handler.MapObject) []reconcile.Request {
					secrets := &resourcesv1alpha1.SecretList{}
					if err := r.List(context.Background(), secrets, &client.ListOptions{LabelSelector: labels.Everything()}); err != nil {
						return nil
					}
					requests := make([]reconcile.Request, 0)
					for _, secret := range secrets.Items {
						if labels.SelectorFromSet(secret.Spec.LabelSelector).Matches(labels.Set(object.Meta.GetLabels())) {
							request := reconcile.Request{
								NamespacedName: types.NamespacedName{Name: secret.Name, Namespace: secret.Namespace},
							}
							requests = append(requests, request)
						}
					}
					return requests
				},
			),
		}).
		Build(r)
	return err
}

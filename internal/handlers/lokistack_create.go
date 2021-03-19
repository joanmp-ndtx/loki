package handlers

import (
	"context"

	"github.com/ViaQ/logerr/log"
	lokiv1beta1 "github.com/ViaQ/loki-operator/api/v1beta1"
	"github.com/ViaQ/loki-operator/internal/manifests"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// CreateLokiStack handles a LokiStack create event
func CreateLokiStack(ctx context.Context, req ctrl.Request, k8s client.Client) error {
	ll := log.WithValues("lokistack", req.NamespacedName, "event", "create")

	var stack lokiv1beta1.LokiStack
	if err := k8s.Get(ctx, req.NamespacedName, &stack); err != nil {
		if apierrors.IsNotFound(err) {
			// maybe the user deleted it before we could react? Either way this isn't an issue
			ll.Error(err, "could not find the requested loki stack", "name", req.NamespacedName)
			return nil
		}
	}

	// Here we will translate the lokiv1beta1.LokiStack options into manifest options
	opts := manifests.Options{
		Name:      req.Name,
		Namespace: req.Namespace,
	}

	ll.Info("begin building manifests")

	objects, err := manifests.BuildAll(opts)
	if err != nil {
		ll.Error(err, "failed to build manifests")
		return err
	}
	ll.Info("manifests built", "count", len(objects))
	for _, obj := range objects {
		l := ll.WithValues("object_name", obj.GetName(),
			"object_kind", obj.GetObjectKind(),
			"object", obj)

		obj.SetNamespace(req.Namespace)
		SetOwner(stack, obj)
		if err := k8s.Create(ctx, obj); err != nil {
			l.Error(err, "failed to create object")
			continue
		}
		l.Info("Resource created", "resource", obj.GetName())
	}

	return nil
}

func SetOwner(stack lokiv1beta1.LokiStack, o client.Object) {
	o.SetOwnerReferences(append(o.GetOwnerReferences(), metav1.OwnerReference{
		APIVersion: lokiv1beta1.GroupVersion.String(),
		Kind:       stack.Kind,
		Name:       stack.Name,
		UID:        stack.UID,
		Controller: pointer.BoolPtr(true),
	}))
}

package main

import (
	"time"

	log "github.com/Sirupsen/logrus"
	"gopkg.in/alecthomas/kingpin.v2"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/pkg/api"
	"k8s.io/client-go/pkg/apis/extensions/v1beta1"
	"k8s.io/client-go/pkg/fields"
	"k8s.io/client-go/pkg/types"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	debug          = kingpin.Flag("debug", "Enable debug mode.").Short('d').Bool()
	masterURL      = kingpin.Flag("master", "").String()
	kubeconfigPath = kingpin.Flag("kubeconfig", "").Default(clientcmd.NewDefaultPathOptions().GetDefaultFilename()).Envar(clientcmd.RecommendedConfigPathEnvVar).String()
	namespace      = kingpin.Flag("namespace", "").Default(api.NamespaceAll).Short('n').String()
)

func main() {
	v, err := version()
	if err != nil {
		log.Error(err.Error())
	}

	kingpin.Version(v)

	kingpin.CommandLine.HelpFlag.Short('h')
	kingpin.CommandLine.VersionFlag.Short('v')

	kingpin.Parse()

	if *debug {
		log.SetLevel(log.DebugLevel)
	}

	config, err := clientcmd.BuildConfigFromFlags(*masterURL, *kubeconfigPath)
	if err != nil {
		log.Panic(err.Error())
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Panic(err.Error())
	}

	lw := cache.NewListWatchFromClient(
		clientset.Extensions().RESTClient(),
		"deployments",
		*namespace,
		fields.Everything(),
	)

	deployments := make(map[types.UID]bool)

	_, controller := cache.NewInformer(
		lw,
		&v1beta1.Deployment{},
		time.Second*0,
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				deployment := obj.(*v1beta1.Deployment)

				deployments[deployment.ObjectMeta.UID] = true

				log.WithFields(log.Fields{
					"namespace": deployment.ObjectMeta.Namespace,
					"name":      deployment.ObjectMeta.Name,
				}).Debug("Deployment added")
			},
			UpdateFunc: func(_, newObj interface{}) {
				deployment := newObj.(*v1beta1.Deployment)

				log.WithFields(log.Fields{
					"namespace": deployment.ObjectMeta.Namespace,
					"name":      deployment.ObjectMeta.Name,
				}).Debug("Deployment updated")

				log.WithFields(log.Fields{
					".metadata.generation":       deployment.ObjectMeta.Generation,
					".status.observedGeneration": deployment.Status.ObservedGeneration,
					".spec.replicas":             *deployment.Spec.Replicas,
					".status.replicas":           deployment.Status.Replicas,
				}).Debug()

				if rolloutStatus(deployment) && deployments[deployment.ObjectMeta.UID] {
					deployments[deployment.ObjectMeta.UID] = false

					log.WithFields(log.Fields{
						"namespace": deployment.ObjectMeta.Namespace,
						"name":      deployment.ObjectMeta.Name,
					}).Info("Deployment rolled out")
				} else if !deployments[deployment.ObjectMeta.UID] {
					deployments[deployment.ObjectMeta.UID] = true

					log.WithFields(log.Fields{
						"namespace": deployment.ObjectMeta.Namespace,
						"name":      deployment.ObjectMeta.Name,
					}).Info("Rolling out deployment")
				}
			},
			DeleteFunc: func(obj interface{}) {
				deployment := obj.(*v1beta1.Deployment)

				delete(deployments, deployment.ObjectMeta.UID)

				log.WithFields(log.Fields{
					"namespace": deployment.ObjectMeta.Namespace,
					"name":      deployment.ObjectMeta.Name,
				}).Debug("Deployment deleted")
			},
		},
	)

	stopCh := make(chan struct{})
	defer close(stopCh)
	controller.Run(stopCh)
	<-stopCh
}

func rolloutStatus(deployment *v1beta1.Deployment) bool {
	// https://github.com/kubernetes/kubernetes/blob/v1.5.1/pkg/kubectl/rollout_status.go#L46-L78

	if deployment.Status.ObservedGeneration < deployment.ObjectMeta.Generation {
		return false
	}

	if deployment.Status.UpdatedReplicas != *deployment.Spec.Replicas {
		return false
	}

	if deployment.Status.Replicas > deployment.Status.UpdatedReplicas {
		return false
	}

	return deployment.Status.AvailableReplicas >= deployment.Status.UpdatedReplicas
}

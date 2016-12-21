package main

import (
	"flag"
	"fmt"

  "github.com/mitchellh/go-homedir"
	"k8s.io/client-go/kubernetes"
	meta_v1 "k8s.io/client-go/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/clientcmd"
)

const namespace string = "default"

func main() {
  flag.Parse()

  dir, _ := homedir.Dir()
	config, err := clientcmd.BuildConfigFromFlags("", dir + "/.kube/config")
	if err != nil {
		panic(err.Error())
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

  name := flag.Arg(0)
  deployment, err := clientset.Extensions().Deployments(namespace).Get(name, meta_v1.GetOptions{})
  if err != nil {
    panic(err.Error())
  }

  for _, container := range deployment.Spec.Template.Spec.Containers {
    fmt.Printf("%s: %s\n", container.Name, container.Image)
  }
}

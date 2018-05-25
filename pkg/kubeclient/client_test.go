package kubeclient_test

import (
	"context"

	"github.com/kubernetes-sigs/kubebuilder/pkg/kubeclient"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func ExampleClient_Create() {
	kc, err := kubeclient.FromConfigOrCluster("/home/dir/config/.kubeconfig")
	if err != nil {
		// handle err
	}
	pod := &v1.Pod{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "",
			Kind:       "Pod",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "MyPod",
			Namespace: "ns-1",
		},
		Spec: {},
	}
	ctx := context.Background()
	err = kc.Create(ctx, pod)
	if err != nil {
		// handle err
	}
}

func ExampleClient_Get() {
	kc, err := kubeclient.FromConfigOrCluster("/home/dir/config/.kubeconfig")
	if err != nil {
		// handle err
	}
	pod := &v1.Pod{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "",
			Kind:       "Pod",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "MyPod",
			Namespace: "ns-1",
		},
	}
	ctx := context.Background()
	err = kc.Get(ctx, pod)
	if err != nil {
		// handle err
	}
}

func ExampleClient_List() {
	kc, err := kubeclient.FromConfigOrCluster("/home/dir/config/.kubeconfig")
	if err != nil {
		// handle err
	}
	podList := &v1.PodList{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Pod",
		},
	}
	ctx := context.Background()
	opts := &metav1.ListOptions{
		LabelSelector: labels.SelectorFromSet(map[string]string{"app": "app-label"}).String(),
	}
	err = kc.List(ctx, "ns-1", podList, opts)
	if err != nil {
		// handle err
	}
	// podList.Items, which is []v1.Pod type, will be populated at this point.
	for _, p := range podList.Items {
		p.Spec.Image = ""
		err := kc.Update(ctx, p)
	}
}

func ExampleFromConfigOrCluster() {
	// create client given kubeconfig path
	kubeConfigPath := ""
	kc, err := kubeclient.FromConfigOrCluster(kubeConfigPath)
	if err != nil {
		// handle err
	}
}

func ExampleInCluster() {
	// creating client within a cluster
	kc, err := kubeclient.InCluster()
	if err != nil {
		// handle err
	}
}

func ExampleFromConfig() {
	// create client given kubeconfig path, typically in CLI application.
	path := "" // may be read from commandline flags
	kc, err := kubeclient.FromConfig(path)
	if err != nil {
		// handle err
	}
}

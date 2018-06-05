package client_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/kubernetes-sigs/kubebuilder/pkg/client"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/clientcmd"
)

// ExampleClient demonstrates a basic workflow using the client.
func ExampleClient() {
	// use helper methods to build config for client.
	conf, err := clientcmd.BuildConfigFromFlags("", "")
	if err != nil {
		// handle err
	}

	// create a Kubernetes client using the config.
	kc, err := client.NewForConfig(conf)
	if err != nil {
		// handle err
	}
	ctx := context.Background()

	// fetch the Pod with name pod-1 in namespace ns-1
	pod := &v1.Pod{}
	podKey := client.ObjectKey{Namespace: "ns-1", Name: "pod-1"}
	err = kc.Get(ctx, podKey, pod)
	if err != nil {
		// handle err
	}

	// update container image in Pod and update it in the cluster
	pod.Spec.Containers[0].Image = "image:v0.0.2"
	err = kc.Update(ctx, pod)
	if err != nil {
		// handle err
	}

	// delete a Pod
	podToDelete := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "MyPod",
			Namespace: "ns-1",
		},
	}
	err = kc.Delete(ctx, podToDelete)
	if err != nil {
		// handle err
	}

	// List pods in a namespace with labels foo=bar
	podList := &v1.PodList{}
	opts := metav1.ListOptions{
		LabelSelector: labels.SelectorFromSet(map[string]string{"foo": "bar"}).String(),
	}
	err = kc.List(ctx, "ns-1", opts, podList)
	if err != nil {
		// handle err
	}
	// podList.Items, which is []v1.Pod type, will be populated at this point.
	for _, p := range podList.Items {
		// print p
		fmt.Println(p)
	}
}

func TestClientCRUD(t *testing.T) {
	// get the base config ?
	conf, err := clientcmd.BuildConfigFromFlags("", filepath.Join(os.Getenv("HOME"), ".kube", "config"))
	if err != nil {
		t.Fatalf("failed to create config: %v", err)
	}

	kc, err := client.NewForConfig(conf)
	if err != nil {
		t.Fatalf("error creating k8s client: %v", err)
	}

	pod := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "mypod",
			Namespace: "default",
		},
		Spec: v1.PodSpec{
			Containers: []v1.Container{{
				Image: "image-that-does-not-exist",
				Name:  "ttyd",
			}},
		},
	}
	err = kc.Create(context.Background(), pod)
	if err != nil {
		t.Fatalf("error in creating pod with name: %v", err)
	}
	t.Logf("created pod 'mypod' successfully")

	podKey := client.ObjectKey{Namespace: "default", Name: "mypod"}
	err = kc.Get(context.Background(), podKey, pod)
	if err != nil {
		t.Fatalf("error in retrieving pod with name: %v", err)
	}
	t.Logf("found pod '%s' \n", pod.Name)

	pod.Spec.Containers[0].Image = "image-2"
	err = kc.Update(context.Background(), pod)
	if err != nil {
		t.Fatalf("error in updating the pod '%s'", pod.Name)
	}

	updatedPod := &v1.Pod{}
	err = kc.Get(context.Background(), podKey, updatedPod)
	if err != nil {
		t.Fatalf("error in retrieving pod with name: %v", err)
	}
	if updatedPod.Spec.Containers[0].Image != "image-2" {
		t.Errorf("expected pod to have updated image")
	}

	podList := &v1.PodList{}
	opts := metav1.ListOptions{}
	err = kc.List(context.Background(), "default", opts, podList)
	if err != nil {
		t.Fatalf("error in fetching pods: %v", err)
	}
	for i, p := range podList.Items {
		t.Logf("[%d]: %s \n", i, p.Name)
	}

	err = kc.Delete(context.Background(), pod)
	if err != nil {
		t.Fatalf("error in deleting the pod: %v", err)
	}
	t.Logf("deleted pod '%s' successfully", pod.Name)
}

func TestMain(m *testing.M) {
	setup()
	rc := m.Run()
	teardown()
	os.Exit(rc)
}

func setup() error {
	fmt.Println("setting up...")
	return nil
}

func teardown() error {
	fmt.Println("tearing down...")
	return nil
}

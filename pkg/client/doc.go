/*
Package client provides Kubernetes client implementation.

Client implements methods to perform CRUD operations on Kubernetes objects in a cluster.

Create, Get, Update, Delete and List Kubernetes Objects:

	// use helper methods to build config for client.
	conf, err := clientcmd.BuildConfigFromFlags("", "")
	if err != nil {
		// handle err
	}

	// create a Kubernetes client using the config.
	kc := client.NewForConfig(conf)
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
	opts := &metav1.ListOptions{
		LabelSelector: labels.SelectorFromSet(map[string]string{"foo": "bar"}).String(),
	}
	err = kc.List(ctx, "ns-1", opts, podList)
	if err != nil {
		// handle err
	}
	// podList.Items, which is []v1.Pod type, will be populated at this point.
	for _, p := range podList.Items {
		// print p
	}
*/
package client

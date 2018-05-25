/*
Package kubeclient provides Kubernetes client implementation.

Client implements methods to perform CRUD operations on Kubernetes objects in a cluster.

Create, Get, Update or Delete Kubernetes Objects:

	kc, err := kubeclient.FromConfigOrCluster("/home/dir/config/.kubeconfig")
	...
	pod := &v1.Pod{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "",
			Kind:       "Pod",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "MyPod",
			Namespace: "ns-1",
		},
		Spec: {
			.....
			.....
		}
	}
	ctx := context.Background()
	....
	err := kc.Get(ctx, pod)
	...
	err := kc.Create(ctx, pod)
	...

	pod.Spec.Image = ....
	err := kc.Update(ctx, pod)
	...


List Kubernetes Objects:
	ctx := context.Background()

	kc, err := kubeclient.FromConfigOrCluster("/home/dir/config/.kubeconfig")
	...
	podList := &v1.PodList{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Pod",
		},
	}
	err = kc.List(ctx, "namespace-1", podList)
	...
	for _, p := range podList.Items {
		p.Spec.Image = ....
		err := kc.Update(ctx, p)
		..
	}
	....
*/
package kubeclient

## 在集成测试中使用 envtest
[`controller-runtime`](http://sigs.k8s.io/controller-runtime) 提供 `envtest` ([godoc](https://pkg.go.dev/sigs.k8s.io/controller-runtime/pkg/envtest?tab=doc))，这个包可以帮助你为你在 etcd 和 Kubernetes API server 中设置并启动的 controllers 实例来写集成测试，不需要 kubelet，controller-manager 或者其他组件。

可以根据以下通用流程在集成测试中使用 `envtest`：

```go
import sigs.k8s.io/controller-runtime/pkg/envtest

//指定 testEnv 配置
testEnv = &envtest.Environment{
	CRDDirectoryPaths: []string{filepath.Join("..", "config", "crd", "bases")},
}

//启动 testEnv
cfg, err = testEnv.Start()

//编写测试逻辑

//停止 testEnv
err = testEnv.Stop()
```

`kubebuilder` 为你提供了 testEnv 的设置和清除模版，在生成的 `/controllers` 目录下的 ginkgo 测试套件中。


测试运行中的 Logs 以 `test-env` 为前缀。

### 配置你的测试控制面
你可以在你的集成测试中使用环境变量和/或者标记位来指定 `api-server` 和 `etcd` 设置。
#### 环境变量

| 变量名称   |    类型   | 使用时机 |
| --- | :--- | :--- |
| `USE_EXISTING_CLUSTER` | boolean | 可以指向一个已存在 cluster 的控制面，而不用设置一个本地的控制面。 |
| `KUBEBUILDER_ASSETS` | 目录路径 | 将集成测试指向一个包含所有二进制文件（api-server，etcd 和 kubectl）的目录。 |
| `TEST_ASSET_KUBE_APISERVER`, `TEST_ASSET_ETCD`, `TEST_ASSET_KUBECTL` | 分别代表 api-server，etcd，和 kubectl 二进制文件的路径 | 和 `KUBEBUILDER_ASSETS` 相似，但是更细一点。指示集成测试使用非默认的二进制文件。这些环境变量也可以被用来确保特定的测试是在期望版本的二进制文件下运行的。|
| `KUBEBUILDER_CONTROLPLANE_START_TIMEOUT` 和 `KUBEBUILDER_CONTROLPLANE_STOP_TIMEOUT` | [`time.ParseDuration`](https://golang.org/pkg/time/#ParseDuration) 支持的持续时间的格式 | 指定不同于测试控制面（分别）启动和停止的超时时间；任何超出设置的测试都会运行失败。|
| `KUBEBUILDER_ATTACH_CONTROL_PLANE_OUTPUT` | boolean | 设置为 `true` 可以将控制面的标准输出和标准错误贴合到 os.Stdout 和 os.Stderr 上。这种做法在调试测试失败时是非常有用的，因为输出包含控制面的输出。|

#### 标记位

下面是一个在你的集成测试中通过修改标记位来启动 API server 的例子，和 `envtest.DefaultKubeAPIServerFlags` 中的默认值相对比：
```go 
var _ = BeforeSuite(func(done Done) {
	Expect(os.Setenv("TEST_ASSET_KUBE_APISERVER", "../testbin/bin/kube-apiserver")).To(Succeed())
	Expect(os.Setenv("TEST_ASSET_ETCD", "../testbin/bin/etcd")).To(Succeed())
	Expect(os.Setenv("TEST_ASSET_KUBECTL", "../testbin/bin/kubectl")).To(Succeed())

	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))
	testenv = &envtest.Environment{}

	_, err := testenv.Start()
	Expect(err).NotTo(HaveOccurred())

	close(done)
}, 60)

var _ = AfterSuite(func() {
	Expect(testenv.Stop()).To(Succeed())

	Expect(os.Unsetenv("TEST_ASSET_KUBE_APISERVER")).To(Succeed())
	Expect(os.Unsetenv("TEST_ASSET_ETCD")).To(Succeed())
	Expect(os.Unsetenv("TEST_ASSET_KUBECTL")).To(Succeed())

})
```  

```go
customApiServerFlags := []string{
	"--secure-port=6884",
	"--admission-control=MutatingAdmissionWebhook",
}

apiServerFlags := append([]string(nil), envtest.DefaultKubeAPIServerFlags...)
apiServerFlags = append(apiServerFlags, customApiServerFlags...)

testEnv = &envtest.Environment{
	CRDDirectoryPaths: []string{filepath.Join("..", "config", "crd", "bases")},
	KubeAPIServerFlags: apiServerFlags,
}
```

### 测试注意事项

除非你在使用一个已存在的 cluster，否则需要记住在测试内容中没有内置的 controllers 在运行。在某些方面，测试控制面会表现的和“真实” clusters 有点不一样，这可能会对你如何写测试有些影响。一个很常见的例子就是垃圾回收；因为没有 controllers 来监控内置的资源，对象是不会被删除的，即使设置了 `OwnerReference`。 

为了测试删除生命周期是否工作正常，要测试所有权而不是仅仅判断是否存在。比如：

```go
expectedOwnerReference := v1.OwnerReference{
	Kind:       "MyCoolCustomResource",
	APIVersion: "my.api.example.com/v1beta1",
	UID:        "d9607e19-f88f-11e6-a518-42010a800195",
	Name:       "userSpecifiedResourceName",
}
Expect(deployment.ObjectMeta.OwnerReferences).To(ContainElement(expectedOwnerReference))
```

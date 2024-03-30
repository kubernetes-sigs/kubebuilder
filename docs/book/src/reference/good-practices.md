# Good Practices

## What is "Reconciliation" in Operators?

When you create a project using Kubebuilder, see the scaffolded code generated under `cmd/main.go`. This code initializes a [Manager][controller-runtime-manager], and the project relies on the [controller-runtime][controller-runtime] framework. The Manager manages [Controllers][controllers], which offer a reconcile function that synchronizes resources until the desired state is achieved within the cluster.

Reconciliation is an ongoing loop that executes necessary operations to maintain the desired state, adhering to Kubernetes principles, such as the [control loop][k8s-control-loop]. For further information, check out the [Operator patterns][k8s-operator-pattern] documentation from Kubernetes to better understand those concepts.

## Why should reconciliations be idempotent?

When developing operators, the controller’s reconciliation loop needs to be idempotent. By following the [Operator pattern][operator-pattern] we create [controllers][controllers] that provide a reconcile function responsible for synchronizing resources until the desired state is reached on the cluster. Developing idempotent solutions will allow the reconciler to correctly respond to generic or unexpected events, easily deal with application startup or upgrade. More explanation on this is available [here][controller-runtime-topic].

Writing reconciliation logic according to specific events, breaks the recommendation of operator pattern and goes against the design principles of [controller-runtime][controller-runtime]. This may lead to unforeseen consequences, such as resources becoming stuck and requiring manual intervention.

## Understanding Kubernetes APIs and following API conventions

Building your operator commonly involves extending the Kubernetes API itself. It is helpful to understand precisely how Custom Resource Definitions (CRDs) interact with the Kubernetes API. Also, the [Kubebuilder documentation][docs] on Groups and Versions and Kinds may be helpful to understand these concepts better as they relate to operators.

Additionally, we recommend checking the documentation on [Operator patterns][operator-pattern] from Kubernetes to better understand the purpose of the standard solutions built with KubeBuilder.

## Why you should adhere to the Kubernetes API conventions and standards

Embracing the [Kubernetes API conventions and standards][k8s-api-conventions] is crucial for maximizing the potential of your applications and deployments. By adhering to these established practices, you can benefit in several ways.

Firstly, adherence ensures seamless interoperability within the Kubernetes ecosystem. Following conventions allows your applications to work harmoniously with other components, reducing compatibility issues and promoting a consistent user experience.

Secondly, sticking to API standards enhances the maintainability and troubleshooting of your applications. Adopting familiar patterns and structures makes debugging and supporting your deployments easier, leading to more efficient operations and quicker issue resolution.

Furthermore, leveraging the Kubernetes API conventions empowers you to harness the platform's full capabilities. By working within the defined framework, you can leverage the rich set of features and resources offered by Kubernetes, enabling scalability, performance optimization, and resilience.

Lastly, embracing these standards future-proofs your native solutions. By aligning with the evolving Kubernetes ecosystem, you ensure compatibility with future updates, new features, and enhancements introduced by the vibrant Kubernetes community.

In summary, by adhering to the Kubernetes API conventions and standards, you unlock the potential for seamless integration, simplified maintenance, optimal performance, and future-readiness, all contributing to the success of your applications and deployments.

## Why should one avoid a system design where a single controller is responsible for managing multiple CRDs (Custom Resource Definitions)(for example, an _'install_all_controller.go'_)?

Avoid a design solution where the same controller reconciles more than one Kind. Having many Kinds (such as CRDs), that are all managed by the same controller, usually goes against the design proposed by controller-runtime. Furthermore, this might hurt concepts such as encapsulation, the Single Responsibility Principle, and Cohesion. Damaging these concepts may cause unexpected side effects and increase the difficulty of extending, reusing, or maintaining the operator.
Having one controller manage many Custom Resources (CRs) in an Operator can lead to several issues:

- **Complexity**: A single controller managing multiple CRs can increase the complexity of the code, making it harder to understand, maintain, and debug.
- **Scalability**: Each controller typically manages a single kind of CR for scalability. If a single controller handles multiple CRs, it could become a bottleneck, reducing the overall efficiency and responsiveness of your system.
- **Single Responsibility Principle**: Following this principle from software engineering, each controller should ideally have only one job. This approach simplifies development and debugging, and makes the system more robust.
- **Error Isolation**: If one controller manages multiple CRs and an error occurs, it could potentially impact all the CRs it manages. Having a single controller per CR ensures that an issue with one controller or CR does not directly affect others.
- **Concurrency and Synchronization**: A single controller managing multiple CRs could lead to race conditions and require complex synchronization, especially if the CRs have interdependencies.

In conclusion, while it might seem efficient to have a single controller manage multiple CRs, it often leads to higher complexity, lower scalability, and potential stability issues. It's generally better to adhere to the single responsibility principle, where each CR is managed by its own controller.

## Why is it recommended to avoid a scenario where multiple controllers are updating the same Custom Resource (CR)?

Managing a single Custom Resource (CR) with multiple controllers can lead to several challenges:
- **Race conditions**: When multiple controllers attempt to reconcile the same CR concurrently, race conditions can emerge. These conditions can produce inconsistent or unpredictable outcomes. For example, if we try to update the CR to add a status condition, we may encounter a range of errors such as “the object has been modified; please apply your changes to the latest version and try again”, triggering a repetitive reconciliation process.
- **Concurrency issues**: When controllers have different interpretations of the CR’s state, they may constantly overwrite each other’s changes. This conflict can create a loop, with the controllers ceaselessly disputing the CR’s state.
- **Maintenance and support difficulties**: Coordinating the logic for multiple controllers operating on the same CR can increase system complexity, making it more challenging to understand or troubleshoot. Typically, a system’s behavior is easier to comprehend when each CR is managed by a single controller.
- **Status tracking complications**: We may struggle to work adequately with status conditions to accurately track the state of each component managed by the Installer.
- **Performance issues**: If multiple controllers are watching and reconciling the Installer Kind, redundant operations may occur, leading to unnecessary resource usage.
These challenges underline the importance of assigning each controller the single responsibility of managing its own CR. This will streamline our processes and ensure a more reliable system.

## Why You Should Adopt Status Conditions

We recommend you manage your solutions using Status Conditionals following the [K8s Api conventions][k8s-api-conventions] because:

- **Standardization**: Conditions provide a standardized way to represent the state of an Operator's custom resources, making it easier for users and tools to understand and interpret the resource's status.
- **Readability**: Conditions can clearly express complex states by using a combination of multiple conditions, making it easier for users to understand the current state and progress of the resource.
- **Extensibility**: As new features or states are added to your Operator, conditions can be easily extended to represent these new states without requiring significant changes to the existing API or structure.
- **Observability**: Status conditions can be monitored and tracked by cluster administrators and external monitoring tools, enabling better visibility into the state of the custom resources managed by the Operator.
- **Compatibility**: By adopting the common pattern of using conditions in Kubernetes APIs, Operator authors ensure their custom resources align with the broader ecosystem, which helps users to have a consistent experience when interacting with multiple Operators and resources in their clusters.

<aside class="note">
<h1> Example of Usage </h1>

Check out the [Deploy Image Plugin][deploy-image]. This plugin allows users to scaffold API/Controllers to deploy and manage an Operand (image) on the cluster following the guidelines and best practices. It abstracts the
complexities of achieving this goal while allowing users to customize the generated code.

Therefore, you can check an example of Status Conditional usage by looking at its API(s) scaffolded and code implemented under the Reconciliation into its Controllers.

</aside>

[docs]: /cronjob-tutorial/gvks.html
[operator-pattern]: https://kubernetes.io/docs/concepts/extend-kubernetes/operator/
[controllers]: https://kubernetes.io/docs/concepts/architecture/controller/
[controller-runtime-topic]: https://github.com/kubernetes-sigs/controller-runtime/blob/main/FAQ.md#q-how-do-i-have-different-logic-in-my-reconciler-for-different-types-of-events-eg-create-update-delete
[controller-runtime]: https://github.com/kubernetes-sigs/controller-runtime
[deploy-image]: /plugins/deploy-image-plugin-v1-alpha.md
[controller-runtime-manager]: https://github.com/kubernetes-sigs/controller-runtime/blob/304027bcbe4b3f6d582180aec5759eb4db3f17fd/pkg/manager/manager.go#L53
[k8s-api-conventions]: https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md
[k8s-control-loop]: https://kubernetes.io/docs/concepts/architecture/controller/
[k8s-operator-pattern]: https://kubernetes.io/docs/concepts/extend-kubernetes/operator/

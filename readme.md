# Network DRA

Example of a DRA driver for call CNI on container creation.

## Demo

```
kind create cluster --config example/kind.yaml
```

```
kubectl apply -f https://raw.githubusercontent.com/k8snetworkplumbingwg/multus-cni/master/e2e/templates/cni-install.yml.j2
kubectl apply -f https://raw.githubusercontent.com/k8snetworkplumbingwg/multus-cni/master/deployments/multus-daemonset-thick.yml
```

```
helm install network-dra deployments/network-DRA
kubectl apply -f examples/pod.yaml
```

```
kubectl delete -f examples/pod.yaml ; helm delete network-dra
```

## Flow

![Flow](docs/resources/Diagrams-Call-Flow.png)

1. Kubelet calls the DRA Plugin (kubelet plugin) with the claims that must be prepared on that node (The node selection has already happened at that time and is not covered by this demo).
2. The DRA plugin is writing CDI Device (a file) which contains a Hook on the createRuntime event and that call a program call `Hook-Callback` with the claim-uid and a socket file path as parameter (more parameters could be passed, e.g. CNI Config).
3. The list of added CDI Devices is returned to kubelet.
4. Kubectl calls CreateContainer with the list of CDI Devices which are to be used in the containerd.
5. containerd call the `Hook-Callback` program on the createRuntime event.
6. `Hook-Callback` receives the OCI State and the parameters (claim uid + socket path) and then call the CreateContainer via the socket passed in parameter (the server runs in the dra-plugin container).
7. The DRA plugins server finds the PodSandboxID in the OCI State and get the pod status via the CRI API in order to get the network namespace.
8. The CNI is called with the network namespace and other information retrieved in the pod status.

The CNI called by this demo is a hardcoded MACVLAN cni config. The CNI call is going through the Multus Thick API which handles the real CNI call.

## Resources

- DRA KEP: https://github.com/kubernetes/enhancements/blob/master/keps/sig-node/3063-dynamic-resource-allocation/README.md
- CDI: https://github.com/cncf-tags/container-device-interface
- DRA API: https://github.com/kubernetes/kubernetes/blob/v1.29.3/staging/src/k8s.io/kubelet/pkg/apis/dra/v1alpha3/api.proto#L34
- DRA Controller: https://pkg.go.dev/k8s.io/dynamic-resource-allocation/controller
- OCI Hooks: https://github.com/opencontainers/runtime-spec/blob/v1.2.0/runtime.md#lifecycle
- DRA Example: https://github.com/kubernetes-sigs/dra-example-driver
- DRA Presentation: https://kccnceu2023.sched.com/event/1HyWy/device-plugins-20-how-to-build-a-driver-for-dynamic-resource-allocation-kevin-klues-nvidia-alexey-fomenko-intel
- Network Device (Pod Resources?): https://github.com/opencontainers/runtime-spec/issues/1239
- Hot pluggable: https://github.com/cncf-tags/container-device-interface/issues/154

# Network DRA

Example of a DRA driver for calling CNIs on container creation.

Slides: https://docs.google.com/presentation/d/1wxR6vAMK2Wl--ZqjnOZDJtvtJHtQe0_OEJH_h2lp2TI/edit?usp=sharing

## Build

Generate Code (Proto, API, ...)
```
make generate
```

build/push (default registry: localhost:5000/network-dra)
```
make REGISTRY=localhost:5000/network-dra
```

## Demo

Create Kind Cluster
```
kind create cluster --config examples/kind.yaml
```

Load Images in Kind
```
kind load docker-image localhost:5000/network-dra/network-dra-controller:latest
kind load docker-image localhost:5000/network-dra/network-dra-plugin:latest
```

Install CNI Plugins + Multus (Required for the NetworkAttachmentDefinition and the Thick API)
```
kubectl apply -f https://raw.githubusercontent.com/k8snetworkplumbingwg/multus-cni/master/e2e/templates/cni-install.yml.j2
kubectl apply -f https://raw.githubusercontent.com/k8snetworkplumbingwg/multus-cni/master/deployments/multus-daemonset-thick.yml
```

Install DRA Plugin
```
helm install network-dra deployments/network-DRA --set registry=localhost:5000/network-dra
```

Demo
```
kubectl apply -f examples/demo-a.yaml
kubectl apply -f examples/demo-b.yaml
kubectl apply -f examples/demo-c.yaml
```

- Demo A
    - Single Pod with a single resource claim.
    - The pod will receive the interface described in the `macvlan-eth0-attachment` resource claim parameter.
- Demo B
    - Single Pod with a  single resource claim (Resource claim used in demo-a).
    - The pod will be in pending state since the resource claim is already in use (by demo-a).
- Demo C
    - Deployment that uses a resource claim template.
    - 2 Pods will be running and new resource claims will be created for each of them.
    - The 2 pods will receive the interface described in the `macvlan-eth0-attach` resource claim template parameter.

## Flow

![Flow](docs/resources/Diagrams-Call-Flow.png)

1. Kubelet calls the DRA Plugin (kubelet plugin) with the claims that must be prepared on that node (The node selection has already happened at that time and is not covered by this demo).
2. The DRA plugin is writing CDI Device (a file) which contains a Hook on the createRuntime event and that call a program call `network-dra-oci-hook` with the claim uid, claim name, claim namespace, Network Specs (CNI Config, Attachment, ...) and a socket file path as parameter.
3. The list of added CDI Devices (filenames) is returned to kubelet.
4. Kubectl calls CreateContainer via the CRI API with the list of CDI Devices which are to be used.
5. Containerd handles the CreateContainer call.
    - 5.0. Containerd builds the OCI Spec from the CRI ContainerConfig, reads the CDI Device files and "merges" the CDI Devices to the OCI Spec.
    - 5.1. Containerd calls runc with the OCI Spec
    - 5.2. Runc runs the `network-dra-oci-hook` program on the createRuntime event with the parameter from the CDI Device file and passes the OCI State over STDIN.
6. `network-dra-oci-hook` receives the OCI State and the parameters and then call the CreateRuntime via the socket passed in parameter (the server runs in the dra-plugin container).
    - Note: 7/8/9 could also be done directly from the `network-dra-oci-hook` without calling this CreateRuntime API function.
7. The DRA plugin server retrieves the network namespace, PodSandboxID (Can be done via CRI API or config file passed in OCI State).
8. The DRA plugin creates the network attachment based on the parameters it received. This can be done using CNI, KNI, Multus Thick API (Server that calls CNI) or anything else.
9. TODO: Expose the status of the attachment.

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

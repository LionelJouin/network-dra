# Network DRA

Example of a DRA integration with an NRI Plugin for calling CNIs on pod creation.

This is a PoC (Proof Of Concept) about resource configuration. A Kubernetes fork implementing KEP-4817 is used ([github.com/LionelJouin/kubernetes/tree/KEP-4817](https://github.com/LionelJouin/kubernetes/tree/KEP-4817)).

PoCs: 
* [v0.0.1 - DRA with CDI Calling CNI with hardcoded CNI specs](https://github.com/LionelJouin/network-dra/tree/v0.0.1)
* [v0.1.0 - DRA with CDI callling CNI with CRD exposing CNI specs](https://github.com/LionelJouin/network-dra/tree/v0.1.0)
    * [Slides](https://docs.google.com/presentation/d/1wxR6vAMK2Wl--ZqjnOZDJtvtJHtQe0_OEJH_h2lp2TI/edit?usp=sharing)
    * [Recording](https://www.youtube.com/watch?v=GdGtEW3ZGHk)
* [v0.1.1 - DRA with CDI callling CNI with CRD exposing CNI specs + default/primary network via DRA](https://github.com/LionelJouin/network-dra/tree/v0.1.1)
* [v0.2.0 - DRA with NRI callling CNI with CRD exposing CNI specs](https://github.com/LionelJouin/network-dra/tree/v0.2.0)
    * [Slides](https://docs.google.com/presentation/d/1CdIexp2Kaf38ktxd-kg5vE4RxjyOjSMzH-P8kUuOxCQ/edit?usp=sharing)
    * [Recording](https://www.youtube.com/watch?v=qNooLu7DWj4)
* [v0.2.1 - DRA (Kubernetes v1.31) with NRI callling CNI with CRD exposing CNI specs](https://github.com/LionelJouin/network-dra/releases/tag/v0.2.1)
* Current - DRA with NRI calling CNI with Opaque parameter exposing CNI config and reporting CNI result in ResourceClaim Status
    * [kubernetes/Kubernetes fork implementing KEP-4817](https://github.com/LionelJouin/kubernetes/tree/KEP-4817)
    * [kubernetes-sigs/multi-network fork implementing the DRA-Driver + CNI Add](https://github.com/LionelJouin/multi-network/tree/init-framework)

Other PoCs:
* [aojea/kubernetes-network-driver](https://github.com/aojea/kubernetes-network-driver)
    * [Slides](https://docs.google.com/presentation/d/1Vdr7BhbYXeWjwmLjGmqnUkvJr_eOUdU0x-JxfXWxUT8/edit#slide=id.p)
    * [Recording](https://www.youtube.com/watch?v=XEfaBtEDWDU)
* [Containerd as DRA-Driver + ResourceClaim Network Status](https://gist.github.com/LionelJouin/5cfc11eecf73663b5657ed3be1eb6c00)

## Build

Generate Code (Proto, API, ...)
```
make generate
```

build/push (default registry: localhost:5000/network-dra)
```
make REGISTRY=localhost:5000/network-dra
```

Clone Kubernetes
```
git clone git@github.com:kubernetes/kubernetes.git
cd kubernetes
git remote add LionelJouin git@github.com:LionelJouin/kubernetes.git
git fetch LionelJouin
git checkout LionelJouin/KEP-4817
```

Build Kubernetes
```
kind build node-image . --image kindest/node:kep-4817
```

## Demo

Create Kind Cluster
```
kind create cluster --config examples/kind.yaml
```

Load Images in Kind
```
kind load docker-image localhost:5000/network-dra/network-nri-plugin:latest
```

Install CNI Plugins
```
kubectl apply -f https://raw.githubusercontent.com/k8snetworkplumbingwg/multus-cni/master/e2e/templates/cni-install.yml.j2
```

Install DRA Plugin
```
helm install network-dra deployments/network-DRA --set registry=localhost:5000/network-dra
```

Demo
```
kubectl apply -f examples/demo-a.yaml
```

- Demo A
    - Single Pod with a single resource claim.
    - The pod will receive the interface described in the `macvlan-eth0-attachment` resource claim parameter.

## Flow

![Flow](docs/resources/Diagrams-Call-Flow-NRI.png)

1. Kubelet calls the NodePrepareResources via the DRA API.
    * The NRI-Plugin is also the DRA-Driver, so it gets the call.
2. The full ResourceClaims are retrieved from the Kubernetes API.
    * The ResourceClaims are stored for the pod in the reservedFor field (Pod requesting this claim stored in the ResourceClaim allocation status).
3. Kubelet creates the pod.
    * Kubelet calls RunPodSanbox to the Container runtime.
3. At the end of RunPodSanbox, the container runtime calls RunPodSanbox([nri.PodSandbox](https://github.com/containerd/nri/blob/v0.6.1/pkg/api/api.proto#L213)) for each NRI Plugin.
    * The pod Name, pod Namespace, network namespace are retrieved.
5. The NRI plugin retrieves the previously stored ResourceClaims for the pod passed to RunPodSanbox.
    * CNI Add is called based on the CNI config stored in the ResourceClaims.
6. The Kubernetes API is used to update the ResourceClaims Devices Status with the CNI result.

## Result

Object applied: [./examples/demo-a.yaml](examples/demo-a.yaml)

Final ResourceClaim object:
```yaml
apiVersion: resource.k8s.io/v1alpha3
kind: ResourceClaim
metadata:
  name: macvlan-eth0-attachment
spec:
  devices:
    config:
    - opaque:
        driver: poc.dra.networking
        parameters:
          config:
            cniVersion: 1.0.0
            name: macvlan-eth0
            plugins:
            - ipam:
                ranges:
                - - subnet: 10.10.1.0/24
                type: host-local
              master: eth0
              mode: bridge
              type: macvlan
          interface: net1
      requests:
      - macvlan-eth0
    requests:
    - allocationMode: ExactCount
      count: 1
      deviceClassName: network-interface
      name: macvlan-eth0
status:
  allocation:
    devices:
      config:
      - opaque:
          driver: poc.dra.networking
          parameters:
            config:
              cniVersion: 1.0.0
              name: macvlan-eth0
              plugins:
              - ipam:
                  ranges:
                  - - subnet: 10.10.1.0/24
                  type: host-local
                master: eth0
                mode: bridge
                type: macvlan
            interface: net1
        requests:
        - macvlan-eth0
        source: FromClaim
      results:
      - device: cni
        driver: poc.dra.networking
        pool: kind-worker
        request: macvlan-eth0
    nodeSelector:
      nodeSelectorTerms:
      - matchFields:
        - key: metadata.name
          operator: In
          values:
          - kind-worker
  devices:
  - conditions: null
    data:
    - cniVersion: 1.0.0
      interfaces:
      - mac: b2:af:6a:f9:12:3b
        name: net1
        sandbox: /var/run/netns/cni-d36910c7-c9a4-78f6-abad-26e9a8142a04
      ips:
      - address: 10.10.1.2/24
        gateway: 10.10.1.1
        interface: 0
    device: cni
    driver: poc.dra.networking
    networkData:
      addresses:
      - cidr: 10.10.1.2/24
      hwAddress: b2:af:6a:f9:12:3b
      interfaceName: net1
    pool: kind-worker
  reservedFor:
  - name: demo-a
    resource: pods
    uid: 680f0a77-8d0b-4e21-8599-62581e335ed6
```

## Resources

- MN KEP: https://github.com/kubernetes/enhancements/pull/3700
- MN Sync: https://docs.google.com/document/d/1pe_0aOsI35BEsQJ-FhFH9Z_pWQcU2uqwAnOx2NIx6OY/edit#heading=h.fo1yo94x96wg
- DRA KEP: https://github.com/kubernetes/enhancements/blob/master/keps/sig-node/3063-dynamic-resource-allocation/README.md
- DRA API: https://github.com/kubernetes/kubernetes/blob/v1.30.0/staging/src/k8s.io/kubelet/pkg/apis/dra/v1alpha3/api.proto#L34
- DRA Controller: https://pkg.go.dev/k8s.io/dynamic-resource-allocation/controller
- NRI: https://github.com/containerd/nri
- NRI in Containerd: https://github.com/containerd/containerd/blob/v2.0.0-rc.2/docs/NRI.md
- Network Device Injector NRI Plugin PR: https://github.com/containerd/nri/pull/82
- NRI Network PR: https://github.com/containerd/nri/pull/57
- KEP-4817: https://github.com/kubernetes/enhancements/issues/4817

# Network DRA

```
helm install network-dra deployments/network-DRA
kubectl apply -f examples/pod.yaml
```

```
kubectl delete -f examples/pod.yaml ; helm delete network-dra
```
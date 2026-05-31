# Kubernetes setup

GMD Typesense deployment in the `gmd` namespace, pinned to node `nitrogen`.

## Operators

- [Typesense Operator](https://github.com/akyriako/typesense-operator) — manages `TypesenseCluster` CRDs
- [CloudNativePG](https://cloudnative-pg.io/) — PostgreSQL operator (available if needed)

## Typesense cluster

| Resource | Detail |
|---|---|
| CRD | `TypesenseCluster` (`ts.opentelekomcloud.com/v1alpha1`) |
| Name | `gmd-ts` |
| API port | 8108 |
| Health port | 8808 |
| ClusterIP | `gmd-ts-svc` (8108, 8808) |
| NodePort | `gmd-ts-nodeport` → 30336 (8108), 32402 (8808) |
| Health check | `curl 192.168.4.26:30336/health` → `{"ok":true}` |

## Files

```
k8s/
└── typesense.yaml   # TypesenseCluster + NodePort Service
```

## Apply

```sh
kubectl apply -f k8s/typesense.yaml
```

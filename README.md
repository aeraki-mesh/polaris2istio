# polaris2istio

Polaris2istio watches Polaris registry and synchronize the Polaris services which match the rules to Istio.

![ polaris2istio ](doc/polaris2istio.png)
### Usage  
#### Build

```bash
make build
```

#### Run

```bash
polaris2istio --polarisAddress <polarishost:port>
```

#### Config
##### Method 1. Sync polaris service base on ServiceEntry:

```yaml
apiVersion: networking.istio.io/v1alpha3
kind: ServiceEntry
metadata:
  name: <polaris-name-for-k8s>
  namespace: polaris
  annotations:
    aeraki.net/polarisNamespace: Test
    aeraki.net/polarisService: test-service
    aeraki.net/external: "false"
  labels:
    manager: aeraki
    registry: polaris
spec:
  hosts:
    - dev.<polaris-name-for-k8s>.polaris
  resolution: NONE # or STATIC
```

We just watch the ServiceEntrys in the polaris namespae.

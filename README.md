# kontainer-engine-driver-oke-capi

For managing the lifecycle of OKE clusters on OCI compute using CAPI.

### How to create a new driver release

1. Run the `make shasum` target to create a release binary corresponding checksum.
2. Create a new github release of this repo.
3. Upload the kontainer driver binary from the `dist` directory to the github release, adding the checksum.

### How to install a dev driver version for testing

1. Run `make build` to create a dev build in the `dist` directory
2. Upload the dev build to OCI Object Storage, or the file server of your choice
3. Create a `kontainerdriver` resource on your cluster that references the dev build location (must be reachable by your cluster)

```yaml
apiVersion: management.cattle.io/v3
kind: KontainerDriver
metadata:
  name: okecapi
spec:
  active: true
  builtIn: false
  checksum: ""
  uiUrl: ""
  url: <DRIVER URI>/kontainer-engine-driver-okecapi-linux
```

After applying the `kontainerdriver` to your cluster, it will be downloaded and installed, after which it is ready for use.

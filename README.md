# certificate-dns-bridge

Based on https://github.com/jetstack/cert-manager-webhook-example


## What it does

The certificate-dns-bridge is a small kubernetes API service that connects the [cert-manager](https://github.com/jetstack/cert-manager) to the [DNS-controller-manager](https://github.com/gardener/external-dns-management) using the cert-manager's webhook interface. This way, the DNS-controller-manager can set the TXT records that are required for the ACME `DNS01` challenge.


## How to use it

This guide assumes that the [cert-manager](https://github.com/jetstack/cert-manager) (at least version v0.8.0) as well as the [dns-controller-manager](https://github.com/gardener/external-dns-management) (at least version 0.5.0) are already deployed in the cluster.


### Configure the Solver (certificate-dns-bridge)

These are important parameters of the `values.yaml` file you should have a look at:

- `groupName` and `solverName`: Together these two form a unique identifier for your solver deployment. `groupName` should be a unique API group name (the creators of cert-manager advise to set it to your company's domain). `solverName` is expected to be unique within all solver deployments that share a `groupName` and can be used to differentiate between them. 

- `certManager.namespace`: The namespace in which cert-manager is deployed.

- `certManager.serviceAccountName`: Name of the service account that is used by cert-manager.

- `verbose`: You can set this value to change the verbosity level of the solver logs. If not specified, it defaults to `2`.

The solver is actually a small API service. It gets a `POST` request from the cert-manager when a TXT record (for the `DNS01` validation) needs to be created, extracts the necessary information from the JSON object that comes with the request and creates a `DNSEntry` object out of it. This object is picked up by the dns-controller-manager, which creates the TXT record. Once the existence of the TXT record has been validated, cert-manager sends another request to the solver, indicating that the record is no longer needed. The solver deletes the corresponding `DNSEntry` object, which in turn triggers the dns-controller-manager to delete the TXT record. 


### Configure the Issuer

To connect the cert-manager to the solver, a specifically configured `Issuer` (or `ClusterIssuer`) is needed. It should look something like this:

```yaml
apiVersion: certmanager.k8s.io/v1alpha1
kind: ClusterIssuer
metadata:
  name: my-issuer
spec:
  acme:
    server: https://acme-v02.api.letsencrypt.org/directory
    email: johndoe@example.com
    privateKeySecretRef:
      name: my-issuer-secret
    solvers:
      - dns01:
          webhook:
            groupName: cert.gardener.cloud
            solverName: certificate-dns-bridge
            config:
              dns-class: my-dns-class
              namespace: kube-system
              ttl: 300
```

Only the `webhook` part of the manifest is explained here, see the [official cert-manager documentation](https://docs.cert-manager.io) for more information on issuers (and specifically [ACME issuers](https://docs.cert-manager.io/en/master/tasks/issuers/setup-acme/index.html)). 

- `groupName` and `solverName`: The values here must match the corresponding values from the solver helm chart, so cert-manager knows where to send the `POST` request to.
- `config`: Here, some solver-specific configuration can be provided.
  - `dns-class`: Which DNS class to use for the `DNSEntry` object. This value is only needed if you run multiple instances of the dns-controller-manager within one cluster - they need different DNS classes to avoid conflicts.
  - `namespace`: Which namespace the `DNSEntry` should be created in. Unless your deployment of the dns-controller-manager is configured to watch all namespaces, you should put its namespace here. If not specified, the namespace of the issuer will be used.
  - `ttl`: The TTL for the TXT record. Defaults to `120` if not specified.
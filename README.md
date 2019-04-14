# certificate-dns-bridge

Based on https://github.com/jetstack/cert-manager-webhook-example

## What it does

The certificate-dns-bridge is a small kubernetes API service that connects the [cert-manager](https://github.com/jetstack/cert-manager) to the [DNS-controller-manager](https://github.com/gardener/external-dns-management) using the cert-manager's webhook interface. This way, the DNS-controller-manager can set the TXT records that are required for the ACME `DNS01` challenge.
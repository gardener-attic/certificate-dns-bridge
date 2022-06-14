module github.com/gardener/certificate-dns-bridge

go 1.18

require (
	github.com/gardener/external-dns-management v0.7.13
	github.com/jetstack/cert-manager v0.12.0
	k8s.io/apiextensions-apiserver v0.17.6
	k8s.io/apimachinery v0.17.6
	k8s.io/client-go v0.17.6
	k8s.io/klog v1.0.0
)

replace github.com/evanphx/json-patch => github.com/evanphx/json-patch v0.0.0-20190203023257-5858425f7550

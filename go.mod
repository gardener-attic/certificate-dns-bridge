module github.com/gardener/certificate-dns-bridge

go 1.12

require (
	github.com/dgrijalva/jwt-go v3.2.0+incompatible // indirect
	github.com/gardener/controller-manager-library v0.0.0-20190508145811-670efa3cd76c // indirect
	github.com/gardener/external-dns-management v0.0.0-20190508150408-ef73be2d81c0
	github.com/imdario/mergo v0.3.7 // indirect
	github.com/jetstack/cert-manager v0.8.0
	k8s.io/apiextensions-apiserver v0.0.0-20190413053546-d0acb7a76918
	k8s.io/apimachinery v0.0.0-20190509063443-7d8f8feb49c5
	k8s.io/client-go v11.0.0+incompatible
	k8s.io/klog v0.3.0
)

replace k8s.io/client-go => k8s.io/client-go v0.0.0-20190413052642-108c485f896e

replace github.com/evanphx/json-patch => github.com/evanphx/json-patch v0.0.0-20190203023257-5858425f7550

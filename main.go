// Copyright 2019 Copyright (c) 2019 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"encoding/json"
	"fmt"
	"hash/crc32"
	"os"
	"strings"

	extapi "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	// "k8s.io/client-go/kubernetes"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/rest"
	log "k8s.io/klog"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	dnsapi "github.com/gardener/external-dns-management/pkg/apis/dns/v1alpha1"
	dnsclient "github.com/gardener/external-dns-management/pkg/client/dns/clientset/versioned"
	"github.com/jetstack/cert-manager/pkg/acme/webhook/apis/acme/v1alpha1"
	"github.com/jetstack/cert-manager/pkg/acme/webhook/cmd"
)

var GroupName = os.Getenv("GROUP_NAME")
var SolverName = os.Getenv("SOLVER_NAME")

func main() {
	if GroupName == "" {
		log.Fatal("GROUP_NAME must be specified")
	}
	if SolverName == "" {
		log.Fatal("SOLVER_NAME must be specified")
	}

	// This will register our custom DNS provider with the webhook serving
	// library, making it available as an API under the provided GroupName.
	// You can register multiple DNS provider implementations with a single
	// webhook, where the Name() method will be used to disambiguate between
	// the different implementations.
	cmd.RunWebhookServer(GroupName,
		&customDNSProviderSolver{},
	)
}

// customDNSProviderSolver implements the provider-specific logic needed to
// 'present' an ACME challenge TXT record for your own DNS provider.
// To do so, it must implement the `github.com/jetstack/cert-manager/pkg/acme/webhook.Solver`
// interface.
type customDNSProviderSolver struct {
	// If a Kubernetes 'clientset' is needed, you must:
	// 1. uncomment the additional `client` field in this structure below
	// 2. uncomment the "k8s.io/client-go/kubernetes" import at the top of the file
	// 3. uncomment the relevant code in the Initialize method below
	// 4. ensure your webhook's service account has the required RBAC role
	//    assigned to it for interacting with the Kubernetes APIs you need.
	// client kubernetes.Clientset
	dclient dnsclient.Clientset
}

// customDNSProviderConfig is a structure that is used to decode into when
// solving a DNS01 challenge.
// This information is provided by cert-manager, and may be a reference to
// additional configuration that's needed to solve the challenge for this
// particular certificate or issuer.
// This typically includes references to Secret resources containing DNS
// provider credentials, in cases where a 'multi-tenant' DNS solver is being
// created.
// If you do *not* require per-issuer or per-certificate configuration to be
// provided to your webhook, you can skip decoding altogether in favour of
// using CLI flags or similar to provide configuration.
// You should not include sensitive information here. If credentials need to
// be used by your provider here, you should reference a Kubernetes Secret
// resource and fetch these credentials using a Kubernetes clientset.
type customDNSProviderConfig struct {
	// Change the two fields below according to the format of the configuration
	// to be decoded.
	// These fields will be set by users in the
	// `issuer.spec.acme.dns01.providers.webhook.config` field.

	//Email           string `json:"email"`
	//APIKeySecretRef v1alpha1.SecretKeySelector `json:"apiKeySecretRef"`
	DNSClass  string `json:"dns-class"`
	TTL       int    `json:"ttl"`
	Namespace string `json:"namespace"`
}

// Name is used as the name for this DNS solver when referencing it on the ACME
// Issuer resource.
// This should be unique **within the group name**, i.e. you can have two
// solvers configured with the same Name() **so long as they do not co-exist
// within a single webhook deployment**.
// For example, `cloudflare` may be used as the name of a solver.
func (c *customDNSProviderSolver) Name() string {
	return SolverName
}

// Present is responsible for actually presenting the DNS record with the
// DNS provider.
// This method should tolerate being called multiple times with the same value.
// cert-manager itself will later perform a self check to ensure that the
// solver has correctly configured the DNS provider.
func (c *customDNSProviderSolver) Present(ch *v1alpha1.ChallengeRequest) error {
	// compute name based on hash of acme challenge domain and key
	name := computeDNSEntryName(ch)
	log.V(2).Infof("CHALLENGE received - %s", name)
	log.V(3).Infof("challenge [%s|-] - set TXT record at '%s' to '%s'", name, ch.ResolvedFQDN, ch.Key)

	cfg, err := loadConfig(ch.Config)
	if err != nil {
		log.Errorf("challenge [%s|-] - error decoding solver config: %v", name, err.Error())
		return err
	}

	var namespace string
	if cfg.Namespace != "" {
		namespace = cfg.Namespace
		log.V(4).Infof("challenge [%s|%s] - issuer configuration: namespace=%s", name, namespace, namespace)
	} else {
		namespace = ch.ResourceNamespace
	}

	// set configuration, if specified
	ann := map[string]string{}
	if cfg.DNSClass != "" {
		log.V(4).Infof("challenge [%s|%s] - issuer configuration: dns-class=%s", name, namespace, cfg.DNSClass)
		ann["dns.gardener.cloud/class"] = cfg.DNSClass
	}
	ttl := int64(120)
	if cfg.TTL > 0 {
		log.V(4).Infof("challenge [%s|%s] - issuer configuration: ttl=%d", name, namespace, cfg.TTL)
		ttl = int64(cfg.TTL)
	}
	// create DNSEntry object
	dnse := dnsapi.DNSEntry{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   namespace,
			Annotations: ann,
		},
		Spec: dnsapi.DNSEntrySpec{
			TTL:     &ttl,
			DNSName: strings.TrimSuffix(ch.ResolvedFQDN, "."),
			Text:    []string{ch.Key},
		},
	}

	log.V(3).Infof("challenge [%s|%s] - creating TXT record for %s", name, namespace, dnse.Spec.DNSName)
	_, err2 := c.dclient.KracV1alpha1().DNSEntries(namespace).Create(&dnse)
	if err2 == nil {
		log.V(2).Infof("challenge [%s|%s] - DNSEntry for '%s' created", name, namespace, dnse.Spec.DNSName)
	} else {
		if errors.IsAlreadyExists(err2) {
			log.V(3).Infof("challenge [%s|%s] - DNSEntry for '%s' seems to exist, updating it", name, namespace, dnse.Spec.DNSName)
			_, err3 := c.dclient.KracV1alpha1().DNSEntries(namespace).Update(&dnse)
			if err3 == nil {
				log.V(3).Infof("challenge [%s|%s] - updated DNSEntry for '%s' updated", name, namespace, dnse.Spec.DNSName)
			} else {
				log.Errorf("challenge [%s|%s] - DNSEntry seems to exist but cannot be updated: %v", name, namespace, err3.Error())
				return err3
			}
		} else {
			log.Errorf("challenge [%s|%s] - DNSEntry cannot be created: %v", name, namespace, err2.Error())
			return err2
		}
	}

	return nil
}

// CleanUp should delete the relevant TXT record from the DNS provider console.
// If multiple TXT records exist with the same record name (e.g.
// _acme-challenge.example.com) then **only** the record with the same `key`
// value provided on the ChallengeRequest should be cleaned up.
// This is in order to facilitate multiple DNS validations for the same domain
// concurrently.
func (c *customDNSProviderSolver) CleanUp(ch *v1alpha1.ChallengeRequest) error {
	name := computeDNSEntryName(ch)
	log.V(2).Infof("CLEANUP received - %s", name)

	cfg, err := loadConfig(ch.Config)
	if err != nil {
		log.Errorf("cleanup [%s|-] - error decoding solver config: %v", name, err.Error())
		return err
	}

	var namespace string
	if cfg.Namespace != "" {
		namespace = cfg.Namespace
		log.V(4).Infof("cleanup [%s|%s] - issuer configuration: namespace=%s", name, namespace, namespace)
	} else {
		namespace = ch.ResourceNamespace
	}

	log.V(3).Infof("cleanup [%s|%s] - deleting DNSEntry", name, namespace)
	err = c.dclient.KracV1alpha1().DNSEntries(namespace).Delete(name, &metav1.DeleteOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			log.Warningf("cleanup [%s|%s] - tried to delete DNSEntry, but it didn't exist", name, namespace)
			return nil
		} else {
			log.Errorf("cleanup [%s|%s] - unable to delete DNSEntry: %v", name, namespace, err)
			return err
		}
	}
	log.V(2).Infof("cleanup [%s|%s] - deleted DNSEntry", name, namespace)

	return nil
}

// Initialize will be called when the webhook first starts.
// This method can be used to instantiate the webhook, i.e. initialising
// connections or warming up caches.
// Typically, the kubeClientConfig parameter is used to build a Kubernetes
// client that can be used to fetch resources from the Kubernetes API, e.g.
// Secret resources containing credentials used to authenticate with DNS
// provider accounts.
// The stopCh can be used to handle early termination of the webhook, in cases
// where a SIGTERM or similar signal is sent to the webhook process.
func (c *customDNSProviderSolver) Initialize(kubeClientConfig *rest.Config, stopCh <-chan struct{}) error {
	dnscl, err := dnsclient.NewForConfig(kubeClientConfig)
	if err != nil {
		log.Fatalf("error building dns-controller clientset: %s", err.Error())
	}
	c.dclient = *dnscl

	for i := 9; i >= 0; i-- {
		if log.V(log.Level(i)) {
			log.V(0).Infof("logging with verbosity %d", i)
			break
		}
	}

	log.V(1).Info("successfully initialized")

	return nil
}

// loadConfig is a small helper function that decodes JSON configuration into
// the typed config struct.
func loadConfig(cfgJSON *extapi.JSON) (customDNSProviderConfig, error) {
	cfg := customDNSProviderConfig{}
	// handle the 'base case' where no configuration has been provided
	if cfgJSON == nil {
		return cfg, nil
	}
	if err := json.Unmarshal(cfgJSON.Raw, &cfg); err != nil {
		return cfg, err
	}

	return cfg, nil
}

func computeDNSEntryName(ch *v1alpha1.ChallengeRequest) string {
	return fmt.Sprintf("acme-challenge-%d", hash(ch.ResolvedFQDN+ch.Key))
}

func hash(s string) uint32 {
	h := crc32.New(crc32.MakeTable(0xD5828281))
	h.Write([]byte(s))
	return h.Sum32()
}

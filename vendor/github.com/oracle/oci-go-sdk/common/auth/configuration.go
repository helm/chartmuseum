// Copyright (c) 2016, 2018, Oracle and/or its affiliates. All rights reserved.

package auth

import (
	"crypto/rsa"
	"fmt"
	"github.com/oracle/oci-go-sdk/common"
)

type instancePrincipalConfigurationProvider struct {
	keyProvider instancePrincipalKeyProvider
	region      *common.Region
}

//InstancePrincipalConfigurationProvider returns a configuration for instance principals
func InstancePrincipalConfigurationProvider() (common.ConfigurationProvider, error) {
	var err error
	var keyProvider *instancePrincipalKeyProvider
	if keyProvider, err = newInstancePrincipalKeyProvider(); err != nil {
		return nil, fmt.Errorf("failed to create a new key provider for instance principal: %s", err.Error())
	}
	return instancePrincipalConfigurationProvider{keyProvider: *keyProvider, region: nil}, nil
}

//InstancePrincipalConfigurationProviderForRegion returns a configuration for instance principals with a given region
func InstancePrincipalConfigurationProviderForRegion(region common.Region) (common.ConfigurationProvider, error) {
	var err error
	var keyProvider *instancePrincipalKeyProvider
	if keyProvider, err = newInstancePrincipalKeyProvider(); err != nil {
		return nil, fmt.Errorf("failed to create a new key provider for instance principal: %s", err.Error())
	}
	return instancePrincipalConfigurationProvider{keyProvider: *keyProvider, region: &region}, nil
}

//InstancePrincipalConfigurationWithCerts returns a configuration for instance principals with a given region and hardcoded certificates in lieu of metadata service certs
func InstancePrincipalConfigurationWithCerts(region common.Region, leafCertificate, leafPassphrase, leafPrivateKey []byte, intermediateCertificates [][]byte) (common.ConfigurationProvider, error) {
	leafCertificateRetriever := staticCertificateRetriever{Passphrase: leafPassphrase, CertificatePem: leafCertificate, PrivateKeyPem: leafPrivateKey}

	//The .Refresh() call actually reads the certificates from the inputs
	err := leafCertificateRetriever.Refresh()
	if err != nil {
		return nil, err
	}

	certificate := leafCertificateRetriever.Certificate()

	tenancyID := extractTenancyIDFromCertificate(certificate)
	fedClient, err := newX509FederationClientWithCerts(region, tenancyID, leafCertificate, leafPassphrase, leafPrivateKey, intermediateCertificates)
	if err != nil {
		return nil, err
	}

	provider := instancePrincipalConfigurationProvider{
		keyProvider: instancePrincipalKeyProvider{
			Region:           region,
			FederationClient: fedClient,
			TenancyID:        tenancyID,
		},
		region: &region,
	}
	return provider, nil

}

func (p instancePrincipalConfigurationProvider) PrivateRSAKey() (*rsa.PrivateKey, error) {
	return p.keyProvider.PrivateRSAKey()
}

func (p instancePrincipalConfigurationProvider) KeyID() (string, error) {
	return p.keyProvider.KeyID()
}

func (p instancePrincipalConfigurationProvider) TenancyOCID() (string, error) {
	return p.keyProvider.TenancyOCID()
}

func (p instancePrincipalConfigurationProvider) UserOCID() (string, error) {
	return "", nil
}

func (p instancePrincipalConfigurationProvider) KeyFingerprint() (string, error) {
	return "", nil
}

func (p instancePrincipalConfigurationProvider) Region() (string, error) {
	if p.region == nil {
		return string(p.keyProvider.RegionForFederationClient()), nil
	}
	return string(*p.region), nil
}

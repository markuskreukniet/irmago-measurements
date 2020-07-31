package irmaclient

import (
	irma "github.com/markuskreukniet/irmago-measurements"
	"github.com/privacybydesign/gabi"
)

// credential represents an IRMA credential, whose zeroth attribute
// is always the secret key and the first attribute the metadata attribute.
type credential struct {
	*gabi.Credential
	*irma.MetadataAttribute
	attrs *irma.AttributeList
}

func newCredential(gabicred *gabi.Credential, attrs *irma.AttributeList, conf *irma.Configuration) (*credential, error) {
	meta := irma.MetadataFromInt(gabicred.Attributes[1], conf)
	cred := &credential{
		Credential:        gabicred,
		MetadataAttribute: meta,
	}

	if cred.CredentialType() == nil {
		// Unknown credtype, populate Pk field later
		return cred, nil
	}

	var err error
	cred.Pk, err = conf.PublicKey(meta.CredentialType().IssuerIdentifier(), cred.KeyCounter())
	if err != nil {
		return nil, err
	}
	cred.attrs = attrs
	return cred, nil
}

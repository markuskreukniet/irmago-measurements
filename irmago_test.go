package irma

import (
	"crypto/rand"
	"encoding/json"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/markuskreukniet/irmago-measurements/internal/common"
	"github.com/markuskreukniet/irmago-measurements/internal/test"
	"github.com/privacybydesign/gabi"
	"github.com/privacybydesign/gabi/big"
	"github.com/privacybydesign/gabi/revocation"
	"github.com/stretchr/testify/require"
)

func init() {
	common.ForceHTTPS = false // globally disable https enforcement
}

func parseConfiguration(t *testing.T) *Configuration {
	conf, err := NewConfiguration("testdata/irma_configuration", ConfigurationOptions{})
	require.NoError(t, err)
	require.NoError(t, conf.ParseFolder())
	return conf
}

// A convenience function for initializing big integers from known correct (10
// base) strings. Use with care, errors are ignored.
func s2big(s string) (r *big.Int) {
	r, _ = new(big.Int).SetString(s, 10)
	return
}

func TestConfigurationAutocopy(t *testing.T) {
	storage := test.CreateTestStorage(t)
	defer test.ClearTestStorage(t, storage)

	path := filepath.Join("testdata", "tmp", "client", "irma_configuration")
	require.NoError(t, common.CopyDirectory(filepath.Join("testdata", "irma_configuration"), path))
	conf, err := NewConfiguration(path, ConfigurationOptions{Assets: filepath.Join("testdata", "irma_configuration_updated")})
	require.NoError(t, err)
	require.NoError(t, conf.ParseFolder())

	credid := NewCredentialTypeIdentifier("irma-demo.RU.studentCard")
	attrid := NewAttributeTypeIdentifier("irma-demo.RU.studentCard.newAttribute")
	require.True(t, conf.CredentialTypes[credid].ContainsAttribute(attrid))
}

func TestParseInvalidIrmaConfiguration(t *testing.T) {
	// The description.xml of the scheme manager under this folder has been edited
	// to invalidate the scheme manager signature
	conf, err := NewConfiguration(filepath.Join("testdata", "irma_configuration_invalid"), ConfigurationOptions{ReadOnly: true})
	require.NoError(t, err)

	// Parsing it should return a SchemeManagerError
	err = conf.ParseFolder()
	require.Error(t, err)
	smerr, ok := err.(*SchemeManagerError)
	require.True(t, ok)
	require.Equal(t, SchemeManagerStatusInvalidSignature, smerr.Status)

	// The manager should still be in conf.SchemeManagers, but also in DisabledSchemeManagers
	require.Contains(t, conf.SchemeManagers, smerr.Manager)
	require.Contains(t, conf.DisabledSchemeManagers, smerr.Manager)
	require.Equal(t, SchemeManagerStatusInvalidSignature, conf.SchemeManagers[smerr.Manager].Status)
	require.Equal(t, false, conf.SchemeManagers[smerr.Manager].Valid)
}

func TestRetryHTTPRequest(t *testing.T) {
	test.StartBadHttpServer(2, 1*time.Second, "42")
	defer test.StopBadHttpServer()

	transport := NewHTTPTransport("http://localhost:48682", false)
	transport.client.HTTPClient.Timeout = 500 * time.Millisecond
	bts, err := transport.GetBytes("")
	require.NoError(t, err)
	require.Equal(t, "42\n", string(bts))
}

func TestInvalidIrmaConfigurationRestoreFromRemote(t *testing.T) {
	test.StartSchemeManagerHttpServer()
	defer test.StopSchemeManagerHttpServer()

	storage := test.CreateTestStorage(t)
	defer test.ClearTestStorage(t, storage)

	conf, err := NewConfiguration(filepath.Join("testdata", "tmp", "client", "irma_configuration"), ConfigurationOptions{
		Assets: filepath.Join("testdata", "irma_configuration_invalid"),
	})
	require.NoError(t, err)

	err = conf.ParseOrRestoreFolder()
	require.NoError(t, err)
	require.Empty(t, conf.DisabledSchemeManagers)
	require.Contains(t, conf.SchemeManagers, NewSchemeManagerIdentifier("irma-demo"))
	require.Contains(t, conf.CredentialTypes, NewCredentialTypeIdentifier("irma-demo.RU.studentCard"))
}

func TestInvalidIrmaConfigurationRestoreFromAssets(t *testing.T) {
	storage := test.CreateTestStorage(t)
	defer test.ClearTestStorage(t, storage)

	conf, err := NewConfiguration(filepath.Join(storage, "client", "irma_configuration"), ConfigurationOptions{
		Assets: filepath.Join("testdata", "irma_configuration_invalid"),
	})
	require.NoError(t, err)

	// Fails: no remote and the version in the assets is broken
	err = conf.ParseOrRestoreFolder()
	require.Error(t, err)
	require.NotEmpty(t, conf.DisabledSchemeManagers)

	// Try again from correct assets
	conf.assets = filepath.Join("testdata", "irma_configuration")
	err = conf.ParseOrRestoreFolder()
	require.NoError(t, err)
	require.Empty(t, conf.DisabledSchemeManagers)
	require.Contains(t, conf.SchemeManagers, NewSchemeManagerIdentifier("irma-demo"))
	require.Contains(t, conf.CredentialTypes, NewCredentialTypeIdentifier("irma-demo.RU.studentCard"))
}

func TestParseIrmaConfiguration(t *testing.T) {
	conf := parseConfiguration(t)

	require.Contains(t, conf.SchemeManagers, NewSchemeManagerIdentifier("irma-demo"))
	require.Contains(t, conf.SchemeManagers, NewSchemeManagerIdentifier("test"))

	pk, err := conf.PublicKey(NewIssuerIdentifier("irma-demo.RU"), 0)
	require.NoError(t, err)
	require.NotNil(t, pk)
	require.NotNil(t, pk.N, "irma-demo.RU public key has no modulus")
	require.Equal(t,
		"Irma Demo",
		conf.SchemeManagers[NewSchemeManagerIdentifier("irma-demo")].Name["en"],
		"irma-demo scheme manager has unexpected name")
	require.Equal(t,
		"Demo Radboud University Nijmegen",
		conf.Issuers[NewIssuerIdentifier("irma-demo.RU")].Name["en"],
		"irma-demo.RU issuer has unexpected name")
	require.Equal(t,
		"Student Card",
		conf.CredentialTypes[NewCredentialTypeIdentifier("irma-demo.RU.studentCard")].ShortName["en"],
		"irma-demo.RU.studentCard has unexpected name")

	require.Equal(t,
		"studentID",
		conf.CredentialTypes[NewCredentialTypeIdentifier("irma-demo.RU.studentCard")].AttributeTypes[2].ID,
		"irma-demo.RU.studentCard.studentID has unexpected name")

	// Hash algorithm pseudocode:
	// Base64(SHA256("irma-demo.RU.studentCard")[0:16])
	//require.Contains(t, conf.reverseHashes, "1stqlPad5edpfS1Na1U+DA==",
	//	"irma-demo.RU.studentCard had improper hash")
	//require.Contains(t, conf.reverseHashes, "CLjnADMBYlFcuGOT7Z0xRg==",
	//	"irma-demo.MijnOverheid.root had improper hash")
}

func TestMetadataAttribute(t *testing.T) {
	metadata := NewMetadataAttribute(0x02)
	if metadata.Version() != 0x02 {
		t.Errorf("Unexpected metadata version: %d", metadata.Version())
	}

	expiry := metadata.SigningDate().Unix() + int64(metadata.ValidityDuration()*ExpiryFactor)
	if !time.Unix(expiry, 0).Equal(metadata.Expiry()) {
		t.Errorf("Invalid signing date")
	}

	if metadata.KeyCounter() != 0 {
		t.Errorf("Unexpected key counter")
	}
}

func TestMetadataCompatibility(t *testing.T) {
	conf, err := NewConfiguration(filepath.Join("testdata", "irma_configuration"), ConfigurationOptions{ReadOnly: true})
	require.NoError(t, err)
	require.NoError(t, conf.ParseFolder())

	// An actual metadata attribute of an IRMA credential extracted from the IRMA app
	attr := MetadataFromInt(s2big("49043481832371145193140299771658227036446546573739245068"), conf)
	require.NotNil(t, attr.CredentialType(), "attr.CredentialType() should not be nil")

	require.Equal(t,
		NewCredentialTypeIdentifier("irma-demo.RU.studentCard"),
		attr.CredentialType().Identifier(),
		"Metadata credential type was not irma-demo.RU.studentCard",
	)
	require.Equal(t, byte(0x02), attr.Version(), "Unexpected metadata version")
	require.Equal(t, time.Unix(1499904000, 0), attr.SigningDate(), "Unexpected signing date")
	require.Equal(t, time.Unix(1516233600, 0), attr.Expiry(), "Unexpected expiry date")
	require.Equal(t, uint(2), attr.KeyCounter(), "Unexpected key counter")
}

func TestTimestamp(t *testing.T) {
	mytime := Timestamp(time.Unix(1500000000, 0))
	timestruct := struct{ Time *Timestamp }{Time: &mytime}
	bytes, err := json.Marshal(timestruct)
	require.NoError(t, err)

	timestruct = struct{ Time *Timestamp }{}
	require.NoError(t, json.Unmarshal(bytes, &timestruct))
	require.Equal(t, time.Time(*timestruct.Time).Unix(), int64(1500000000))
}

func TestVerifyValidSig(t *testing.T) {
	conf := parseConfiguration(t)

	irmaSignedMessageJson := "{\"signature\":[{\"c\":\"pliyrSE7wXcDcKXuBtZW5bnucvBSXpILIRvnNBgx7hQ=\",\"A\":\"D/8wLPq9860bpXZ5c+VYyoPJ+Z8CWDZNQ0jXvst8qnPRdivy/GQIfJHjVnpOPlHbguphb/7JVbfcV3bZeybA3bCF/4UesjRUZlMf/iJ/QgKHbt41ogN1PPT5z7qBJpkxuNTIkHxaUPoDvhouHmuC9pNj4afRUyLJerxKPkpdBw0=\",\"e_response\":\"YOrKTrMSs4/QOUtPkT0YaYNEmW7Cs+cu624zr2xrHodyL88ub6yaXB7MGHAcQ1+iXsGN8jkfxB/0\",\"v_response\":\"AYSa1p8ISs//MsocJjODwWuPB/z6+iKHHi+sTToRs0eJ2X1gwmWoA5QB0aHjRkWye3/+2rtosfUzI77FlPQVnrbMERwcuYM/fx3fpNCpjm2qcs3AOJRcSRxcNFMe1+4ECsmJhByMDutS1KXAAKiNvnhEXx9f0JrQGwQFtpSFPh8dOuvEKUZHAUALr4FcHCa2HL9nDRiqy2KAOxE0nAANAcMaBo/ed+WZeHtv4CTB7egyYs27cklVbwlBzmRrbjNZk57ICd0jVd6SZ2Ir93r/aPejkyhQ03xh9RVVyhOn4bkbjKIBzEybXTJAXgNmvd6F8Ds00srBZVWlo7Z23JZ7\",\"a_responses\":{\"0\":\"QHTznWWrECRNNmUNcy0yGu2L6qsZU6qkvaII8QB8QjbUxpwHzSeJWkzrn/Kk1KIowfoqB1DKGaFLATvuBl+bCoJjea+2VfK9Ns8=\",\"2\":\"H57Y9CTXJ5MAVo+aFfNSbmRMFQpraBIZVOXiRxCD/P7Aw4fW8r9P5l9pO9DTUeExaqFzsLyF5i5EridVWxlP2Wv0zbH8ku9Sg9w=\",\"3\":\"joggAmOhqM4QsKdoLHAfaslzXqJswS7MwZ/5+AKYdkMaHQ45biMdZU/6R+B7bjvsumg2f6KyTyg0G+BI+wVdJOjh3kGezdANB7Y=\",\"5\":\"5YP4A82WWeqc33e5Zg/Q8lqQQ1amLE8mOxMwCXb3N4J0UJRfV9lUFvbH1Q3Yb3YHAZpzGvhN/pBacwqktMkP4L71PnMldqA+nqA=\"},\"a_disclosed\":{\"1\":\"AgAJuwB+AALWy2qU9p3l52l9LU1rVT4M\",\"4\":\"NDU2\"}}],\"nonce\":\"Kg==\",\"context\":\"BTk=\",\"message\":\"I owe you everything\",\"timestamp\":{\"Time\":1527196489,\"ServerUrl\":\"https://metrics.privacybydesign.foundation/atum\",\"Sig\":{\"Alg\":\"ed25519\",\"Data\":\"ZV1qkvDrFK14QrUSC66xTNr9HitCOV4vwfGX0bh3iwY7qyHCi9rIOE97KY8CZifU5oLgVhFWy5E+ALR+gEpACw==\",\"PublicKey\":\"e/nMAJF7nwrvNZRpuJljNpRx+CsT7caaXyn9OX683R8=\"}}}"
	irmaSignedMessage := &SignedMessage{}
	err := json.Unmarshal([]byte(irmaSignedMessageJson), irmaSignedMessage)
	require.NoError(t, err)

	attrs, status, err := irmaSignedMessage.Verify(conf, nil)
	require.NoError(t, err)
	require.Equal(t, ProofStatusValid, status)
	require.Len(t, attrs, 1)
	require.Equal(t, "456", attrs[0][0].Value["en"])
}

func TestVerifyInValidSig(t *testing.T) {
	conf := parseConfiguration(t)

	// Same json as valid case, but has modified c
	irmaSignedMessageJson := "{\"signature\":[{\"c\":\"blablaE7wXcDcKXuBtZW5bnucvBSXpILIRvnNBgx7hQ=\",\"A\":\"D/8wLPq9860bpXZ5c+VYyoPJ+Z8CWDZNQ0jXvst8qnPRdivy/GQIfJHjVnpOPlHbguphb/7JVbfcV3bZeybA3bCF/4UesjRUZlMf/iJ/QgKHbt41ogN1PPT5z7qBJpkxuNTIkHxaUPoDvhouHmuC9pNj4afRUyLJerxKPkpdBw0=\",\"e_response\":\"YOrKTrMSs4/QOUtPkT0YaYNEmW7Cs+cu624zr2xrHodyL88ub6yaXB7MGHAcQ1+iXsGN8jkfxB/0\",\"v_response\":\"AYSa1p8ISs//MsocJjODwWuPB/z6+iKHHi+sTToRs0eJ2X1gwmWoA5QB0aHjRkWye3/+2rtosfUzI77FlPQVnrbMERwcuYM/fx3fpNCpjm2qcs3AOJRcSRxcNFMe1+4ECsmJhByMDutS1KXAAKiNvnhEXx9f0JrQGwQFtpSFPh8dOuvEKUZHAUALr4FcHCa2HL9nDRiqy2KAOxE0nAANAcMaBo/ed+WZeHtv4CTB7egyYs27cklVbwlBzmRrbjNZk57ICd0jVd6SZ2Ir93r/aPejkyhQ03xh9RVVyhOn4bkbjKIBzEybXTJAXgNmvd6F8Ds00srBZVWlo7Z23JZ7\",\"a_responses\":{\"0\":\"QHTznWWrECRNNmUNcy0yGu2L6qsZU6qkvaII8QB8QjbUxpwHzSeJWkzrn/Kk1KIowfoqB1DKGaFLATvuBl+bCoJjea+2VfK9Ns8=\",\"2\":\"H57Y9CTXJ5MAVo+aFfNSbmRMFQpraBIZVOXiRxCD/P7Aw4fW8r9P5l9pO9DTUeExaqFzsLyF5i5EridVWxlP2Wv0zbH8ku9Sg9w=\",\"3\":\"joggAmOhqM4QsKdoLHAfaslzXqJswS7MwZ/5+AKYdkMaHQ45biMdZU/6R+B7bjvsumg2f6KyTyg0G+BI+wVdJOjh3kGezdANB7Y=\",\"5\":\"5YP4A82WWeqc33e5Zg/Q8lqQQ1amLE8mOxMwCXb3N4J0UJRfV9lUFvbH1Q3Yb3YHAZpzGvhN/pBacwqktMkP4L71PnMldqA+nqA=\"},\"a_disclosed\":{\"1\":\"AgAJuwB+AALWy2qU9p3l52l9LU1rVT4M\",\"4\":\"NDU2\"}}],\"nonce\":\"Kg==\",\"context\":\"BTk=\",\"message\":\"I owe you everything\",\"timestamp\":{\"Time\":1527196489,\"ServerUrl\":\"https://metrics.privacybydesign.foundation/atum\",\"Sig\":{\"Alg\":\"ed25519\",\"Data\":\"ZV1qkvDrFK14QrUSC66xTNr9HitCOV4vwfGX0bh3iwY7qyHCi9rIOE97KY8CZifU5oLgVhFWy5E+ALR+gEpACw==\",\"PublicKey\":\"e/nMAJF7nwrvNZRpuJljNpRx+CsT7caaXyn9OX683R8=\"}}}"
	irmaSignedMessage := &SignedMessage{}
	err := json.Unmarshal([]byte(irmaSignedMessageJson), irmaSignedMessage)
	require.NoError(t, err)

	_, status, err := irmaSignedMessage.Verify(conf, nil)
	require.NoError(t, err)
	require.Equal(t, status, ProofStatusInvalid)
}

func TestVerifyInValidNonce(t *testing.T) {
	conf := parseConfiguration(t)

	// Same json as valid case, but with modified nonce
	irmaSignedMessageJson := "{\"signature\":[{\"c\":\"pliyrSE7wXcDcKXuBtZW5bnucvBSXpILIRvnNBgx7hQ=\",\"A\":\"D/8wLPq9860bpXZ5c+VYyoPJ+Z8CWDZNQ0jXvst8qnPRdivy/GQIfJHjVnpOPlHbguphb/7JVbfcV3bZeybA3bCF/4UesjRUZlMf/iJ/QgKHbt41ogN1PPT5z7qBJpkxuNTIkHxaUPoDvhouHmuC9pNj4afRUyLJerxKPkpdBw0=\",\"e_response\":\"YOrKTrMSs4/QOUtPkT0YaYNEmW7Cs+cu624zr2xrHodyL88ub6yaXB7MGHAcQ1+iXsGN8jkfxB/0\",\"v_response\":\"AYSa1p8ISs//MsocJjODwWuPB/z6+iKHHi+sTToRs0eJ2X1gwmWoA5QB0aHjRkWye3/+2rtosfUzI77FlPQVnrbMERwcuYM/fx3fpNCpjm2qcs3AOJRcSRxcNFMe1+4ECsmJhByMDutS1KXAAKiNvnhEXx9f0JrQGwQFtpSFPh8dOuvEKUZHAUALr4FcHCa2HL9nDRiqy2KAOxE0nAANAcMaBo/ed+WZeHtv4CTB7egyYs27cklVbwlBzmRrbjNZk57ICd0jVd6SZ2Ir93r/aPejkyhQ03xh9RVVyhOn4bkbjKIBzEybXTJAXgNmvd6F8Ds00srBZVWlo7Z23JZ7\",\"a_responses\":{\"0\":\"QHTznWWrECRNNmUNcy0yGu2L6qsZU6qkvaII8QB8QjbUxpwHzSeJWkzrn/Kk1KIowfoqB1DKGaFLATvuBl+bCoJjea+2VfK9Ns8=\",\"2\":\"H57Y9CTXJ5MAVo+aFfNSbmRMFQpraBIZVOXiRxCD/P7Aw4fW8r9P5l9pO9DTUeExaqFzsLyF5i5EridVWxlP2Wv0zbH8ku9Sg9w=\",\"3\":\"joggAmOhqM4QsKdoLHAfaslzXqJswS7MwZ/5+AKYdkMaHQ45biMdZU/6R+B7bjvsumg2f6KyTyg0G+BI+wVdJOjh3kGezdANB7Y=\",\"5\":\"5YP4A82WWeqc33e5Zg/Q8lqQQ1amLE8mOxMwCXb3N4J0UJRfV9lUFvbH1Q3Yb3YHAZpzGvhN/pBacwqktMkP4L71PnMldqA+nqA=\"},\"a_disclosed\":{\"1\":\"AgAJuwB+AALWy2qU9p3l52l9LU1rVT4M\",\"4\":\"NDU2\"}}],\"nonce\":\"aa==\",\"context\":\"BTk=\",\"message\":\"I owe you everything\",\"timestamp\":{\"Time\":1527196489,\"ServerUrl\":\"https://metrics.privacybydesign.foundation/atum\",\"Sig\":{\"Alg\":\"ed25519\",\"Data\":\"ZV1qkvDrFK14QrUSC66xTNr9HitCOV4vwfGX0bh3iwY7qyHCi9rIOE97KY8CZifU5oLgVhFWy5E+ALR+gEpACw==\",\"PublicKey\":\"e/nMAJF7nwrvNZRpuJljNpRx+CsT7caaXyn9OX683R8=\"}}}"
	irmaSignedMessage := &SignedMessage{}
	require.NoError(t, json.Unmarshal([]byte(irmaSignedMessageJson), irmaSignedMessage))

	_, status, err := irmaSignedMessage.Verify(conf, nil)
	require.NoError(t, err)
	require.Equal(t, status, ProofStatusInvalid)
}

func TestEmptySignature(t *testing.T) {
	msg := &SignedMessage{}
	_, status, _ := msg.Verify(&Configuration{}, nil)
	require.NotEqual(t, ProofStatusValid, status)
}

// Test attribute decoding with both old and new metadata versions
func TestAttributeDecoding(t *testing.T) {
	expected := "male"

	newAttribute, _ := new(big.Int).SetString("3670202571", 10)
	newString := decodeAttribute(newAttribute, 3)
	require.Equal(t, *newString, expected)

	oldAttribute, _ := new(big.Int).SetString("1835101285", 10)
	oldString := decodeAttribute(oldAttribute, 2)
	require.Equal(t, *oldString, expected)
}

func TestSessionRequests(t *testing.T) {
	attrval := "hello"
	sigMessage := "message to be signed"

	base := &DisclosureRequest{
		BaseRequest: BaseRequest{LDContext: LDContextDisclosureRequest},
		Disclose: AttributeConDisCon{
			AttributeDisCon{
				AttributeCon{NewAttributeRequest("irma-demo.MijnOverheid.ageLimits.over18")},
				AttributeCon{NewAttributeRequest("irma-demo.MijnOverheid.ageLimits.over21")},
			},
			AttributeDisCon{
				AttributeCon{AttributeRequest{Type: NewAttributeTypeIdentifier("irma-demo.MijnOverheid.fullName.firstname"), Value: &attrval}},
			},
		},
		Labels: map[int]TranslatedString{0: trivialTranslation("Age limit"), 1: trivialTranslation("First name")},
	}

	tests := []struct {
		oldJson, currentJson   string
		old, current, expected SessionRequest
	}{
		{
			expected: base,
			old:      &DisclosureRequest{},
			oldJson: `{
				"type": "disclosing",
				"content": [{
					"label": "Age limit",
					"attributes": [
						"irma-demo.MijnOverheid.ageLimits.over18",
						"irma-demo.MijnOverheid.ageLimits.over21"
					]
				},
				{
					"label": "First name",
					"attributes": {
						"irma-demo.MijnOverheid.fullName.firstname": "hello"
					}
				}]
			}`,
			current: &DisclosureRequest{},
			currentJson: `{
				"@context": "https://irma.app/ld/request/disclosure/v2",
				"disclose": [
					[
						[
							"irma-demo.MijnOverheid.ageLimits.over18"
						],
						[
							"irma-demo.MijnOverheid.ageLimits.over21"
						]
					],
					[
						[
							{ "type": "irma-demo.MijnOverheid.fullName.firstname", "value": "hello" }
						]
					]
				],
				"labels": {
					"0": {
						"en": "Age limit",
						"nl": "Age limit"
					},
					"1": {
						"en": "First name",
						"nl": "First name"
					}
				}
			}`,
		},

		{
			expected: &SignatureRequest{
				DisclosureRequest{BaseRequest{LDContext: LDContextSignatureRequest}, base.Disclose, base.Labels},
				sigMessage,
			},
			old: &SignatureRequest{},
			oldJson: `{
				"type": "signing",
				"message": "message to be signed",
				"content": [{
					"label": "Age limit",
					"attributes": [
						"irma-demo.MijnOverheid.ageLimits.over18",
						"irma-demo.MijnOverheid.ageLimits.over21"
					]
				},
				{
					"label": "First name",
					"attributes": {
						"irma-demo.MijnOverheid.fullName.firstname": "hello"
					}
				}]
			}`,
			current: &SignatureRequest{},
			currentJson: `{
				"@context": "https://irma.app/ld/request/signature/v2",
				"disclose": [
					[
						[
							"irma-demo.MijnOverheid.ageLimits.over18"
						],
						[
							"irma-demo.MijnOverheid.ageLimits.over21"
						]
					],
					[
						[
							{ "type": "irma-demo.MijnOverheid.fullName.firstname", "value": "hello" }
						]
					]
				],
				"labels": {
					"0": {
						"en": "Age limit",
						"nl": "Age limit"
					},
					"1": {
						"en": "First name",
						"nl": "First name"
					}
				},
				"message": "message to be signed"
			}`,
		},

		{
			expected: &IssuanceRequest{
				DisclosureRequest: DisclosureRequest{BaseRequest{LDContext: LDContextIssuanceRequest}, base.Disclose, base.Labels},
				Credentials: []*CredentialRequest{
					{
						CredentialTypeID: NewCredentialTypeIdentifier("irma-demo.MijnOverheid.root"),
						Attributes:       map[string]string{"BSN": "12345"},
					},
				},
			},
			old: &IssuanceRequest{},
			oldJson: `{
				"type": "issuing",
				"credentials": [{
					"credential": "irma-demo.MijnOverheid.root",
					"attributes": { "BSN": "12345" }
				}],
				"disclose": [{
					"label": "Age limit",
					"attributes": [
						"irma-demo.MijnOverheid.ageLimits.over18",
						"irma-demo.MijnOverheid.ageLimits.over21"
					]
				},
				{
					"label": "First name",
					"attributes": {
						"irma-demo.MijnOverheid.fullName.firstname": "hello"
					}
				}]
			}`,
			current: &IssuanceRequest{},
			currentJson: `{
				"@context": "https://irma.app/ld/request/issuance/v2",
				"credentials": [
					{
						"credential": "irma-demo.MijnOverheid.root",
						"attributes": {
							"BSN": "12345"
						}
					}
				],
				"disclose": [
					[
						[
							"irma-demo.MijnOverheid.ageLimits.over18"
						],
						[
							"irma-demo.MijnOverheid.ageLimits.over21"
						]
					],
					[
						[
							{ "type": "irma-demo.MijnOverheid.fullName.firstname", "value": "hello" }
						]
					]
				],
				"labels": {
					"0": {
						"en": "Age limit",
						"nl": "Age limit"
					},
					"1": {
						"en": "First name",
						"nl": "First name"
					}
				}
			}`,
		},
	}

	for _, tst := range tests {
		require.NoError(t, json.Unmarshal([]byte(tst.oldJson), tst.old))
		require.NoError(t, json.Unmarshal([]byte(tst.currentJson), tst.current))
		tst.old.Base().legacy = false // We don't care about this field differing, override it
		tst.old.Base().Type = ""      // same
		require.True(t, reflect.DeepEqual(tst.old, tst.expected), "Legacy %s did not unmarshal to expected value", reflect.TypeOf(tst.old).String())
		require.True(t, reflect.DeepEqual(tst.current, tst.expected), "%s did not unmarshal to expected value", reflect.TypeOf(tst.old).String())

		_, err := tst.expected.Legacy()
		require.NoError(t, err)
	}
}

func trivialTranslation(str string) TranslatedString {
	return TranslatedString{"en": str, "nl": str}
}

func TestConDisconSingletons(t *testing.T) {
	tests := []struct {
		attrs   AttributeConDisCon
		allowed bool
	}{
		{
			AttributeConDisCon{
				AttributeDisCon{
					AttributeCon{
						NewAttributeRequest("irma-demo.RU.studentCard.studentID"), // non singleton
						NewAttributeRequest("test.test.email.email"),              // non singleton
					},
				},
			},
			false, // multiple non-singletons in one inner conjunction is not allowed
		},
		{
			AttributeConDisCon{
				AttributeDisCon{
					AttributeCon{
						NewAttributeRequest("irma-demo.RU.studentCard.studentID"), // non singleton
						NewAttributeRequest("test.test.mijnirma.email"),           // singleton
					},
				},
			},
			true,
		},
		{
			AttributeConDisCon{
				AttributeDisCon{
					AttributeCon{
						NewAttributeRequest("irma-demo.MijnOverheid.root.BSN"), // singleton
						NewAttributeRequest("test.test.mijnirma.email"),        // singleton
					},
				},
			},
			true,
		},
	}

	conf := parseConfiguration(t)
	for _, args := range tests {
		if args.allowed {
			require.NoError(t, args.attrs.Validate(conf))
		} else {
			require.Error(t, args.attrs.Validate(conf))
		}
	}
}

func parseDisclosure(t *testing.T) (*Configuration, *DisclosureRequest, *Disclosure) {
	conf := parseConfiguration(t)

	requestJson := `{"@context":"https://irma.app/ld/request/disclosure/v2","context":"AQ==","nonce":"zVQJMG6TKZwfcv5TExFVSQ==","protocolVersion":"2.5","disclose":[[["irma-demo.RU.studentCard.studentID"]]],"labels":{"0":null}}`
	dislosureJson := `{"proofs":[{"c":"o21UPItMKWXmXNhBKsCBHDWjfRoy+uDdbDB1yhhpg3k=","A":"Bl68Ut2nu2nwhIweU9QGoNd6TkjUIRbQ6SDg22m8PzMEgca0KA4/Oy1gaJCUHM3FFJ0Gdj0+6/VpcF85JyuQZou93UXXwzN/Y7ohUw+YxVTQ7WcJmZ/VGDh3SME5KJ9aWjGmq61J2LQiiDSq+XrcWFfKPwad6BkDhV2reo4yo68=","e_response":"VD0pWdeDkd3V+R3734xyRcGeWMMTzpB0ZiJhKMzv37DmHN6RpRzTF/0HroAsMIMz8mBWxYPVRBiw","v_response":"3OWsmIDM7v0ByEXax2YZGp3BnJ5nkCLMcT6/ENU0EcpjrOz+rT+NayQSLgMshxAATpgkgAluFQ3owOoQEL8ZAkZTWUDW5j+qy7GDFd22ZOKEZLWf8Q1XRK3x6exV9CIMkcBQrv5W6EI9XB5OKKNB3Z/VTALY3UW8cQQ0DPHj83YBEL3LJQDxwaxvQeHx4nysJjsEoLJE1KPBynXlfxpk17O3HTg+NuX5gj7+ckiHrmXgthJHvqCTnNpEORtXDJTmKJUccUiyWuftA36cIXIxW4N6I88T4BYctwN+T9NY+hcjYESITtxB+r2elB98bzlWgHF8ohpOkkJGuNjTFjw=","a_responses":{"0":"eDQA3Lrh2WC3o/VP6KD/uaMSRy/em3gEfuqXD9tVT+yJFYb7GT91lle5dB6lg235pUSHzYIOET7FYOHwb4/YSAGQiix0IzqFkLo=","2":"kT3kfcIaPy3UBYPX78X10w/R1Cb5rHqoW5OUd06xqC1V9MqVw3zhtc/nBgWmvVwTgJrl2CyuBjjoF10RJz/FEjYZ0JAF57uUXW8=","3":"4oSBcyUT6mOBhk/Szk/5G5QrgaAADW6wSl91hGwTTNDTIUiK01GE11JozbwDeZsLPoFikzikwkPu9ZsOAtOtb/+IcadB6NP0KXA=","5":"OwUSSCBb9NOMOYYSGSYCrdFUNLKJ/b2YP5LlElFG5r4GPR71zTQsZ4QuJiMIt9iFPRP6PQUvMvjWA59UTQ9AlwKc9JcQzbScYBM="},"a_disclosed":{"1":"AwAKOQIBAALWy2qU9p3l52l9LU1rVT4M","4":"aGpt"}}],"indices":[[{"cred":0,"attr":4}]]}`
	request := &DisclosureRequest{}
	require.NoError(t, json.Unmarshal([]byte(requestJson), request))
	disclosure := &Disclosure{}
	require.NoError(t, json.Unmarshal([]byte(dislosureJson), disclosure))

	return conf, request, disclosure
}

func TestVerify(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		conf, request, disclosure := parseDisclosure(t)
		attr, status, err := disclosure.Verify(conf, request)
		require.NoError(t, err)
		require.Equal(t, ProofStatusValid, status)
		require.Equal(t, "456", *attr[0][0].RawValue)
	})

	t.Run("invalid", func(t *testing.T) {
		conf, request, disclosure := parseDisclosure(t)
		disclosure.Proofs[0].(*gabi.ProofD).AResponses[0] = big.NewInt(100)
		_, status, err := disclosure.Verify(conf, request)
		require.NoError(t, err)
		require.Equal(t, ProofStatusInvalid, status)
	})

	t.Run("wrong attribute", func(t *testing.T) {
		conf, request, disclosure := parseDisclosure(t)
		request.Disclose[0][0][0].Type = NewAttributeTypeIdentifier("irma-demo.MijnOverheid.root.BSN")
		_, status, err := disclosure.Verify(conf, request)
		require.NoError(t, err)
		require.Equal(t, ProofStatusMissingAttributes, status)
	})

	t.Run("wrong nonce", func(t *testing.T) {
		conf, request, disclosure := parseDisclosure(t)
		request.Nonce = big.NewInt(100)
		_, status, err := disclosure.Verify(conf, request)
		require.NoError(t, err)
		require.Equal(t, ProofStatusInvalid, status)
	})
}

var (
	revocationTestCred  = NewCredentialTypeIdentifier("irma-demo.MijnOverheid.root")
	revocationPkCounter = uint(2)
)

func TestRevocationMemoryStore(t *testing.T) {
	conf := parseConfiguration(t)
	db := conf.Revocation.memdb
	require.NotNil(t, db)

	// prepare key material
	sk, err := conf.Revocation.Keys.PrivateKey(revocationTestCred.IssuerIdentifier(), revocationPkCounter)
	require.NoError(t, err)
	pk, err := conf.Revocation.Keys.PublicKey(revocationTestCred.IssuerIdentifier(), revocationPkCounter)
	require.NoError(t, err)

	// construct initial update
	update, err := revocation.NewAccumulator(sk)
	require.NoError(t, err)

	// insert and retrieve it and check its validity
	db.Insert(revocationTestCred, update)
	retrieve(t, pk, db, 0, 0)

	// construct new update message with a few revocation events
	update = revokeMultiple(t, sk, update)
	oldupdate := *update // save a copy for below

	// insert it, retrieve it with a varying amount of events, verify
	db.Insert(revocationTestCred, update)
	retrieve(t, pk, db, 4, 3)

	// construct and test against a new update whose events have no overlap with that of our db
	update = revokeMultiple(t, sk, update)
	update.Events = update.Events[4:]
	require.Equal(t, uint64(4), update.Events[0].Index)
	db.Insert(revocationTestCred, update)
	retrieve(t, pk, db, 4, 6)

	// attempt to insert an update that is too new
	update = revokeMultiple(t, sk, update)
	update.Events = update.Events[5:]
	require.Equal(t, uint64(9), update.Events[0].Index)
	db.Insert(revocationTestCred, update)
	retrieve(t, pk, db, 4, 6)

	// attempt to insert an update that is too old
	db.Insert(revocationTestCred, &oldupdate)
	retrieve(t, pk, db, 4, 6)
}

func revokeMultiple(t *testing.T, sk *revocation.PrivateKey, update *revocation.Update) *revocation.Update {
	acc := update.SignedAccumulator.Accumulator
	event := update.Events[len(update.Events)-1]
	events := update.Events
	for i := 0; i < 3; i++ {
		acc, event = revoke(t, acc, event, sk)
		events = append(events, event)
	}
	update, err := revocation.NewUpdate(sk, acc, events)
	require.NoError(t, err)
	return update
}

func retrieve(t *testing.T, pk *revocation.PublicKey, db memRevStorage, count uint64, expectedIndex uint64) {
	var updates map[uint]*revocation.Update
	var err error
	for i := uint64(0); i <= count; i++ {
		updates = db.Latest(revocationTestCred, i)
		require.Len(t, updates, 1)
		require.NotNil(t, updates[revocationPkCounter])
		require.Len(t, updates[revocationPkCounter].Events, int(i))
		_, err = updates[revocationPkCounter].Verify(pk)
		require.NoError(t, err)
	}
	sacc := db.SignedAccumulator(revocationTestCred, revocationPkCounter)
	acc, err := sacc.UnmarshalVerify(pk)
	require.NoError(t, err)
	require.Equal(t, expectedIndex, acc.Index)
}

func revoke(t *testing.T, acc *revocation.Accumulator, parent *revocation.Event, sk *revocation.PrivateKey) (*revocation.Accumulator, *revocation.Event) {
	e, err := rand.Prime(rand.Reader, 100)
	require.NoError(t, err)
	acc, event, err := acc.Remove(sk, big.Convert(e), parent)
	require.NoError(t, err)
	return acc, event
}

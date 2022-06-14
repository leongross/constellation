//go:build gcp
// +build gcp

package gcp

import (
	"encoding/json"
	"testing"

	"github.com/edgelesssys/constellation/internal/attestation/vtpm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAttestation(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	issuer := NewIssuer()
	validator := NewValidator(map[uint32][]byte{0: PCR0})

	nonce := []byte{2, 3, 4}
	challenge := []byte("Constellation")

	attDocRaw, err := issuer.Issue(challenge, nonce)
	assert.NoError(err)

	var attDoc vtpm.AttestationDocument
	err = json.Unmarshal(attDocRaw, &attDoc)
	require.NoError(err)
	assert.Equal(challenge, attDoc.UserData)
	originalPCR := attDoc.Attestation.Quotes[1].Pcrs.Pcrs[uint32(vtpm.PCRIndexOwnerID)]

	out, err := validator.Validate(attDocRaw, nonce)
	assert.NoError(err)
	assert.Equal(challenge, out)

	// Mark node as intialized. We should still be abe to validate
	assert.NoError(vtpm.MarkNodeAsInitialized(vtpm.OpenVTPM, []byte("Test"), []byte("Nonce")))

	attDocRaw, err = issuer.Issue(challenge, nonce)
	assert.NoError(err)

	// Make sure the PCR changed
	err = json.Unmarshal(attDocRaw, &attDoc)
	require.NoError(err)
	assert.NotEqual(originalPCR, attDoc.Attestation.Quotes[1].Pcrs.Pcrs[uint32(vtpm.PCRIndexOwnerID)])

	out, err = validator.Validate(attDocRaw, nonce)
	assert.NoError(err)
	assert.Equal(challenge, out)
}
package layout

import (
	"errors"

	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/partial"
	"github.com/sigstore/cosign/v2/pkg/oci"
)

var ErrImageNotFound = errors.New("image not found in registry")

type image struct {
	v1.Image
	path string
}

// The wrapped Image implements ConfigLayer, but the wrapping hides that from typechecks in pkg/v1/remote.
// Make image explicitly implement ConfigLayer so that this returns a mountable config layer for pkg/v1/remote.
func (i *image) ConfigLayer() (v1.Layer, error) {
	return partial.ConfigLayer(i.Image)
}

var _ oci.SignedImage = (*image)(nil)

// Signatures implements oci.SignedImage
func (i *image) Signatures() (oci.Signatures, error) {
	return signatures(i, i.path)
}

// Attestations implements oci.SignedImage
func (i *image) Attestations() (oci.Signatures, error) {
	return attestations(i, i.path)
}

// Attestations implements oci.SignedImage
func (i *image) Attachment(name string) (oci.File, error) {
	return nil, nil
}

// signatures is a shared implementation of the oci.Signed* Signatures method.
func signatures(digestable oci.SignedEntity, path string) (oci.Signatures, error) {
	h, err := digestable.Digest()
	if err != nil {
		return nil, err
	}
	return Signatures(normalize(h, CustomTagPrefix, SignatureTagSuffix), path)
}

// attestations is a shared implementation of the oci.Signed* Attestations method.
func attestations(digestable oci.SignedEntity, path string) (oci.Signatures, error) {
	h, err := digestable.Digest()
	if err != nil {
		return nil, err
	}
	return Signatures(normalize(h, CustomTagPrefix, AttestationTagSuffix), path)
}

package layout

import (
	"errors"
	"fmt"
	"io"
	"os"
	"sync"

	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/partial"
	"github.com/google/go-containerregistry/pkg/v1/types"
	"github.com/sigstore/cosign/v2/pkg/oci"
)

var ErrImageNotFound = errors.New("image not found in registry")

type v1Image v1.Image
type image struct {
	v1Image
	path string
}

// The wrapped Image implements ConfigLayer, but the wrapping hides that from typechecks in pkg/v1/remote.
// Make image explicitly implement ConfigLayer so that this returns a mountable config layer for pkg/v1/remote.
func (i *image) ConfigLayer() (v1.Layer, error) {
	return partial.ConfigLayer(i.v1Image)
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
func (i *image) Attachment(_ string) (oci.File, error) {
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

type layoutImage struct {
	path         Path
	desc         v1.Descriptor
	manifestLock sync.Mutex // Protects rawManifest
	rawManifest  []byte
}

var _ partial.CompressedImageCore = (*layoutImage)(nil)

// Image reads a v1.Image with digest h from the Path.
func (l Path) Image(h v1.Hash) (v1.Image, error) {
	ii, err := l.ImageIndex()
	if err != nil {
		return nil, err
	}

	return ii.Image(h)
}

func (li *layoutImage) MediaType() (types.MediaType, error) {
	return li.desc.MediaType, nil
}

// Implements WithManifest for partial.Blobset.
func (li *layoutImage) Manifest() (*v1.Manifest, error) {
	return partial.Manifest(li)
}

func (li *layoutImage) RawManifest() ([]byte, error) {
	li.manifestLock.Lock()
	defer li.manifestLock.Unlock()
	if li.rawManifest != nil {
		return li.rawManifest, nil
	}

	b, err := li.path.Bytes(li.desc.Digest)
	if err != nil {
		return nil, err
	}

	li.rawManifest = b
	return li.rawManifest, nil
}

func (li *layoutImage) RawConfigFile() ([]byte, error) {
	manifest, err := li.Manifest()
	if err != nil {
		return nil, err
	}

	return li.path.Bytes(manifest.Config.Digest)
}

func (li *layoutImage) LayerByDigest(h v1.Hash) (partial.CompressedLayer, error) {
	manifest, err := li.Manifest()
	if err != nil {
		return nil, err
	}

	if h == manifest.Config.Digest {
		return &compressedBlob{
			path: li.path,
			desc: manifest.Config,
		}, nil
	}

	for _, desc := range manifest.Layers {
		if h == desc.Digest {
			return &compressedBlob{
				path: li.path,
				desc: desc,
			}, nil
		}
	}

	return nil, fmt.Errorf("could not find layer in image: %s", h)
}

type compressedBlob struct {
	path Path
	desc v1.Descriptor
}

func (b *compressedBlob) Digest() (v1.Hash, error) {
	return b.desc.Digest, nil
}

func (b *compressedBlob) Compressed() (io.ReadCloser, error) {
	return b.path.Blob(b.desc.Digest)
}

func (b *compressedBlob) Size() (int64, error) {
	return b.desc.Size, nil
}

func (b *compressedBlob) MediaType() (types.MediaType, error) {
	return b.desc.MediaType, nil
}

// Descriptor implements partial.withDescriptor.
func (b *compressedBlob) Descriptor() (*v1.Descriptor, error) {
	return &b.desc, nil
}

// See partial.Exists.
func (b *compressedBlob) Exists() (bool, error) {
	_, err := os.Stat(b.path.blobPath(b.desc.Digest))
	if os.IsNotExist(err) {
		return false, nil
	}
	return err == nil, err
}

//
// Copyright 2021 The Sigstore Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package layout

import (
	"strings"

	v1 "github.com/spectrocloud/go-containerregistry/pkg/v1"
	"github.com/sigstore/cosign/v2/pkg/oci"
	"github.com/sigstore/cosign/v2/pkg/oci/empty"
	"github.com/sigstore/cosign/v2/pkg/oci/internal/signature"
)

const maxLayers = 1000

type sigs struct {
	v1.Image
}

var _ oci.Signatures = (*sigs)(nil)

// Get implements oci.Signatures
func (s *sigs) Get() ([]oci.Signature, error) {
	manifest, err := s.Manifest()
	if err != nil {
		return nil, err
	}
	numLayers := int64(len(manifest.Layers))
	if numLayers > maxLayers {
		return nil, oci.NewMaxLayersExceeded(numLayers, maxLayers)
	}
	signatures := make([]oci.Signature, 0, numLayers)
	for _, desc := range manifest.Layers {
		l, err := s.LayerByDigest(desc.Digest)
		if err != nil {
			return nil, err
		}
		signatures = append(signatures, signature.New(l, desc))
	}
	return signatures, nil
}

// Signatures fetches the signatures image represented by the named reference.
// If the tag is not found, this returns an empty oci.Signatures.
func Signatures(ref string, path string) (oci.Signatures, error) {
	sii, err := SignedImageIndex(path)
	if err != nil {
		return nil, err
	}

	manifest, err := sii.IndexManifest()
	if err != nil {
		return nil, err
	}
	for _, m := range manifest.Manifests {
		if val, ok := m.Annotations[KindAnnotation]; ok && val == SigsAnnotation {
			imgRef, ok := m.Annotations[ImageRefAnnotation]
			if !ok {
				continue
			}

			if !strings.HasSuffix(imgRef, ref) {
				continue
			}

			i, err := sii.Image(m.Digest)
			if err != nil {
				return nil, err
			}
			return &sigs{
				Image: i,
			}, nil
		}
	}

	return empty.Signatures(), nil
}

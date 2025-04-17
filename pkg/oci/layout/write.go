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
	"errors"
	"fmt"
	"maps"
	"os"

	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/empty"
	"github.com/google/go-containerregistry/pkg/v1/layout"
	"github.com/google/go-containerregistry/pkg/v1/match"
	"github.com/sigstore/cosign/v2/pkg/oci"
)

// WriteSignedImage writes the image and all related signatures, attestations and attachments
func WriteSignedImage(path string, si oci.SignedImage, ref name.Reference, extraAnnotations map[string]string) error {
	layoutPath, err := layout.FromPath(path)
	if os.IsNotExist(err) {
		// If the layout doesn't exist, create a new one
		layoutPath, err = layout.Write(path, empty.Index)
	}
	if err != nil {
		return err
	}

	if extraAnnotations == nil {
		extraAnnotations = make(map[string]string)
	}
	extraAnnotations[KindAnnotation] = ImageAnnotation

	// write the image
	if err := appendImage(layoutPath, si, ref, extraAnnotations); err != nil {
		return fmt.Errorf("appending signed image: %w", err)
	}
	return writeSignedEntity(layoutPath, si, ref)
}

// WriteSignedImageIndex writes the image index and all related signatures, attestations and attachments
func WriteSignedImageIndex(path string, si oci.SignedImageIndex, ref name.Reference, extraAnnotations map[string]string) error {
	layoutPath, err := layout.FromPath(path)
	if os.IsNotExist(err) {
		// If the layout doesn't exist, create a new one
		layoutPath, err = layout.Write(path, empty.Index)
	}
	if err != nil {
		return err
	}

	m, err := si.IndexManifest()
	if err != nil {
		return fmt.Errorf("getting index manifest: %w", err)
	}

	annotations := make(map[string]string)
	if m != nil {
		maps.Copy(annotations, m.Annotations)
	}

	if extraAnnotations != nil {
		maps.Copy(annotations, extraAnnotations)
	}
	annotations[KindAnnotation] = ImageIndexAnnotation
	imageRef, err := getImageRef(ref)
	if err != nil {
		return err // Return the error from getImageRef immediately.
	}
	annotations[ImageRefAnnotation] = imageRef

	if err := layoutPath.ReplaceIndex(si, match.Name(imageRef), layout.WithAnnotations(annotations)); err != nil {
		return fmt.Errorf("appending signed image index: %w", err)
	}

	return writeSignedEntity(layoutPath, si, ref)
}

func writeSignedEntity(path layout.Path, se oci.SignedEntity, ref name.Reference) error {
	// write the signatures
	sigs, err := se.Signatures()
	if err != nil {
		return fmt.Errorf("getting signatures: %w", err)
	}
	if !isEmpty(sigs) {
		h, err := se.Digest()
		if err != nil {
			return fmt.Errorf("getting digest: %w", err)
		}
		tag := ref.Context().Tag(normalize(h, CustomTagPrefix, SignatureTagSuffix))
		if err := appendImage(path, sigs, tag, map[string]string{
			KindAnnotation: SigsAnnotation,
		}); err != nil {
			return fmt.Errorf("appending signatures: %w", err)
		}
	}

	// write attestations
	atts, err := se.Attestations()
	if err != nil {
		return fmt.Errorf("getting atts")
	}
	if !isEmpty(atts) {
		h, err := se.Digest()
		if err != nil {
			return fmt.Errorf("getting digest: %w", err)
		}
		tag := ref.Context().Tag(normalize(h, CustomTagPrefix, AttestationTagSuffix))
		if err := appendImage(path, atts, tag, map[string]string{
			KindAnnotation: AttsAnnotation,
		}); err != nil {
			return fmt.Errorf("appending atts: %w", err)
		}
	}

	// TODO (priyawadhwa@) and attachments
	// temp handle sboms - amartin120@
	sboms, err := se.Attachment("sbom")
	if err != nil {
		return nil // no sbom found
	}
	if sboms != nil {
		h, err := se.Digest()
		if err != nil {
			return fmt.Errorf("getting digest: %w", err)
		}
		tag := ref.Context().Tag(normalize(h, CustomTagPrefix, SBOMTagSuffix))
		if err := appendImage(path, sboms, tag, map[string]string{
			KindAnnotation: SbomsAnnotation,
		}); err != nil {
			return fmt.Errorf("appending attachments: %w", err)
		}
	}
	return nil
}

// isEmpty returns true if the signatures or attestations are empty
func isEmpty(s oci.Signatures) bool {
	ss, _ := s.Get()
	return ss == nil
}

func appendImage(path layout.Path, img v1.Image, ref name.Reference, extraAnnotations map[string]string) error {
	imageRef, err := getImageRef(ref)
	if err != nil {
		return err // Return the error from getImageRef immediately.
	}

	m, err := img.Manifest()
	if err != nil {
		return fmt.Errorf("getting manifest: %w", err)
	}
	annotations := make(map[string]string)
	if m != nil {
		maps.Copy(annotations, m.Annotations)
	}
	if extraAnnotations != nil {
		maps.Copy(annotations, extraAnnotations)
	}
	annotations[ImageRefAnnotation] = imageRef

	return path.ReplaceImage(img,
		match.Name(imageRef),
		layout.WithAnnotations(
			annotations,
		),
	)
}

func getImageRef(ref name.Reference) (string, error) {
	if ref == nil {
		return "", errors.New("reference is nil")
	}
	imageRef := ref.Name()
	return imageRef, nil
}

func normalize(h v1.Hash, prefix string, suffix string) string {
	return normalizeWithSeparator(h, prefix, suffix, "-")
}

// normalizeWithSeparator turns image digests into tags with optional prefix & suffix:
// sha256:d34db33f -> [prefix]sha256[algorithmSeparator]d34db33f[.suffix]
func normalizeWithSeparator(h v1.Hash, prefix string, suffix string, algorithmSeparator string) string {
	if suffix == "" {
		return fmt.Sprint(prefix, h.Algorithm, algorithmSeparator, h.Hex)
	}
	return fmt.Sprint(prefix, h.Algorithm, algorithmSeparator, h.Hex, ".", suffix)
}

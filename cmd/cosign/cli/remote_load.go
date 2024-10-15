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

package cli

import (
	"context"
	"fmt"
	"path"

	"github.com/google/go-containerregistry/pkg/name"

	"github.com/sigstore/cosign/v2/cmd/cosign/cli/options"
	"github.com/sigstore/cosign/v2/pkg/oci"
	ociplatform "github.com/sigstore/cosign/v2/pkg/oci/platform"
	"github.com/sigstore/cosign/v2/pkg/oci/remote"
	ociremote "github.com/sigstore/cosign/v2/pkg/oci/remote"

	"github.com/spf13/cobra"
)

func RemoteLoad() *cobra.Command {
	o := &options.RemoteLoadOptions{}

	cmd := &cobra.Command{
		Use:              "remote-load",
		Example:          `cosign remote-load <IMAGE> --registry <REGISTRY>`,
		Args:             cobra.ExactArgs(1),
		PersistentPreRun: options.BindViper,
		RunE: func(cmd *cobra.Command, args []string) error {
			return RemoteLoadCmd(cmd.Context(), *o, args[0])
		},
	}

	o.AddFlags(cmd)
	return cmd
}

func RemoteLoadCmd(ctx context.Context, opts options.RemoteLoadOptions, imageRef string) error {
	srcRef, err := name.ParseReference(imageRef)
	if err != nil {
		return err
	}

	targetImage := path.Join(opts.Registry.Name, imageRef)
	targetRef, err := name.ParseReference(targetImage)
	if err != nil {
		return err
	}

	se, err := ociremote.SignedEntity(srcRef)
	if err != nil {
		return err
	}

	ociremoteOpts, err := opts.Registry.ClientOpts(ctx)
	if err != nil {
		return err
	}

	se, err = ociplatform.SignedEntityForPlatform(se, "")
	if err != nil {
		return err
	}

	if _, ok := se.(oci.SignedImage); ok {
		si := se.(oci.SignedImage)
		return remote.WriteSignedImage(si, targetRef, ociremoteOpts...)
	}

	if _, ok := se.(oci.SignedImageIndex); ok {
		sii := se.(oci.SignedImageIndex)
		return remote.WriteSignedImageIndexImages(targetRef, sii, ociremoteOpts...)
	}

	return fmt.Errorf("unsupported signed entity type")
}

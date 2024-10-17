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

	"github.com/google/go-containerregistry/pkg/crane"
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
		Example:          `cosign remote-load <SRC> <DST>`,
		Args:             cobra.ExactArgs(2),
		PersistentPreRun: options.BindViper,
		RunE: func(cmd *cobra.Command, args []string) error {
			return RemoteLoadCmd(cmd.Context(), *o, args[0], args[1])
		},
	}

	o.AddFlags(cmd)
	return cmd
}

func RemoteLoadCmd(ctx context.Context, opts options.RemoteLoadOptions, src, dst string) error {
	srcRef, err := name.ParseReference(src)
	if err != nil {
		return err
	}

	dstRef, err := name.ParseReference(dst)
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

	signed, err := imageHasSignature(se)
	if err != nil {
		return err
	}

	if !signed {
		return crane.Copy(src, dst)
	} else {
		fmt.Println("image has signature")
	}

	return remote.WriteSignedEntity(srcRef, dstRef, se, ociremoteOpts...)
}

func imageHasSignature(se oci.SignedEntity) (bool, error) {
	sigs, err := se.Signatures()
	if err != nil {
		return false, err
	}

	if sigs == nil {
		return false, nil
	}

	ss, err := sigs.Get()
	if err != nil {
		return false, err
	}

	return len(ss) > 0, nil
}

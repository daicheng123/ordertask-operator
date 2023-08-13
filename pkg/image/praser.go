package image

import (
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
)

func ParseImage(img string) (*ImageInfo, error) {
	ref, err := name.ParseReference(img, name.WeakValidation)
	if err != nil {
		return nil, err
	}
	des, err := remote.Get(ref)
	if err != nil {
		return nil, err
	}

	imgBuilder := NewImageInfo(ref, img, des.Digest)

	// image and index type, https://docs.docker.com/registry/spec/manifest-v2-2/
	if des.MediaType.IsImage() {
		image, err := des.Image()
		if err != nil {
			return nil, err
		}
		config, err := image.ConfigFile()
		imgBuilder.AddImageCommand(config.OS, config.Architecture, config.Config.Entrypoint, config.Config.Cmd)
	}
	if des.MediaType.IsIndex() {
		idx, err := des.ImageIndex()
		if err != nil {
			return nil, err
		}
		mf, err := idx.IndexManifest()
		if err != nil {
			return nil, err
		}
		for _, d := range mf.Manifests {
			image, err := idx.Image(d.Digest)
			if err != nil {
				return nil, err
			}
			config, err := image.ConfigFile()
			if err != nil {
				return nil, err
			}
			imgBuilder.AddImageCommand(
				config.OS, config.Architecture, config.Config.Entrypoint, config.Config.Cmd)
		}
	}
	return imgBuilder, nil
}

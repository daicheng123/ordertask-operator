package image

import (
	"fmt"
	registry_name "github.com/google/go-containerregistry/pkg/name"
	registry_v1 "github.com/google/go-containerregistry/pkg/v1"
	"strings"
)

type ImageInfoCache struct {
}

type ImageCommand struct {
	Command []string // 对应docker的entrypoint
	Args    []string // 对应docker的 cmd
}

type ImageInfo struct {
	Ref     registry_name.Reference  // 增加了一个, 缓存里直接用这个作为key
	Name    string                   // 譬如 alpine:3.12
	Digest  registry_v1.Hash         // 唯一的hash
	Command map[string]*ImageCommand // map的key 譬如 Linux/amd64
}

func NewImageInfo(ref registry_name.Reference, name string, digest registry_v1.Hash) *ImageInfo {
	return &ImageInfo{
		Ref:     ref,
		Name:    name,
		Digest:  digest,
		Command: make(map[string]*ImageCommand),
	}
}

func (info *ImageInfo) addImageCommand(os, arch string, cmds []string, args []string) {
	cmdKey := fmt.Sprintf("%s/%s", strings.ToLower(os), strings.ToLower(arch))
	info.Command[cmdKey] = &ImageCommand{
		Command: cmds,
		Args:    args,
	}
}

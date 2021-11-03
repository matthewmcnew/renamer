package renamer

import (
	"fmt"
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/google/go-containerregistry/pkg/v1/random"
	"github.com/google/go-containerregistry/pkg/v1/remote"
)

const (
	layerMetadataLabel     = "io.buildpacks.buildpack.layers"
	buildpackMetadataLabel = "io.buildpacks.buildpackage.metadata"
)

func Rename(buildpack, newID, tag string) (string, error) {
	reference, err := name.ParseReference(buildpack)
	if err != nil {
		return "", err
	}

	image, err := remote.Image(reference, remote.WithAuthFromKeychain(authn.DefaultKeychain))
	if err != nil {
		return "", err
	}

	metadata := Metadata{}
	err = GetLabel(image, buildpackMetadataLabel, &metadata)
	if err != nil {
		return "", err
	}

	layerMetadata := BuildpackLayerMetadata{}
	err = GetLabel(image, layerMetadataLabel, &layerMetadata)
	if err != nil {
		return "", err
	}

	newLayersMetedata, layers, err := layerMetadata.metadataAndLayersFor(image, metadata.Id, metadata.Version, newID)
	if err != nil {
		return "", err
	}

	newBuildpackage, err := random.Image(0, 0)
	if err != nil {
		return "", err
	}

	newBuildpackage, err = mutate.AppendLayers(newBuildpackage, layers...)
	if err != nil {
		return "", err
	}

	metadata.Id = newID
	newBuildpackage, err = SetLabels(newBuildpackage, map[string]interface{}{
		layerMetadataLabel:     newLayersMetedata,
		buildpackMetadataLabel: metadata,
	})

	reference, err = name.ParseReference(tag)
	if err != nil {
		return "", err
	}

	err = remote.Write(reference, newBuildpackage, remote.WithAuthFromKeychain(authn.DefaultKeychain))
	if err != nil {
		return "", err
	}

	digest, err := newBuildpackage.Digest()
	if err != nil {
		return "", err
	}

	identifer := fmt.Sprintf("%s@%s", tag, digest.String())

	return identifer, nil
}

func (m BuildpackLayerMetadata) metadataAndLayersFor(sourceImage v1.Image, oldId string, oldVersion string, newId string) (BuildpackLayerMetadata, []v1.Layer, error) {
	newLayerMetdata := BuildpackLayerMetadata{}

	var layers []v1.Layer
	for id, versions := range m {
		for v, buildpack := range versions {

			if v != oldVersion && id != oldId {
				if _, ok := newLayerMetdata[id]; !ok {
					newLayerMetdata[id] = map[string]BuildpackLayerInfo{}
				}

				newLayerMetdata[id][v] = buildpack

				diffId, err := v1.NewHash(buildpack.LayerDiffID)
				if err != nil {
					return nil, nil, err
				}
				layer, err := sourceImage.LayerByDiffID(diffId)
				if err != nil {
					return nil, nil, err
				}

				layers = append(layers, layer)
			} else {
				if _, ok := newLayerMetdata[newId]; !ok {
					newLayerMetdata[newId] = map[string]BuildpackLayerInfo{}
				}

				diffId, err := v1.NewHash(buildpack.LayerDiffID)
				if err != nil {
					return nil, nil, err
				}
				layer, err := sourceImage.LayerByDiffID(diffId)
				if err != nil {
					return nil, nil, err
				}

				layer, err = rewriteLayer(layer, oldId, newId)
				if err != nil {
					return nil, nil, err
				}

				diffID, err := layer.DiffID()
				if err != nil {
					return nil, nil, err
				}

				buildpack.LayerDiffID = diffID.String()

				newLayerMetdata[newId][v] = buildpack
				layers = append(layers, layer)
			}
		}
	}

	return newLayerMetdata, layers, nil
}

type BuildpackLayerMetadata map[string]map[string]BuildpackLayerInfo

type BuildpackLayerInfo struct {
	API         string  `json:"api"`
	Stacks      []Stack `json:"stacks,omitempty"`
	Order       Order   `json:"order,omitempty"`
	LayerDiffID string  `json:"layerDiffID"`
}

type Order []OrderEntry

type OrderEntry struct {
	Group []BuildpackRef `json:"group,omitempty"`
}

type BuildpackRef struct {
	BuildpackInfo `json:",inline"`
	Optional      bool `json:"optional,omitempty"`
}

type BuildpackInfo struct {
	Id      string `json:"id"`
	Version string `json:"version,omitempty"`
}

type Stack struct {
	ID     string   `json:"id"`
	Mixins []string `json:"mixins,omitempty"`
}

type Metadata struct {
	Id          string      `json:"id"`
	Version     string      `json:"version,omitempty"`
	Homepage    interface{} `json:"homepage,omitempty"`
	Description interface{} `json:"description,omitempty"`
	Keywords    interface{} `json:"keywords,omitempty"`
	Licenses    interface{} `json:"licenses,omitempty"`
	Stacks      interface{} `toml:"stacks" json:"stacks,omitempty"`
}

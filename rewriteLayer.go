package renamer

import (
	"archive/tar"
	"bytes"
	"github.com/BurntSushi/toml"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/tarball"
	"github.com/pkg/errors"
	"io/ioutil"
	"path"
	"strings"
)

func rewriteLayer(layer v1.Layer, old, new string) (v1.Layer, error) {
	b := &bytes.Buffer{}
	tw := tar.NewWriter(b)

	uncompressed, err := layer.Uncompressed()
	if err != nil {
		return nil, err
	}
	defer uncompressed.Close()

	tr := tar.NewReader(uncompressed)
	for {
		header, err := tr.Next()
		if err != nil {
			break
		}

		newName := strings.ReplaceAll(header.Name, escapedID(old), escapedID(new))

		if strings.HasSuffix(path.Clean(header.Name), "buildpack.toml") {
			buf, err := ioutil.ReadAll(tr)
			if err != nil {
				return nil, err
			}

			bd := BuildpackDescriptor{}
			_, err = toml.Decode(string(buf), &bd)
			if err != nil {
				return nil, errors.Wrap(err, "decoding buildpack.toml")
			}

			bd.Info.ID = new

			updatedBuildpackToml := &bytes.Buffer{}
			err = toml.NewEncoder(updatedBuildpackToml).Encode(bd)
			if err != nil {
				return nil, err
			}

			contents := updatedBuildpackToml.Bytes()
			header.Name = newName
			header.Size = int64(len(contents))
			err = tw.WriteHeader(header)
			if err != nil {
				return nil, err
			}

			_, err = tw.Write(contents)
			if err != nil {
				return nil, err
			}
		} else {
			header.Name = newName
			err = tw.WriteHeader(header)
			if err != nil {
				return nil, err
			}

			buf, err := ioutil.ReadAll(tr)
			if err != nil {
				return nil, err
			}

			_, err = tw.Write(buf)
			if err != nil {
				return nil, err
			}
		}

	}

	return tarball.LayerFromReader(b)
}

type BuildpackDescriptor struct {
	API      string            `toml:"api"`
	Info     BuildpackTomlInfo `toml:"buildpack"`
	Stacks   interface{}       `toml:"stacks"`
	Order    interface{}       `toml:"order"`
	Metadata interface{}       `toml:"metadata"`
}

type BuildpackTomlInfo struct {
	ID       string `toml:"id"`
	Version  string `toml:"version"`
	Name     string `toml:"name"`
	ClearEnv bool   `toml:"clear-env,omitempty"`
}

func escapedID(id string) string {
	return strings.ReplaceAll(id, "/", "_")
}

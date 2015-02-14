package util

import (
	"archive/tar"
	"encoding/json"
	"fmt"
	"io/ioutil"

	"github.com/appc/spec/schema"
	"github.com/appc/spec/schema/types"
	"github.com/coreos/rocket/cas"

	ptar "github.com/sgotti/acido/pkg/tar"
)

func GetImageManifest(hash *types.Hash, ds *cas.Store) (*schema.ImageManifest, error) {
	rs, err := ds.ReadStream(hash.String())
	if err != nil {
		return nil, fmt.Errorf("error extracting ImageManifest: %v", err)
	}

	imb, err := ptar.ExtractFileFromTar(tar.NewReader(rs), "manifest")
	if err != nil {
		return nil, fmt.Errorf("error extracting ImageManifest: %v", err)
	}

	var im schema.ImageManifest
	if err := json.Unmarshal(imb, &im); err != nil {
		return nil, fmt.Errorf("error unmarshaling image manifest: %v", err)
	}

	return &im, nil

}

func LoadImageManifest(file string) (*schema.ImageManifest, error) {
	imb, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, err
	}
	var im schema.ImageManifest
	if err := json.Unmarshal(imb, &im); err != nil {
		return nil, fmt.Errorf("error unmarshaling image manifest: %v", err)
	}
	return &im, nil
}

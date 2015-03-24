package util

import (
	"encoding/json"
	"fmt"
	"io/ioutil"

	"github.com/sgotti/acido/Godeps/_workspace/src/github.com/appc/spec/schema"
)

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

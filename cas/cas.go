package cas

import (
	"archive/tar"
	"crypto/sha512"
	"encoding/json"
	"fmt"
	"hash"

	"github.com/appc/spec/schema"
	"github.com/appc/spec/schema/types"
	"github.com/coreos/rocket/cas"

	ptar "github.com/sgotti/acido/pkg/tar"
)

// temp cas extension to handle ACIRegistry.GetImageManifest()
const (
	hashPrefix = "sha512-"
	lenHash    = sha512.Size       // raw byte size
	lenHashKey = (lenHash / 2) * 2 // half length, in hex characters
	lenKey     = len(hashPrefix) + lenHashKey
)

type ourCAS struct {
	*cas.Store
}

func NewStore(base string) (*ourCAS, error) {
	ds, err := cas.NewStore(base)
	return &ourCAS{ds}, err
}

func (c *ourCAS) GetImageManifest(key string) (*schema.ImageManifest, error) {
	rs, err := c.ReadStream(key)
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

func (c *ourCAS) GetACI(name types.ACName, labels types.Labels) (string, error) {
	return "", fmt.Errorf("At the moment an aci must be specified by its imageID")
}

// HashToKey takes a hash.Hash (which currently _MUST_ represent a full SHA512),
// calculates its sum, and returns a string which should be used as the key to
// store the data matching the hash.
func (c *ourCAS) HashToKey(h hash.Hash) string {
	s := h.Sum(nil)
	return keyToString(s)
}

// keyToString takes a key and returns a shortened and prefixed hexadecimal string version
func keyToString(k []byte) string {
	if len(k) != lenHash {
		panic(fmt.Sprintf("bad hash passed to hashToKey: %x", k))
	}
	return fmt.Sprintf("%s%x", hashPrefix, k)[0:lenKey]
}

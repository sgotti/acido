package acirenderer

import (
	"archive/tar"
	"container/list"
	"fmt"
	"os"
	"path/filepath"

	ptar "github.com/sgotti/acido/pkg/tar"
	"github.com/sgotti/acido/util"

	"github.com/appc/spec/schema"
	"github.com/appc/spec/schema/types"
	"github.com/coreos/rocket/cas"
)

// TODO by now the dependency are searched in the store by hash. Waiting for a fetch mechanism: see https://github.com/appc/spec/issues/16
type Image struct {
	im    *schema.ImageManifest
	Hash  *types.Hash
	Level uint16
}

func CreateDepList(hash *types.Hash, ds *cas.Store) (*list.List, error) {
	im, err := util.GetImageManifest(hash, ds)
	if err != nil {
		return nil, err
	}
	images := list.New()
	img := Image{im: im, Hash: hash, Level: 0}
	images.PushFront(img)

	for el := images.Front(); el != nil; el = el.Next() {
		img := el.Value.(Image)
		dependencies := img.im.Dependencies
		for _, d := range dependencies {
			hash := d.Hash
			im, err := util.GetImageManifest(&hash, ds)
			if err != nil {
				return nil, err
			}
			depimg := Image{im: im, Hash: &hash, Level: img.Level + 1}
			images.InsertAfter(depimg, el)
		}
	}

	return images, nil
}

func RenderImage(hashStr string, dir string, ds *cas.Store) error {
	hash, err := types.NewHash(hashStr)
	if err != nil {
		return err
	}
	images, err := CreateDepList(hash, ds)
	if err != nil {
		return err
	}

	err = renderImage(images, dir, ds)
	if err != nil {
		return err
	}

	return nil
}

func renderImage(images *list.List, dir string, ds *cas.Store) error {
	img := images.Back().Value.(Image)
	prevlevel := img.Level

	for el := images.Back(); el != nil; el = el.Prev() {
		img := el.Value.(Image)
		rs, err := ds.ReadStream(img.Hash.String())
		defer rs.Close()
		if err != nil {
			return err
		}
		if err := ptar.ExtractTar(tar.NewReader(rs), dir, true, sliceToMap(img.im.PathWhitelist)); err != nil {
			return fmt.Errorf("error extracting ACI: %v", err)
		}
		// If the image is an a previous level then apply PathWhiteList
		if img.Level < prevlevel {
			m := sliceToMap(img.im.PathWhitelist)
			rootfs := filepath.Join(dir, "rootfs/")
			err = filepath.Walk(rootfs, func(path string, info os.FileInfo, err error) error {

				relpath, err := filepath.Rel(rootfs, path)
				if err != nil {
					return err
				}
				// Ignore directories as if a file inside it is
				// in the pathWhiteList but its parent
				// directories are not in the pathWhiteList we
				// assume that they should be created.
				// Later we will remove empty directories not in the pathWhiteList
				if info.IsDir() {
					return nil
				}
				if _, ok := m[relpath]; !ok {
					err := os.Remove(path)
					if err != nil {
						return err
					}
				}
				return nil
			})

			if err != nil {
				return fmt.Errorf("build: Error walking rootfs: %v", err)
			}

			removeEmptyDirs(rootfs, rootfs, m)

			prevlevel = img.Level
		}
	}
	return nil
}

func removeEmptyDirs(rootfs string, dir string, pathWhitelistMap map[string]uint8) error {
	dirs, err := getDirectories(dir)
	if err != nil {
		return err
	}
	for _, dir := range dirs {
		removeEmptyDirs(rootfs, dir, pathWhitelistMap)
	}
	f, err := os.Open(dir)
	if err != nil {
		return err
	}
	names, err := f.Readdirnames(-1)
	f.Close()
	if err != nil {
		return err
	}
	if len(names) == 0 {
		relpath, err := filepath.Rel(rootfs, dir)
		if err != nil {
			return err
		}
		if _, ok := pathWhitelistMap[relpath]; !ok {
			err := os.Remove(dir)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func getDirectories(dir string) ([]string, error) {

	f, err := os.Open(dir)
	if err != nil {
		return nil, err
	}
	infos, err := f.Readdir(-1)
	f.Close()
	if err != nil {
		return nil, err
	}

	dirs := []string{}
	for _, info := range infos {
		if info.IsDir() {
			dirs = append(dirs, filepath.Join(dir, info.Name()))
		}
	}
	return dirs, nil
}

func sliceToMap(slice []string) map[string]uint8 {
	m := make(map[string]uint8, len(slice))
	for _, v := range slice {
		relpath, _ := filepath.Rel("/", v)
		m[relpath] = 1
	}
	return m
}

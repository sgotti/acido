package acirenderer

import (
	"archive/tar"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/appc/spec/schema"
	"github.com/appc/spec/schema/types"
	"github.com/coreos/rocket/cas"
)

type testTarEntry struct {
	header   *tar.Header
	contents string
}

func newTestTar(entries []*testTarEntry) (string, error) {
	tmpdir, err := ioutil.TempDir("", "test-tar")
	if err != nil {
		return "", err
	}
	t, err := os.OpenFile(filepath.Join(tmpdir, "tar.tar"), os.O_RDWR|os.O_CREATE|os.O_EXCL, 0600)
	defer t.Close()
	tw := tar.NewWriter(t)
	for _, entry := range entries {
		if err := tw.WriteHeader(entry.header); err != nil {
			return "", err
		}
		if _, err := io.WriteString(tw, entry.contents); err != nil {
			return "", err
		}
	}
	if err := tw.Close(); err != nil {
		return "", err
	}
	return t.Name(), nil
}

type fileInfo struct {
	path     string
	typeflag byte
	size     int64
}

func newTestAci(entries []*testTarEntry, ds *cas.Store) (string, error) {
	testTarPath, err := newTestTar(entries)
	if err != nil {
		return "", err
	}
	containerTar, err := os.Open(testTarPath)
	defer containerTar.Close()

	if err != nil {
		return "", err
	}

	tmp := types.NewHashSHA256([]byte(testTarPath)).String()
	hash, err := ds.WriteACI(tmp, containerTar)
	if err != nil {
		return "", err
	}

	return hash, nil
}

func TestRenderImage(t *testing.T) {
	storedir, err := ioutil.TempDir("", "storedir")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	ds := cas.NewStore(storedir)

	// Test an image without pathWhiteList
	imj := []byte(`
		{
		    "acKind": "ImageManifest",
		    "acVersion": "0.1.1",
		    "name": "example.com/test01"
		}
	`)

	entries := []*testTarEntry{
		{
			contents: string(imj),
			header: &tar.Header{
				Name: "manifest",
				Size: int64(len(imj)),
			},
		},
		{
			contents: "hello",
			header: &tar.Header{
				Name: "rootfs/a/file01.txt",
				Size: 5,
			},
		},
		{
			contents: "hello",
			header: &tar.Header{
				Name: "rootfs/a/file02.txt",
				Size: 5,
			},
		},
		{
			contents: "hello",
			header: &tar.Header{
				Name: "rootfs/a/file03.txt",
				Size: 5,
			},
		},
		{
			header: &tar.Header{
				Name:     "rootfs/b/link01.txt",
				Linkname: "file01.txt",
				Typeflag: tar.TypeSymlink,
			},
		},
		{
			contents: "hello",
			header: &tar.Header{
				Name: "rootfs/c/file04.txt",
				Size: 5,
			},
		},
	}

	expectedFiles := []*fileInfo{
		&fileInfo{path: "manifest", typeflag: tar.TypeReg},
		&fileInfo{path: "rootfs", typeflag: tar.TypeDir},
		&fileInfo{path: "rootfs/a", typeflag: tar.TypeDir},
		&fileInfo{path: "rootfs/a/file01.txt", typeflag: tar.TypeReg, size: 5},
		&fileInfo{path: "rootfs/a/file02.txt", typeflag: tar.TypeReg, size: 5},
		&fileInfo{path: "rootfs/a/file03.txt", typeflag: tar.TypeReg, size: 5},
		&fileInfo{path: "rootfs/b", typeflag: tar.TypeDir},
		&fileInfo{path: "rootfs/b/link01.txt", typeflag: tar.TypeSymlink},
		&fileInfo{path: "rootfs/c", typeflag: tar.TypeDir},
		&fileInfo{path: "rootfs/c/file04.txt", typeflag: tar.TypeReg, size: 5},
	}

	hash1, err := newTestAci(entries, ds)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	err = checkRenderImage(hash1, expectedFiles, ds)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Test an image with pathWhiteList excluding a file provided by the same image (strange but it can happen)
	// rootfs/a/file03.txt is excluded
	imj = []byte(`
		{
		    "acKind": "ImageManifest",
		    "acVersion": "0.1.1",
		    "name": "example.com/test02",
		    "pathWhitelist" : [ "/a/file01.txt", "/a/file02.txt", "/b/link01.txt", "/c/file04.txt" ]
		}
	`)

	entries = []*testTarEntry{
		{
			contents: string(imj),
			header: &tar.Header{
				Name: "manifest",
				Size: int64(len(imj)),
			},
		},
		{
			contents: "hello",
			header: &tar.Header{
				Name: "rootfs/a/file01.txt",
				Size: 5,
			},
		},
		{
			contents: "hello",
			header: &tar.Header{
				Name: "rootfs/a/file02.txt",
				Size: 5,
			},
		},
		{
			contents: "hellohello",
			header: &tar.Header{
				Name: "rootfs/a/file03.txt",
				Size: 10,
			},
		},
		{
			header: &tar.Header{
				Name:     "rootfs/b/link01.txt",
				Linkname: "file01.txt",
				Typeflag: tar.TypeSymlink,
			},
		},
		{
			contents: "hello",
			header: &tar.Header{
				Name: "rootfs/c/file04.txt",
				Size: 5,
			},
		},
	}

	expectedFiles = []*fileInfo{
		&fileInfo{path: "manifest", typeflag: tar.TypeReg},
		&fileInfo{path: "rootfs", typeflag: tar.TypeDir},
		&fileInfo{path: "rootfs/a", typeflag: tar.TypeDir},
		&fileInfo{path: "rootfs/a/file01.txt", typeflag: tar.TypeReg, size: 5},
		&fileInfo{path: "rootfs/a/file02.txt", typeflag: tar.TypeReg, size: 5},
		&fileInfo{path: "rootfs/b", typeflag: tar.TypeDir},
		&fileInfo{path: "rootfs/b/link01.txt", typeflag: tar.TypeSymlink},
		&fileInfo{path: "rootfs/c", typeflag: tar.TypeDir},
		&fileInfo{path: "rootfs/c/file04.txt", typeflag: tar.TypeReg, size: 5},
	}

	hash2, err := newTestAci(entries, ds)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	err = checkRenderImage(hash2, expectedFiles, ds)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Test an image with a pathwhitelist and 1 dep on an image without pathWhiteList

	// "/c/" is en empty dir to keep
	imj = []byte(`
		{
		    "acKind": "ImageManifest",
		    "acVersion": "0.1.1",
		    "name": "example.com/test03",
		    "pathWhitelist" : [ "/a/file01.txt", "/a/file02.txt", "/b/link01.txt", "/c/", "/d/file05.txt" ]
		}
	`)
	var im schema.ImageManifest
	im.UnmarshalJSON([]byte(imj))

	h1, _ := types.NewHash(hash1)
	im.Dependencies = types.Dependencies{
		types.Dependency{
			Name: "example.com/test01",
			Hash: *h1},
	}
	imj, err = im.MarshalJSON()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	entries = []*testTarEntry{
		{
			contents: string(imj),
			header: &tar.Header{
				Name: "manifest",
				Size: int64(len(imj)),
			},
		},
		// Updated file
		{
			contents: "hellohello",
			header: &tar.Header{
				Name: "rootfs/a/file01.txt",
				Size: 10,
			},
		},
		// rootfs/a/file02.txt unchanged
		// rootfs/a/file03.txt removed (see pathWhiteList)
		{
			header: &tar.Header{
				Name:     "rootfs/b/link01.txt",
				Linkname: "file01.txt",
				Typeflag: tar.TypeSymlink,
			},
		},
		// New file
		{
			contents: "hello",
			header: &tar.Header{
				Name: "rootfs/d/file05.txt",
				Size: 5,
			},
		},
	}

	expectedFiles = []*fileInfo{
		&fileInfo{path: "manifest", typeflag: tar.TypeReg},
		&fileInfo{path: "rootfs", typeflag: tar.TypeDir},
		&fileInfo{path: "rootfs/a", typeflag: tar.TypeDir},
		&fileInfo{path: "rootfs/b", typeflag: tar.TypeDir},
		&fileInfo{path: "rootfs/a/file01.txt", typeflag: tar.TypeReg, size: 10},
		&fileInfo{path: "rootfs/a/file02.txt", typeflag: tar.TypeReg, size: 5},
		&fileInfo{path: "rootfs/b/link01.txt", typeflag: tar.TypeSymlink},
		&fileInfo{path: "rootfs/c", typeflag: tar.TypeDir},
		&fileInfo{path: "rootfs/d", typeflag: tar.TypeDir},
		&fileInfo{path: "rootfs/d/file05.txt", typeflag: tar.TypeReg, size: 5},
	}

	hash3, err := newTestAci(entries, ds)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	err = checkRenderImage(hash3, expectedFiles, ds)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Test an image without a pathwhitelist and 1 dep on an image with pathWhiteList
	imj = []byte(`
		{
		    "acKind": "ImageManifest",
		    "acVersion": "0.1.1",
		    "name": "example.com/test04"
		}
	`)
	err = im.UnmarshalJSON([]byte(imj))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	h2, _ := types.NewHash(hash2)
	im.Dependencies = types.Dependencies{
		types.Dependency{
			Name: "example.com/test02",
			Hash: *h2},
	}
	imj, err = im.MarshalJSON()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	entries = []*testTarEntry{
		{
			contents: string(imj),
			header: &tar.Header{
				Name: "manifest",
				Size: int64(len(imj)),
			},
		},
		// Updated file
		{
			contents: "hellohello",
			header: &tar.Header{
				Name: "rootfs/a/file01.txt",
				Size: 10,
			},
		},
		// rootfs/a/file02.txt unchanged
		{
			header: &tar.Header{
				Name:     "rootfs/b/link01.txt",
				Linkname: "file01.txt",
				Typeflag: tar.TypeSymlink,
			},
		},
		// New file
		{
			contents: "hello",
			header: &tar.Header{
				Name: "rootfs/d/file05.txt",
				Size: 5,
			},
		},
		// New Empty dir
		{
			header: &tar.Header{
				Name:     "rootfs/e",
				Typeflag: tar.TypeDir,
			},
		},
	}

	expectedFiles = []*fileInfo{
		&fileInfo{path: "manifest", typeflag: tar.TypeReg},
		&fileInfo{path: "rootfs", typeflag: tar.TypeDir},
		&fileInfo{path: "rootfs/a", typeflag: tar.TypeDir},
		&fileInfo{path: "rootfs/b", typeflag: tar.TypeDir},
		&fileInfo{path: "rootfs/a/file01.txt", typeflag: tar.TypeReg, size: 10},
		&fileInfo{path: "rootfs/a/file02.txt", typeflag: tar.TypeReg, size: 5},
		&fileInfo{path: "rootfs/b/link01.txt", typeflag: tar.TypeSymlink},
		&fileInfo{path: "rootfs/c", typeflag: tar.TypeDir},
		&fileInfo{path: "rootfs/c/file04.txt", typeflag: tar.TypeReg, size: 5},
		&fileInfo{path: "rootfs/d", typeflag: tar.TypeDir},
		&fileInfo{path: "rootfs/d/file05.txt", typeflag: tar.TypeReg, size: 5},
		&fileInfo{path: "rootfs/e", typeflag: tar.TypeDir},
	}

	hash4, err := newTestAci(entries, ds)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	err = checkRenderImage(hash4, expectedFiles, ds)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Test an image with a pathwhitelist and 1 dep on an image with pathWhiteList
	// "/c/" is en empty dir to keep
	imj = []byte(`
		{
		    "acKind": "ImageManifest",
		    "acVersion": "0.1.1",
		    "name": "example.com/test05",
		    "pathWhitelist" : [ "/a/file01.txt", "/a/file02.txt", "/a/file03.txt", "/b/link01.txt", "/c/", "/d/file05.txt", "/e/" ]
		}
	`)
	im.UnmarshalJSON([]byte(imj))

	h2, _ = types.NewHash(hash2)
	im.Dependencies = types.Dependencies{
		types.Dependency{
			Name: "example.com/test02",
			Hash: *h2},
	}
	imj, err = im.MarshalJSON()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	entries = []*testTarEntry{
		{
			contents: string(imj),
			header: &tar.Header{
				Name: "manifest",
				Size: int64(len(imj)),
			},
		},
		// Updated file
		{
			contents: "hellohello",
			header: &tar.Header{
				Name: "rootfs/a/file01.txt",
				Size: 10,
			},
		},
		// rootfs/a/file02.txt unchanged
		{
			header: &tar.Header{
				Name:     "rootfs/b/link01.txt",
				Linkname: "file01.txt",
				Typeflag: tar.TypeSymlink,
			},
		},
		// This file was removed from the dep's pathWhiteList and now readded
		{
			contents: "hellohellohello",
			header: &tar.Header{
				Name: "rootfs/a/file03.txt",
				Size: 15,
			},
		},
		// New file
		{
			contents: "hello",
			header: &tar.Header{
				Name: "rootfs/d/file05.txt",
				Size: 5,
			},
		},
		// New Empty dir
		{
			header: &tar.Header{
				Name:     "rootfs/e",
				Typeflag: tar.TypeDir,
			},
		},
	}

	expectedFiles = []*fileInfo{
		&fileInfo{path: "manifest", typeflag: tar.TypeReg},
		&fileInfo{path: "rootfs", typeflag: tar.TypeDir},
		&fileInfo{path: "rootfs/a", typeflag: tar.TypeDir},
		&fileInfo{path: "rootfs/b", typeflag: tar.TypeDir},
		&fileInfo{path: "rootfs/a/file01.txt", typeflag: tar.TypeReg, size: 10},
		&fileInfo{path: "rootfs/a/file02.txt", typeflag: tar.TypeReg, size: 5},
		&fileInfo{path: "rootfs/a/file03.txt", typeflag: tar.TypeReg, size: 15},
		&fileInfo{path: "rootfs/b/link01.txt", typeflag: tar.TypeSymlink},
		&fileInfo{path: "rootfs/c", typeflag: tar.TypeDir},
		&fileInfo{path: "rootfs/d", typeflag: tar.TypeDir},
		&fileInfo{path: "rootfs/d/file05.txt", typeflag: tar.TypeReg, size: 5},
		&fileInfo{path: "rootfs/e", typeflag: tar.TypeDir},
	}

	hash5, err := newTestAci(entries, ds)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	err = checkRenderImage(hash5, expectedFiles, ds)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Test an image with a pathwhitelist and 2 deps (first without pathWhiteList and the second with pathWhiteList)
	// "/c/" is en empty dir to keep
	imj = []byte(`
		{
		    "acKind": "ImageManifest",
		    "acVersion": "0.1.1",
		    "name": "example.com/test06",
		    "pathWhitelist" : [ "/a/file01.txt", "/a/file02.txt", "/a/file03.txt", "/b/link01.txt", "/c/", "/d/file05.txt", "/e/" ]
		}
	`)
	im.UnmarshalJSON([]byte(imj))

	h1, _ = types.NewHash(hash1)
	h2, _ = types.NewHash(hash2)
	im.Dependencies = types.Dependencies{
		types.Dependency{
			Name: "example.com/test01",
			Hash: *h1},
		types.Dependency{
			Name: "example.com/test02",
			Hash: *h2},
	}
	imj, err = im.MarshalJSON()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	entries = []*testTarEntry{
		{
			contents: string(imj),
			header: &tar.Header{
				Name: "manifest",
				Size: int64(len(imj)),
			},
		},
		// Updated file
		{
			contents: "hellohello",
			header: &tar.Header{
				Name: "rootfs/a/file01.txt",
				Size: 10,
			},
		},
		// rootfs/a/file02.txt unchanged
		{
			header: &tar.Header{
				Name:     "rootfs/b/link01.txt",
				Linkname: "file01.txt",
				Typeflag: tar.TypeSymlink,
			},
		},
		// New file
		{
			contents: "hello",
			header: &tar.Header{
				Name: "rootfs/d/file05.txt",
				Size: 5,
			},
		},
		// New Empty dir
		{
			header: &tar.Header{
				Name:     "rootfs/e",
				Typeflag: tar.TypeDir,
			},
		},
	}

	expectedFiles = []*fileInfo{
		&fileInfo{path: "manifest", typeflag: tar.TypeReg},
		&fileInfo{path: "rootfs", typeflag: tar.TypeDir},
		&fileInfo{path: "rootfs/a", typeflag: tar.TypeDir},
		&fileInfo{path: "rootfs/b", typeflag: tar.TypeDir},
		&fileInfo{path: "rootfs/a/file01.txt", typeflag: tar.TypeReg, size: 10},
		&fileInfo{path: "rootfs/a/file02.txt", typeflag: tar.TypeReg, size: 5},
		// rootfs/a/file03.txt should be the one from the first dep. The second dep doesn't have it in the pathWhitelist but should be removed as they are on the same level.
		&fileInfo{path: "rootfs/a/file03.txt", typeflag: tar.TypeReg, size: 5},
		&fileInfo{path: "rootfs/b/link01.txt", typeflag: tar.TypeSymlink},
		&fileInfo{path: "rootfs/c", typeflag: tar.TypeDir},
		&fileInfo{path: "rootfs/d", typeflag: tar.TypeDir},
		&fileInfo{path: "rootfs/d/file05.txt", typeflag: tar.TypeReg, size: 5},
		&fileInfo{path: "rootfs/e", typeflag: tar.TypeDir},
	}

	hash6, err := newTestAci(entries, ds)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	err = checkRenderImage(hash6, expectedFiles, ds)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

}

func checkRenderImage(hash string, expectedFiles []*fileInfo, ds *cas.Store) error {

	imagedir, err := ioutil.TempDir("", "imagedir")
	if err != nil {
		return err
	}
	err = RenderImage(hash, imagedir, ds)
	if err != nil {
		return err
	}
	err = checkExpectedFiles(imagedir, FISliceToMap(expectedFiles))
	if err != nil {
		return err
	}

	return nil

}
func checkExpectedFiles(dir string, expectedFiles map[string]*fileInfo) error {

	files := make(map[string]*fileInfo)
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		fm := info.Mode()
		if path == dir {
			return nil
		}
		relpath, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		switch {
		case fm.IsRegular():
			files[relpath] = &fileInfo{path: relpath, typeflag: tar.TypeReg, size: info.Size()}
		case info.IsDir():
			files[relpath] = &fileInfo{path: relpath, typeflag: tar.TypeDir}
		case fm&os.ModeSymlink != 0:
			files[relpath] = &fileInfo{path: relpath, typeflag: tar.TypeSymlink}
		default:
			return fmt.Errorf("not handled file mode %v", fm)
		}

		return nil
	})
	if err != nil {
		return err
	}

	for _, ef := range expectedFiles {
		_, ok := files[ef.path]
		if !ok {
			return fmt.Errorf("Expected file \"%s\" not in files", ef.path)
		}

	}

	for _, file := range files {
		ef, ok := expectedFiles[file.path]
		if !ok {
			return fmt.Errorf("file \"%s\" not in expectedFiles", file.path)
		}
		if ef.typeflag != file.typeflag {
			return fmt.Errorf("file \"%s\": file type differs: found %d, wanted: %d", file.path, file.typeflag, ef.typeflag)
		}
		if ef.typeflag == tar.TypeReg && file.path != "manifest" {
			if ef.size != file.size {
				return fmt.Errorf("file \"%s\": size differs: found %d, wanted: %d", file.path, file.size, ef.size)
			}
		}

	}
	return nil
}

func FISliceToMap(slice []*fileInfo) map[string]*fileInfo {
	fim := make(map[string]*fileInfo, len(slice))
	for _, fi := range slice {
		fim[fi.path] = fi
	}
	return fim
}

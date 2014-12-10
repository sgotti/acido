package main

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/appc/spec/aci"
	"github.com/appc/spec/pkg/tarheader"
	"github.com/coreos/fleet/log"
	"github.com/coreos/rocket/cas"
	"github.com/sgotti/acido/acirenderer"
	"github.com/sgotti/acido/fsdiffer"
	"github.com/sgotti/acido/util"
)

var (
	buildImageManifest string
	buildBaseImageHash string

	cmdBuild = &Command{
		Name:        "build",
		Summary:     "Build a previously prepared image with only the differences from a base and the new image",
		Usage:       "IMAGEFS OUTPUTFILE",
		Description: `IMAGEFS is the directory containing the image data.`,
		Run:         runBuild,
	}
)

func init() {
	commands = append(commands, cmdBuild)
}

func builder(root string, files []string, aw aci.ArchiveWriter) error {
	// cache of inode -> filepath, used to leverage hard links in the archive
	inos := map[uint64]string{}
	for _, path := range files {
		relpath, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		relpath = filepath.Join("rootfs/", relpath)

		info, err := os.Lstat(path)
		if err != nil {
			return err
		}
		var file *os.File
		link := ""
		switch info.Mode() & os.ModeType {
		default:
			file, err = os.Open(path)
			if err != nil {
				return err
			}
			defer file.Close()
		case os.ModeSymlink:
			target, err := os.Readlink(path)
			if err != nil {
				return err
			}
			link = target
		}

		hdr, err := tar.FileInfoHeader(info, link)
		if err != nil {
			panic(err)
		}
		// Because os.FileInfo's Name method returns only the base
		// name of the file it describes, it may be necessary to
		// modify the Name field of the returned header to provide the
		// full path name of the file.
		hdr.Name = relpath
		tarheader.Populate(hdr, info, inos)
		// If the file is a hard link we don't need the contents
		if hdr.Typeflag == tar.TypeLink {
			hdr.Size = 0
			file = nil
		}
		aw.AddFile(relpath, hdr, file)

	}
	return nil
}

func build(args []string) error {
	if len(args) != 2 {
		return fmt.Errorf("build: Must provide imagefs and output file")
	}

	ds := cas.NewStore(globalFlags.Dir)

	imagefs := args[0]
	out := args[1]
	log.Infof("out file: %s", out)
	tmpdir, err := ioutil.TempDir(globalFlags.WorkDir, "")
	if err != nil {
		return err
	}
	log.V(1).Infof("tmpdir: %s", tmpdir)
	os.Mkdir(filepath.Join(tmpdir, "/rootfs"), 0755)

	im, err := util.LoadImageManifest(filepath.Join(imagefs, "manifest"))
	if err != nil {
		return err
	}
	dependencies := im.Dependencies
	for _, d := range dependencies {
		//if _, ok := seenImages[d.Hash]
		log.V(1).Infof("Dependency Hash: %s\n", d.Hash)
		if d.Hash.Val != "" {
			err = acirenderer.RenderImage(d.Hash.String(), tmpdir, ds)
			if err != nil {
				return err
			}
		}
	}

	A := filepath.Join(tmpdir, "/rootfs")
	B := filepath.Join(imagefs, "/rootfs")
	changes, err := fsdiffer.FSDiff(A, B)

	if err != nil {
		return err
	}

	// Create a file list with all the Added and Modified files
	files := []string{}
	for _, change := range changes {
		if change.ChangeType == fsdiffer.Added || change.ChangeType == fsdiffer.Modified {
			files = append(files, filepath.Join(B, change.Path))
		}
	}

	pathWhitelist := []string{}
	err = filepath.Walk(B, func(path string, info os.FileInfo, err error) error {
		relpath, err := filepath.Rel(B, path)
		if err != nil {
			return err
		}
		pathWhitelist = append(pathWhitelist, filepath.Join("/", relpath))
		return nil
	})

	if err != nil {
		return fmt.Errorf("build: Error walking rootfs: %v", err)
	}

	//log.Infof("changes: %v\n", changes)
	//log.Infof("files: %v\n", files)
	//log.Infof("pathWhitelist: %v\n", pathWhitelist)

	mode := os.O_CREATE | os.O_WRONLY | os.O_TRUNC
	fh, err := os.OpenFile(out, mode, 0644)
	if err != nil {
		if os.IsExist(err) {
			return fmt.Errorf("build: Target file exists (try --overwrite)")
		} else {
			return fmt.Errorf("build: Unable to open target %s: %v", out, err)
		}
	}

	gw := gzip.NewWriter(fh)
	tr := tar.NewWriter(gw)

	defer func() {
		tr.Close()
		gw.Close()
		fh.Close()
	}()

	im.PathWhitelist = pathWhitelist

	aw := aci.NewImageWriter(*im, tr)

	err = builder(B, files, aw)
	if err != nil {
		return err
	}

	err = aw.Close()
	if err != nil {
		return fmt.Errorf("build: Unable to close Fileset image %s: %v", out, err)
	}

	return nil
}
func runBuild(args []string) (exit int) {
	err := build(args)
	if err != nil {
		log.Errorf("error: %v", err)
		return 1
	}
	return 0
}

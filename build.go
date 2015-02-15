package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/sgotti/acido/cas"
	"github.com/sgotti/acido/pkg/aci"
	"github.com/sgotti/acido/util"

	"github.com/coreos/fleet/log"
	"github.com/sgotti/acibuilder"
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
	log.Debugf("tmpdir: %s", tmpdir)
	os.Mkdir(filepath.Join(tmpdir, "/rootfs"), 0755)

	im, err := util.LoadImageManifest(filepath.Join(imagefs, "manifest"))
	if err != nil {
		return err
	}
	dependencies := im.Dependencies
	switch s := len(dependencies); {
	case s > 1 || s < 1:
		return fmt.Errorf("build: exactly one dependency is required")

	}

	dependency := dependencies[0]
	log.Debugf("Dependency ImageID: %s\n", dependency.ImageID)
	if !dependency.ImageID.Empty() {
		err := aci.RenderACIWithImageID(*dependency.ImageID, tmpdir, ds)
		if err != nil {
			return err
		}

	}

	mode := os.O_CREATE | os.O_WRONLY | os.O_TRUNC
	fh, err := os.OpenFile(out, mode, 0644)
	if err != nil {
		if os.IsExist(err) {
			return fmt.Errorf("build: Target file exists (try --overwrite)")
		} else {
			return fmt.Errorf("build: Unable to open target %s: %v", out, err)
		}
	}
	defer func() {
		fh.Close()
	}()

	ACIBuilder := acibuilder.NewDiffACIBuilder(tmpdir, imagefs)

	err = ACIBuilder.Build(*im, fh)
	if err != nil {
		return err
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

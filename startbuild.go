package main

import (
	"io/ioutil"
	"path/filepath"

	"github.com/appc/spec/schema"
	"github.com/appc/spec/schema/types"
	"github.com/coreos/fleet/log"
	"github.com/coreos/rocket/cas"
	"github.com/sgotti/acido/acirenderer"
	"github.com/sgotti/acido/util"
)

var (
	cmdStartBuild = &Command{
		Name:        "startbuild",
		Summary:     "Prepare an image for future build. If base image is specified it's extracted (satisfying all its dependencies) and a base app-manifest with dependencies set to the baseimage is created",
		Usage:       "BASEIMAGEHASH...",
		Description: `BASEIMAGEHASH hash of base image (it must exists in the store or it sould be imported with the \"import\" command.`,
		Run:         runStartBuild,
	}
)

func init() {
	commands = append(commands, cmdStartBuild)
}

func startBuild(args []string) error {
	ds := cas.NewStore(globalFlags.Dir)

	baseImageIDStr := args[0]
	baseImageID, err := types.NewHash(baseImageIDStr)
	if err != nil {
		return err
	}

	tmpdir, err := ioutil.TempDir(globalFlags.WorkDir, "")
	if err != nil {
		return err
	}
	log.Debugf("tmpdir: %s", tmpdir)
	baseim, err := util.GetImageManifest(baseImageID, ds)
	if err != nil {
		return err
	}

	err = acirenderer.RenderImage(baseImageIDStr, tmpdir, ds)
	if err != nil {
		return err
	}
	log.Infof("Image extracted to %s", tmpdir)

	version, _ := types.NewSemVer("0.1.0")
	im := schema.ImageManifest{
		ACKind:    "ImageManifest",
		ACVersion: *version,
		Name:      "example.com/changeme",
		Dependencies: types.Dependencies{
			types.Dependency{
				App:     baseim.Name,
				ImageID: baseImageID,
			},
		},
	}

	out, err := im.MarshalJSON()
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(filepath.Join(tmpdir, "manifest"), out, 0644)
	if err != nil {
		return err
	}

	return nil
}

func runStartBuild(args []string) (exit int) {
	err := startBuild(args)
	if err != nil {
		log.Errorf("error: %v", err)
		return 1
	}
	return 0
}

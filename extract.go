package main

import (
	"io/ioutil"

	"github.com/coreos/fleet/log"
	"github.com/coreos/rocket/cas"
	"github.com/sgotti/acido/acirenderer"
)

var (
	cmdExtract = &Command{
		Name:        "extract",
		Summary:     "Extracts an image already imported in the store (satisfying all its dependencies)",
		Usage:       "IMAGEHASH",
		Description: `IMAGEHASH hash of base image (it must exists in the store or it sould be imported with the \"import\" command.`,
		Run:         runExtract,
	}
)

func init() {
	commands = append(commands, cmdExtract)
}

func runExtract(args []string) (exit int) {
	ds := cas.NewStore(globalFlags.Dir)

	hash := args[0]

	tmpdir, err := ioutil.TempDir(globalFlags.WorkDir, "")
	if err != nil {
		log.Errorf("error: %v", err)
		return 1
	}
	log.Debugf("tmpdir: %s", tmpdir)
	err = acirenderer.RenderImage(hash, tmpdir, ds)
	if err != nil {
		log.Errorf("error: %v", err)
		return 1
	}
	log.Infof("Image extracted to %s", tmpdir)
	return 0
}

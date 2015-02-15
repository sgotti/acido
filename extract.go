package main

import (
	"io/ioutil"

	"github.com/sgotti/acido/cas"
	"github.com/sgotti/acido/pkg/aci"

	"github.com/appc/spec/schema/types"
	"github.com/coreos/fleet/log"
)

var (
	cmdExtract = &Command{
		Name:        "extract",
		Summary:     "Extracts an image already imported in the store (satisfying all its dependencies)",
		Usage:       "IMAGEID",
		Description: `IMAGEID imageID of base image (it must exists in the store or it sould be imported with the \"import\" command.`,
		Run:         runExtract,
	}
)

func init() {
	commands = append(commands, cmdExtract)
}

func runExtract(args []string) (exit int) {
	ds := cas.NewStore(globalFlags.Dir)

	imageIDStr := args[0]
	imageID, err := types.NewHash(imageIDStr)
	if err != nil {
		log.Errorf("error: %v", err)
		return 1
	}

	tmpdir, err := ioutil.TempDir(globalFlags.WorkDir, "")
	if err != nil {
		log.Errorf("error: %v", err)
		return 1
	}
	log.Debugf("tmpdir: %s", tmpdir)

	err = aci.RenderACIWithImageID(*imageID, tmpdir, ds)
	if err != nil {
		log.Errorf("error: %v", err)
		return 1
	}
	log.Infof("Image extracted to %s", tmpdir)
	return 0
}

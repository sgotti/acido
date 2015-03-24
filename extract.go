package main

import (
	"io/ioutil"

	"github.com/sgotti/acido/util"

	"github.com/sgotti/acido/Godeps/_workspace/src/github.com/appc/spec/schema/types"
	"github.com/sgotti/acido/Godeps/_workspace/src/github.com/coreos/fleet/log"
	"github.com/sgotti/acido/Godeps/_workspace/src/github.com/coreos/rocket/cas"
	"github.com/sgotti/acido/Godeps/_workspace/src/github.com/coreos/rocket/pkg/aci"
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
	ds, err := cas.NewStore(globalFlags.Dir)
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

	imageIDStr := args[0]
	key, err := util.KeyFromArg(imageIDStr, ds)
	if err != nil {
		log.Errorf("error: %v", err)
		return 1
	}
	log.Debugf("key: %s", key)

	imageID, err := types.NewHash(key)
	if err != nil {
		log.Errorf("error: %v", err)
		return 1
	}

	err = aci.RenderACIWithImageID(*imageID, tmpdir, ds)
	if err != nil {
		log.Errorf("error: %v", err)
		return 1
	}
	log.Infof("Image extracted to %s", tmpdir)
	return 0
}

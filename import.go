package main

import (
	"os"

	"github.com/appc/spec/schema/types"
	"github.com/coreos/fleet/log"
	"github.com/coreos/rocket/cas"
)

var (
	cmdImport = &Command{
		Name:        "import",
		Summary:     "Import image(s) in the rocket cas",
		Usage:       "IMAGE...",
		Description: `IMAGE should be a string referencing an image as a local file on disk.`,
		Run:         runImport,
	}
)

func init() {
	commands = append(commands, cmdImport)
}

func runImport(args []string) (exit int) {
	ds, err := cas.NewStore(globalFlags.Dir)
	if err != nil {
		log.Errorf("error: %v", err)
		return 1
	}

	for _, img := range args {
		// import the local file if it exists
		file, err := os.Open(img)
		if err != nil {
			log.Errorf("error: %v", err)
			return 1
		}
		key, err := ds.WriteACI(file)
		file.Close()
		if err != nil {
			log.Errorf("error importing %s: %v", img, err)
			return 1
		}
		h, err := types.NewHash(key)
		if err != nil {
			// should never happen
			panic(err)
		}
		log.Infof("image: %s, hash: %s\n", img, h)
		continue
	}

	return 0
}

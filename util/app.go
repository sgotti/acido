package util

import (
	"fmt"

	"github.com/sgotti/acido/Godeps/_workspace/src/github.com/appc/spec/discovery"
	"github.com/sgotti/acido/Godeps/_workspace/src/github.com/appc/spec/schema/types"
	"github.com/sgotti/acido/Godeps/_workspace/src/github.com/coreos/rocket/cas"
)

func AppLabelToLabels(labelsMap map[string]string) (types.Labels, error) {
	labels := types.Labels{}
	for n, v := range labelsMap {
		name, err := types.NewACName(n)
		if err != nil {
			return nil, err
		}
		labels = append(labels, types.Label{Name: *name, Value: v})
	}
	return labels, nil
}

func KeyFromArg(arg string, ds *cas.Store) (string, error) {
	var key string
	// check if it is a valid hash, if so let it pass through
	_, err := types.NewHash(arg)
	if err == nil {
		key, err = ds.ResolveKey(arg)
		if err != nil {
			return "", fmt.Errorf("could not resolve key: %v", err)
		}
	} else {
		app, err := discovery.NewAppFromString(arg)
		if err != nil {
			return "", err
		}
		labels, err := AppLabelToLabels(app.Labels)
		if err != nil {
			return "", err
		}
		key, err = ds.GetACI(app.Name, labels)
		if err != nil {
			return "", err
		}
	}
	return key, nil
}

package release

import (
	"fmt"
	"regexp"

	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/release"
)

func (rel *Release) List() (*release.Release, error) {
	client := action.NewList(rel.Cfg())
	client.Filter = fmt.Sprintf("^%s$", regexp.QuoteMeta(rel.Name()))

	result, err := client.Run()
	if err != nil {
		return nil, fmt.Errorf("failed to list release %s: %w", rel.Uniq(), err)
	}

	switch len(result) {
	case 0:
		return nil, ErrNotFound
	case 1:
		return result[0], nil
	default:
		return nil, ErrFoundMultiple
	}
}

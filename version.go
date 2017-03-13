package main

import (
	"github.com/Masterminds/semver"
)

var metadata string

func version() (string, error) {
	v, err := semver.NewVersion("0.3.0")
	if err != nil {
		return v.String(), err
	}

	vNext, err := v.SetMetadata(metadata)
	if err != nil {
		return vNext.String(), err
	}

	return vNext.String(), nil
}

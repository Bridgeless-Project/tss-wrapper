package utils

import (
	"github.com/pkg/errors"
	"github.com/tendermint/tendermint/libs/os"
)

func StoreCertificate(cert []byte, path string) error {
	found := os.FileExists(path)
	if found {
		// any action if file exists
		return nil
	}

	if err := os.WriteFile(path, cert, 0644); err != nil {
		return errors.Wrap(err, "failed to store certificate")
	}

	return nil
}

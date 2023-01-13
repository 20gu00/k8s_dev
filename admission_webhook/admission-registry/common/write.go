package common

import (
	"os"
)

func WriteFile(filePath string, bts []byte) error {
	f, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer f.Close()

	if _, err := f.Write(bts); err != nil {
		return err
	}

	return nil
}
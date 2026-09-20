package cli

import (
	"fmt"

	"github.com/hpcsc/emod/internal/version"
)

func RunVersion() error {
	_, err := fmt.Println(version.Current())
	return err
}

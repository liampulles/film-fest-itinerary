package commands

import "fmt"

func RunDIFF2026(args []string) error {
	if len(args) != 0 {
		return fmt.Errorf("usage: filmscrapecli diff-2026")
	}

	return nil
}

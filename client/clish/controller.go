package clish

import "fmt"

type ShellController struct {
	Stage *ShellControllerStage `flagly:"handler"`
}

type ShellControllerStage struct {
}

func (*ShellControllerStage) FlaglyHandle(c Client) error {
	info, err := c.ShowControllerStage()
	if err != nil {
		return err
	}
	return fmt.Errorf("staging: %v\n", len(info))
}

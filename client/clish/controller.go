package clish

import "fmt"

type Controller struct {
	Stage *ControllerStage `flagly:"handler"`
}

type ControllerStage struct{}

func (*ControllerStage) FlaglyHandle(c Client) error {
	info, err := c.ShowControllerStage()
	if err != nil {
		return err
	}
	return fmt.Errorf("staging: %v", len(info))
}

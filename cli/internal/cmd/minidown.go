/*
Copyright (c) Edgeless Systems GmbH

SPDX-License-Identifier: AGPL-3.0-only
*/

package cmd

import (
	"errors"
	"fmt"
	"os"

	"github.com/edgelesssys/constellation/v2/internal/cloud/cloudprovider"
	"github.com/edgelesssys/constellation/v2/internal/constants"
	"github.com/edgelesssys/constellation/v2/internal/file"
	"github.com/edgelesssys/constellation/v2/internal/state"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"go.uber.org/multierr"
)

func newMiniDownCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "down",
		Short: "Destroy a mini Constellation cluster",
		Long:  "Destroy a mini Constellation cluster.",
		Args:  cobra.ExactArgs(0),
		RunE:  runDown,
	}

	return cmd
}

func runDown(cmd *cobra.Command, args []string) error {
	if err := checkForMiniCluster(file.NewHandler(afero.NewOsFs())); err != nil {
		return fmt.Errorf("failed to destroy cluster: %w. Are you in the correct working directory?", err)
	}

	err := runTerminate(cmd, args)
	if removeErr := os.Remove(constants.MasterSecretFilename); removeErr != nil && !os.IsNotExist(removeErr) {
		err = multierr.Append(err, removeErr)
	}
	return err
}

func checkForMiniCluster(fileHandler file.Handler) error {
	var state state.ConstellationState
	if err := fileHandler.ReadJSON(constants.StateFilename, &state); err != nil {
		return err
	}
	if cloudprovider.FromString(state.CloudProvider) != cloudprovider.QEMU {
		return errors.New("cluster is not a QEMU based Constellation")
	}
	if state.Name != "mini" {
		return errors.New("cluster is not a mini Constellation cluster")
	}

	return nil
}
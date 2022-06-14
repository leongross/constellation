package cmd

import (
	"github.com/edgelesssys/constellation/internal/cloud/cloudprovider"
	"github.com/edgelesssys/constellation/internal/config"
	"github.com/edgelesssys/constellation/internal/constants"
	"github.com/edgelesssys/constellation/internal/file"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"github.com/talos-systems/talos/pkg/machinery/config/encoder"
)

func newConfigGenerateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "generate {aws|azure|gcp}",
		Short: "Generate a default configuration file",
		Long:  "Generate a default configuration file for your selected cloud provider.",
		Args: cobra.MatchAll(
			cobra.ExactArgs(1),
			isCloudProvider(0),
			warnAWS(0),
		),
		ValidArgsFunction: generateCompletion,
		RunE:              runConfigGenerate,
	}
	cmd.Flags().StringP("file", "f", constants.ConfigFilename, "path to output file, or '-' for stdout")

	return cmd
}

type generateFlags struct {
	file string
}

func runConfigGenerate(cmd *cobra.Command, args []string) error {
	fileHandler := file.NewHandler(afero.NewOsFs())
	provider := cloudprovider.FromString(args[0])
	return configGenerate(cmd, fileHandler, provider)
}

func configGenerate(cmd *cobra.Command, fileHandler file.Handler, provider cloudprovider.Provider) error {
	flags, err := parseGenerateFlags(cmd)
	if err != nil {
		return err
	}

	conf := config.Default()
	conf.RemoveProviderExcept(provider)

	if flags.file == "-" {
		content, err := encoder.NewEncoder(conf).Encode()
		if err != nil {
			return err
		}
		_, err = cmd.OutOrStdout().Write(content)
		return err
	}

	return fileHandler.WriteYAML(flags.file, conf, 0o644)
}

func parseGenerateFlags(cmd *cobra.Command) (generateFlags, error) {
	file, err := cmd.Flags().GetString("file")
	if err != nil {
		return generateFlags{}, err
	}
	return generateFlags{
		file: file,
	}, nil
}

// createCompletion handles the completion of the create command. It is frequently called
// while the user types arguments of the command to suggest completion.
func generateCompletion(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	switch len(args) {
	case 0:
		return []string{"aws", "gcp", "azure"}, cobra.ShellCompDirectiveNoFileComp
	default:
		return []string{}, cobra.ShellCompDirectiveError
	}
}
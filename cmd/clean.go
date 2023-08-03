package main

import (
	"context"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"go.uber.org/zap"
)

func cleanCmd() *cobra.Command {
	var c cleaner
	cmd := &cobra.Command{
		Use:   "clean",
		Short: "Clean the generated Go files",
		Run: func(cmd *cobra.Command, args []string) {
			if err := c.clean(cmd.Context()); err != nil {
				c.logger.Fatal(err)
			}
		},
	}
	c.addCLIFlags(cmd.Flags())
	return cmd
}

type cleaner struct {
	logger         *zap.SugaredLogger
	loggingOptions LoggingOptions
}

func (b *cleaner) addCLIFlags(fs *pflag.FlagSet) {
	b.loggingOptions.AddCLIFlags(fs)
}

func (b *cleaner) clean(ctx context.Context) error {
	b.logger = b.loggingOptions.MustCreateLogger()

	return os.RemoveAll(genRoot())
}

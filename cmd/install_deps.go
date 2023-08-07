package main

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"go.uber.org/zap"
)

func installDepsCmd() *cobra.Command {
	var i depsInstaller
	cmd := &cobra.Command{
		Use:   "install-deps",
		Short: "Install tool dependencies",
		Run: func(cmd *cobra.Command, args []string) {
			if err := i.installDeps(cmd.Context()); err != nil {
				i.logger.Fatal(err)
			}
		},
	}
	i.addCLIFlags(cmd.Flags())
	return cmd
}

type depsInstaller struct {
	logger         *zap.SugaredLogger
	loggingOptions LoggingOptions
}

func (l *depsInstaller) addCLIFlags(fs *pflag.FlagSet) {
	l.loggingOptions.AddCLIFlags(fs)
}

func (l *depsInstaller) installDeps(ctx context.Context) error {
	l.logger = l.loggingOptions.MustCreateLogger()

	args := []string{"list", "-f", `{{join .Imports " "}}`, "tools.go"}
	l.logger.Info("Getting dependency list")
	cmd := exec.CommandContext(ctx, "go", args...)
	cmd.Dir = filepath.Join(projectRoot(), "tools")
	cmd.Stderr = os.Stderr
	out, err := cmd.Output()
	if err != nil {
		return err
	}
	outString := strings.TrimRight(string(out), "\r\n")
	deps := strings.Split(outString, " ")

	args = append([]string{"install"}, deps...)
	l.logger.Infow("Installing dependencies", "deps", deps)
	cmd = exec.CommandContext(ctx, "go", args...)
	cmd.Dir = filepath.Join(projectRoot(), "tools")
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	return cmd.Run()
}

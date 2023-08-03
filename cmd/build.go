package main

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/otiai10/copy"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"go.uber.org/zap"
)

func buildCmd() *cobra.Command {
	var b builder
	cmd := &cobra.Command{
		Use:   "build",
		Short: "Generate Go from proto files",
		Run: func(cmd *cobra.Command, args []string) {
			if err := b.build(cmd.Context()); err != nil {
				b.logger.Fatal(err)
			}
		},
	}
	b.addCLIFlags(cmd.Flags())
	return cmd
}

type builder struct {
	logger         *zap.SugaredLogger
	loggingOptions LoggingOptions
}

func (b *builder) addCLIFlags(fs *pflag.FlagSet) {
	b.loggingOptions.AddCLIFlags(fs)
}

func (b *builder) build(ctx context.Context) error {
	b.logger = b.loggingOptions.MustCreateLogger()
	// First clean existing generated files
	if err := os.RemoveAll(genRoot()); err != nil {
		return err
	}
	protoFiles, err := findProtos()
	if err != nil {
		return err
	}

	tempDir, err := os.MkdirTemp("", "proto-build")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tempDir)

	// Go gRPC and proto
	baseArgs := []string{"--fatal_warnings", "-I", protoRoot()}
	args := append(baseArgs,
		"--go_out", tempDir,
		"--go_opt", "paths=source_relative",
		"--go-grpc_out", tempDir,
		"--go-grpc_opt", "paths=source_relative",
	)
	args = append(args, protoFiles...)

	b.logger.Infow("Running protoc", "args", args)
	cmd := exec.CommandContext(ctx, "protoc", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return err
	}

	// Go gRPC gateway
	args = append(baseArgs,
		"--grpc-gateway_out", tempDir,
		"--grpc-gateway_opt", "logtostderr=true",
		"--grpc-gateway_opt", "paths=source_relative",
		"--grpc-gateway_opt", "generate_unbound_methods=true",
	)
	args = append(args, protoFiles...)
	b.logger.Infow("Running protoc", "args", args)
	cmd = exec.CommandContext(ctx, "protoc", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return err
	}

	genDir := filepath.Join(tempDir, "nexus", "rpc", "api")
	return copy.Copy(genDir, genRoot())
}

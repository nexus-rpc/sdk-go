package nexus

import (
	"context"
)

type CompletionClient struct {
	options CompletionClientOptions
}

type CompletionClientOptions struct {
	Transport Transport
}

func NewCompletionClient(options CompletionClientOptions) (*CompletionClient, error) {
	return &CompletionClient{options: options}, nil
}

func (c *CompletionClient) CompleteOperation(
	ctx context.Context,
	result any,
	options CompleteOperationOptions,
) error {
	topts := TransportCompleteOperationOptions{
		ClientOptions: options,
		Success:       result,
	}
	_, err := c.options.Transport.CompleteOperation(ctx, topts)
	return err
}

func (c *CompletionClient) FailOperation(
	ctx context.Context,
	opError *OperationError,
	options CompleteOperationOptions,
) error {
	topts := TransportCompleteOperationOptions{
		ClientOptions: options,
		Error:         opError,
	}
	_, err := c.options.Transport.CompleteOperation(ctx, topts)
	return err
}

package nexus

import (
	"context"
	"errors"
)

var errResultAndErrorSet = errors.New("completion cannot contain both Result and Error")

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
	url string,
	options CompleteOperationOptions,
) error {
	if options.Result != nil && options.Error != nil {
		return errResultAndErrorSet
	}

	topts := TransportCompleteOperationOptions{
		ClientOptions: options,
		URL:           url,
	}
	if options.Error != nil {
		topts.State = options.Error.State
	} else {
		topts.State = OperationStateSucceeded
	}

	_, err := c.options.Transport.CompleteOperation(ctx, topts)
	return err
}

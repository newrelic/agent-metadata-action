package client

import (
	"context"
	"fmt"
)

// SigningClient handles OCI artifact signing operations
type SigningClient struct {
	baseURL string
	// Add additional fields as needed
}

// SigningRequest contains information needed to sign an OCI artifact
type SigningRequest struct {
	Digest     string `json:"digest"`
	Repository string `json:"repository"`
	Tag        string `json:"tag"`
}

// SigningResponse contains the signature information
type SigningResponse struct {
	Signature   string `json:"signature"`
	Certificate string `json:"certificate"`
	SignedAt    string `json:"signed_at"`
}

// SignArtifact requests signing for an OCI artifact
func (c *SigningClient) SignArtifact(ctx context.Context, req *SigningRequest) (*SigningResponse, error) {
	// TODO: Implement artifact signing
	// POST to signing service endpoint
	return nil, fmt.Errorf("not implemented")
}

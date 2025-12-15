package client

import (
	"context"
	"fmt"
	"io"
)

// OCIClient handles OCI registry operations
type OCIClient struct {
	registryURL string
	// Add additional fields as needed
}

// UploadRequest contains information for uploading to OCI registry
type UploadRequest struct {
	Repository string
	Tag        string
	Reader     io.Reader
	Size       int64
}

// UploadResponse contains the result of an OCI upload
type UploadResponse struct {
	Digest   string `json:"digest"`
	Location string `json:"location"`
}

// UploadFile uploads a file to the Docker OCI registry
func (c *OCIClient) UploadFile(ctx context.Context, req *UploadRequest) (*UploadResponse, error) {
	// TODO: Implement OCI file upload
	// POST/PUT to OCI registry endpoint
	return nil, fmt.Errorf("not implemented")
}

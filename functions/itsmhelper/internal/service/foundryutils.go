package service

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"runtime/debug"

	fdk "github.com/CrowdStrike/foundry-fn-go"
	"github.com/crowdstrike/gofalcon/falcon"
	"github.com/crowdstrike/gofalcon/falcon/client"
)

// WithPanicRecoveryWorkflow wraps a handler function with panic recovery logic
func WithPanicRecoveryWorkflow[T any](logger *slog.Logger, handlerFn func(context.Context, fdk.RequestOf[T], fdk.WorkflowCtx) fdk.Response) func(context.Context, fdk.RequestOf[T], fdk.WorkflowCtx) fdk.Response {
	return func(ctx context.Context, r fdk.RequestOf[T], wrkCtx fdk.WorkflowCtx) (response fdk.Response) {
		defer func() {
			if rec := recover(); rec != nil {
				stacktrace := string(debug.Stack())

				logger.Error("Handler panic recovered",
					"error", rec,
					"trace_id", r.TraceID,
					"url", r.URL,
					"stacktrace", stacktrace)

				errMsg := fmt.Sprintf("Internal fn error: %v (trace_id: '%s')", rec, r.TraceID)
				response = fdk.ErrResp(fdk.APIError{Code: http.StatusInternalServerError, Message: errMsg})
			}
		}()

		return handlerFn(ctx, r, wrkCtx)
	}
}

// NewFalconClient creates a new Falcon client.
func NewFalconClient(token string, logger *slog.Logger) (*client.CrowdStrikeAPISpecification, string, error) {
	ctx := context.Background()
	opts := fdk.FalconClientOpts()
	cloud := opts.Cloud

	if os.Getenv("FALCON_CLOUD") != "" {
		cloud = os.Getenv("FALCON_CLOUD")
	}

	apiConfig := &falcon.ApiConfig{
		AccessToken: token,
		Cloud:       falcon.Cloud(cloud),
		Context:     ctx,
	}

	if apiConfig.AccessToken == "" {
		apiConfig.ClientId = os.Getenv("FALCON_CLIENT_ID")
		apiConfig.ClientSecret = os.Getenv("FALCON_CLIENT_SECRET")
	}

	// When cloud is set to autodiscover, the client will attempt to determine the cloud based on the API response and update the config.
	// When the NewClient function returns, the cloud will be set to the actual cloud used.
	cloud = apiConfig.Cloud.String()

	logger.Info("Creating Falcon client", "cloud", cloud)

	client, err := falcon.NewClient(apiConfig)

	return client, cloud, err
}

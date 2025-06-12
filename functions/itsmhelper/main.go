package main

import (
	"context"
	"log/slog"

	"itsmhelper/internal/handler"
	"itsmhelper/internal/service"

	fdk "github.com/CrowdStrike/foundry-fn-go"
)

func main() {
	fdk.Run(context.Background(), newHandler)
}

type config struct {
	IsProd bool `json:"is_production"`
}

func (c config) OK() error {
	return nil
}

func newHandler(ctx context.Context, logger *slog.Logger, cfg config) fdk.Handler {
	m := fdk.NewMux()
	h := handler.NewHandler(logger, service.NewFalconClient)

	m.Post("/check_if_ext_entity_exists", fdk.HandleFnOf(func(ctx context.Context, r fdk.RequestOf[handler.CheckIfExtExistsReq]) fdk.Response {
		return h.HandleCheckIfExtEntityExists(ctx, r)
	}))

	m.Post("/create_entity_mapping", fdk.HandleFnOf(func(ctx context.Context, r fdk.RequestOf[handler.CreateEntityMappingReq]) fdk.Response {
		return h.HandleCreateEntityMapping(ctx, r)
	}))

	m.Post("/create_incident", fdk.HandleWorkflowOf(service.WithPanicRecoveryWorkflow(logger,
		func(ctx context.Context, r fdk.RequestOf[handler.CreateIncidentRequest], wrkCtx fdk.WorkflowCtx) fdk.Response {
			return h.HandleCreateIncident(ctx, r, wrkCtx)
		})))

	m.Post("/create_sir_incident", fdk.HandleWorkflowOf(service.WithPanicRecoveryWorkflow(logger,
		func(ctx context.Context, r fdk.RequestOf[handler.CreateIncidentRequest], wrkCtx fdk.WorkflowCtx) fdk.Response {
			return h.HandleCreateSIRIncident(ctx, r, wrkCtx)
		})))

	m.Post("/throttle", fdk.HandleFnOf(func(ctx context.Context, r fdk.RequestOf[handler.ThrottleFunctionRequest]) fdk.Response {
		return h.HandleThrottle(ctx, r)
	}))

	return m
}

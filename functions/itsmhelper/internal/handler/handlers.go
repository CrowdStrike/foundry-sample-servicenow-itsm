package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"itsmhelper/internal/storage"

	fdk "github.com/CrowdStrike/foundry-fn-go"
	"github.com/crowdstrike/gofalcon/falcon/client"
	"github.com/crowdstrike/gofalcon/falcon/client/api_integrations"
	"github.com/crowdstrike/gofalcon/falcon/models"
)

const (
	ExternalSystemIDServiceNowIncident    = "servicenow_incident"
	ExternalSystemIDServiceNowSIRIncident = "servicenow_sir_incident"
)

var (
	// Defined in 'api-integrations/servicenow.json'
	pluginDefIDServiceNow = "servicenow-foundry"

	pluginOpIDServiceNowCreateIncident    = "create_incident"
	pluginOpIDServiceNowCreateSIRIncident = "create_sn_si_incident"
)

type CheckIfExtExistsReq struct {
	InternalEntityID string `json:"internal_entity_id"`
	ExternalSystemID string `json:"external_system_id"`
}

type CreateEntityMappingReq struct {
	InternalEntityID string `json:"internal_entity_id"`
	ExternalEntityID string `json:"external_entity_id"`
	ExternalSystemID string `json:"external_system_id"`
}

// CreateIncidentRequest represents the request body for creating an incident
type CreateIncidentRequest struct {
	ConfigID string `json:"config_id"`
	EntityID string `json:"entity_id"`

	AssignmentGroup  string `json:"assignment_group"`
	Category         string `json:"category"`
	Description      string `json:"description"`
	Impact           string `json:"impact"`
	Severity         string `json:"severity"`
	ShortDescription string `json:"short_description"`
	State            string `json:"state"`
	Urgency          string `json:"urgency"`
	WorkNotes        string `json:"work_notes"`
	CustomFields     string `json:"custom_fields"`
}

// CreateIncidentResponse represents the response body for creating an incident
type CreateIncidentResponse struct {
	Exists     bool   `json:"exists"`
	TicketID   string `json:"ticket_id"`
	TicketType string `json:"ticket_type"`
}

// ThrottleFunctionRequest represents the schema for deduplication requests
type ThrottleFunctionRequest struct {
	InternalEntityID string `json:"internal_entity_id"`
	DedupObjType     string `json:"dedup_obj_type"`
	DedupObjID       string `json:"dedup_obj_id"`
	TimeBucket       string `json:"time_bucket"`
}

// FalconClientBuilder is a function type for creating Falcon clients
type FalconClientBuilder func(token string, logger *slog.Logger) (*client.CrowdStrikeAPISpecification, string, error)

// Handler contains all the handler functions and dependencies
type Handler struct {
	logger           *slog.Logger
	falconClientFunc FalconClientBuilder
}

// NewHandler creates a new Handler with the given logger
func NewHandler(logger *slog.Logger, falconClientBuilder FalconClientBuilder) *Handler {
	return &Handler{
		logger:           logger,
		falconClientFunc: falconClientBuilder,
	}
}

// HandleCheckIfExtEntityExists handles the /check_if_ext_entity_exists endpoint
func (h *Handler) HandleCheckIfExtEntityExists(ctx context.Context, r fdk.RequestOf[CheckIfExtExistsReq]) fdk.Response {
	accessToken := r.AccessToken

	falconClient, cloud, err := h.falconClientFunc(accessToken, h.logger)
	if err != nil {
		errMsg := fmt.Sprintf("error creating Falcon client: %v", err)
		return fdk.ErrResp(fdk.APIError{Code: http.StatusInternalServerError, Message: errMsg})
	}
	_ = cloud

	internalEntityID := r.Body.InternalEntityID
	externalSystemID := r.Body.ExternalSystemID

	exists, extRecord, err := storage.CheckExternalEntityExists(ctx, falconClient.CustomStorage, h.logger, internalEntityID, externalSystemID)
	if err != nil {
		errMsg := fmt.Sprintf("failed to check if ticket exists: %v", err)
		return fdk.ErrResp(fdk.APIError{Code: http.StatusInternalServerError, Message: errMsg})
	}

	if !exists {
		return fdk.Response{
			Code: http.StatusOK,
			Body: fdk.JSON(map[string]any{
				"exists": false,
			}),
		}
	}

	return fdk.Response{
		Code: http.StatusOK,
		Body: fdk.JSON(map[string]any{
			"exists":        true,
			"ext_id":        extRecord.ExternalEntityID,
			"ext_system_id": extRecord.ExternalSystemID,
		}),
	}
}

// HandleCreateEntityMapping handles the /create_entity_mapping endpoint
func (h *Handler) HandleCreateEntityMapping(ctx context.Context, r fdk.RequestOf[CreateEntityMappingReq]) fdk.Response {
	accessToken := r.AccessToken

	falconClient, cloud, err := h.falconClientFunc(accessToken, h.logger)
	if err != nil {
		errMsg := fmt.Sprintf("error creating Falcon client: %v", err)
		return fdk.ErrResp(fdk.APIError{Code: http.StatusInternalServerError, Message: errMsg})
	}
	_ = cloud

	entityRecord := storage.ExternalEntityRecord{
		InternalEntityID: r.Body.InternalEntityID,
		ExternalEntityID: r.Body.ExternalEntityID,
		ExternalSystemID: r.Body.ExternalSystemID,
	}

	err = storage.CreateOrUpdateExternalEntityMapping(ctx, falconClient.CustomStorage, h.logger, entityRecord)
	if err != nil {
		return fdk.ErrResp(fdk.APIError{Code: http.StatusInternalServerError, Message: err.Error()})
	}

	return fdk.Response{
		Code: http.StatusCreated,
		Body: fdk.JSON(entityRecord),
	}
}

// buildRequestPayload creates the request payload from the incident request
func buildRequestPayload(body CreateIncidentRequest) map[string]interface{} {
	requestPayload := map[string]interface{}{
		"short_description": body.ShortDescription,
	}

	// Add optional fields if they are provided
	if body.AssignmentGroup != "" {
		requestPayload["assignment_group"] = body.AssignmentGroup
	}
	if body.Category != "" {
		requestPayload["category"] = body.Category
	}
	if body.Description != "" {
		requestPayload["description"] = body.Description
	}
	if body.Impact != "" {
		requestPayload["impact"] = body.Impact
	}
	if body.Severity != "" {
		requestPayload["severity"] = body.Severity
	}
	if body.State != "" {
		requestPayload["state"] = body.State
	}
	if body.Urgency != "" {
		requestPayload["urgency"] = body.Urgency
	}
	if body.WorkNotes != "" {
		requestPayload["work_notes"] = body.WorkNotes
	}

	if body.CustomFields != "" {
		var customFields map[string]interface{}
		if err := json.Unmarshal([]byte(body.CustomFields), &customFields); err == nil {
			for key, value := range customFields {
				requestPayload[key] = value
			}
		}
	}

	return requestPayload
}

// createIncident handles the common logic for creating both regular and SIR incidents
func (h *Handler) createIncident(
	ctx context.Context,
	r fdk.RequestOf[CreateIncidentRequest],
	wrkCtx fdk.WorkflowCtx,
	operationID string,
	ticketType string,
	externalSystemID string,
) fdk.Response {
	h.logger.Info("Creating incident", "type", ticketType, "trace_id", r.TraceID, "wrk_ctx", wrkCtx)
	accessToken := r.AccessToken

	falconClient, cloud, err := h.falconClientFunc(accessToken, h.logger)
	if err != nil {
		errMsg := fmt.Sprintf("error creating Falcon client: %v", err)
		return fdk.ErrResp(fdk.APIError{Code: http.StatusInternalServerError, Message: errMsg})
	}
	_ = cloud

	// First check if a ticket for this entity already exists with the specific external system ID
	exists, extRecord, err := storage.CheckExternalEntityExists(ctx, falconClient.CustomStorage, h.logger, r.Body.EntityID, externalSystemID)
	if err != nil {
		errMsg := fmt.Sprintf("failed to check if ticket exists: %v", err)
		return fdk.ErrResp(fdk.APIError{Code: http.StatusInternalServerError, Message: errMsg})
	}

	// If the entity has an existing ticket with the specified external system ID, return it
	if exists {
		h.logger.Info("ticket already exists for entity", "entity_id", r.Body.EntityID, "ticket_id", extRecord.ExternalEntityID)
		return fdk.Response{
			Code: http.StatusOK,
			Body: fdk.JSON(CreateIncidentResponse{
				Exists:     true,
				TicketID:   extRecord.ExternalEntityID,
				TicketType: ticketType,
			}),
		}
	}

	// If no existing ticket, proceed with creating a new one
	// Prepare the request payload using the input parameters
	requestPayload := buildRequestPayload(r.Body)

	configID := r.Body.ConfigID
	execCmdParams := &api_integrations.ExecuteCommandParams{
		Body: &models.DomainExecuteCommandRequestV1{Resources: []*models.DomainExecuteCommandV1{
			{
				DefinitionID: &pluginDefIDServiceNow,
				OperationID:  &operationID,
				ConfigID:     &configID,
				Request: &models.DomainRequest{
					JSON: requestPayload,
				},
			},
		}},
		Context: ctx,
	}

	execResp, err := falconClient.APIIntegrations.ExecuteCommand(execCmdParams)
	if err != nil {
		errMsg := fmt.Sprintf("failed to execute command: %v", err)
		return fdk.ErrResp(fdk.APIError{Code: http.StatusInternalServerError, Message: errMsg})
	}

	if execResp == nil {
		return fdk.ErrResp(fdk.APIError{Code: http.StatusInternalServerError, Message: "failed to execute command - nil response"})
	}

	h.logger.Info("plugin execution completed", "status_code", execResp.Code())
	if execResp.Payload == nil {
		return fdk.ErrResp(fdk.APIError{Code: http.StatusInternalServerError, Message: "failed to execute command - empty response"})
	}

	resources := execResp.Payload.Resources
	if len(resources) == 0 {
		return fdk.ErrResp(fdk.APIError{Code: http.StatusInternalServerError, Message: "failed to execute command - empty resources in response payload"})
	}

	resource := resources[0]
	resourceRespBody := resource.ResponseBody

	snowSysClassName := ""
	snowSysID := ""
	errorText := ""

	if result, ok := resourceRespBody.(map[string]interface{})["result"]; ok {
		if resultMap, ok := result.(map[string]interface{}); ok {
			// Try to get sys_class_name
			if sysClassName, ok := resultMap["sys_class_name"].(string); ok {
				snowSysClassName = sysClassName
			}

			// Try to get sys_id
			if sysID, ok := resultMap["sys_id"].(string); ok {
				snowSysID = sysID
			}
		}
	}

	// Check if there's an error field in the response
	if errorField, ok := resourceRespBody.(map[string]interface{})["error"]; ok {
		// Convert the error field to a string
		if errorStr, ok := errorField.(string); ok {
			errorText = errorStr
		} else {
			// If it's not a string, try to convert it to JSON
			if errorBytes, err := json.Marshal(errorField); err == nil {
				errorText = string(errorBytes)
			} else {
				errorText = fmt.Sprintf("Error field present but could not be parsed: %v", errorField)
			}
		}

		errMsg := fmt.Sprintf("failed to execute command: ServiceNow Error: %s", errorText)
		return fdk.ErrResp(fdk.APIError{Code: http.StatusInternalServerError, Message: errMsg})
	}

	h.logger.Info("received response from ITSM", "ticket_id", snowSysID, "ticket_type", snowSysClassName)

	// If we successfully created a ticket, store the mapping
	if snowSysID != "" {
		// Create the entity mapping record with the specific external system ID
		entityRecord := storage.ExternalEntityRecord{
			InternalEntityID: r.Body.EntityID,
			ExternalEntityID: snowSysID,
			ExternalSystemID: externalSystemID,
		}

		// Store the mapping using the reusable function
		err := storage.CreateOrUpdateExternalEntityMapping(ctx, falconClient.CustomStorage, h.logger, entityRecord)
		if err != nil {
			h.logger.Error("failed to store entity mapping", "error", err)
			return fdk.ErrResp(fdk.APIError{Code: http.StatusInternalServerError, Message: err.Error()})
		}
	}

	response := CreateIncidentResponse{
		TicketID:   snowSysID,
		TicketType: snowSysClassName,
		Exists:     false,
	}

	return fdk.Response{
		Code: http.StatusCreated,
		Body: fdk.JSON(response),
	}
}

// HandleCreateIncident handles the /create_incident endpoint
func (h *Handler) HandleCreateIncident(ctx context.Context, r fdk.RequestOf[CreateIncidentRequest], wrkCtx fdk.WorkflowCtx) fdk.Response {
	return h.createIncident(ctx, r, wrkCtx, pluginOpIDServiceNowCreateIncident, "incident", ExternalSystemIDServiceNowIncident)
}

// HandleCreateSIRIncident handles the /create_sir_incident endpoint
func (h *Handler) HandleCreateSIRIncident(ctx context.Context, r fdk.RequestOf[CreateIncidentRequest], wrkCtx fdk.WorkflowCtx) fdk.Response {
	return h.createIncident(ctx, r, wrkCtx, pluginOpIDServiceNowCreateSIRIncident, "sn_si_incident", ExternalSystemIDServiceNowSIRIncident)
}

// handleThrottle handles the /throttle endpoint
func (h *Handler) HandleThrottle(ctx context.Context, r fdk.RequestOf[ThrottleFunctionRequest]) fdk.Response {
	accessToken := r.AccessToken

	falconClient, _, err := h.falconClientFunc(accessToken, h.logger)
	if err != nil {
		errMsg := fmt.Sprintf("error creating Falcon client: %v", err)
		return fdk.ErrResp(fdk.APIError{Code: http.StatusInternalServerError, Message: errMsg})
	}

	internalEntityID := r.Body.InternalEntityID
	dedupObjType := r.Body.DedupObjType
	dedupObjId := r.Body.DedupObjID
	timeBucket := r.Body.TimeBucket

	// Check throttling store for deduplication
	isDuplicate, err := storage.CheckThrottlingStore(ctx, falconClient.CustomStorage, h.logger, internalEntityID, dedupObjType, dedupObjId, timeBucket)
	if err != nil {
		return fdk.ErrResp(fdk.APIError{Code: http.StatusInternalServerError, Message: err.Error()})
	}

	// If it's a duplicate, don't allow the action
	return fdk.Response{
		Code: http.StatusOK,
		Body: fdk.JSON(map[string]any{
			"allowed": !isDuplicate,
		}),
	}
}

"""
ServiceNow ITSM Integration Function - Python Implementation

This function integrates ServiceNow ITSM/SIR with Falcon security alerts.
It provides endpoints for creating incidents and tracking entity mappings.

Key Features:
- Check if external entity mappings exist
- Create entity mappings between internal and external systems
- Create ServiceNow incidents and SIR incidents
- Throttling/deduplication for preventing duplicate actions
- Support for custom fields and flexible configuration

Architecture:
- Uses falconpy SDK for Falcon API interactions
- Stores entity mappings in Falcon Custom Storage collections
- Executes ServiceNow API calls via Falcon API Integrations
"""
# pylint: disable=broad-exception-caught

import json
import hashlib
import re
from datetime import datetime, timezone
from logging import Logger
from typing import Dict, Any, Optional, Tuple
from http import HTTPStatus

from crowdstrike.foundry.function import Function, Request, Response, APIError
from falconpy import APIIntegrations, CustomStorage

# Initialize the function instance
FUNC = Function.instance()

# Constants
EXTERNAL_SYSTEM_ID_SERVICENOW_INCIDENT = "servicenow_incident"
EXTERNAL_SYSTEM_ID_SERVICENOW_SIR_INCIDENT = "servicenow_sir_incident"

# Defined in 'api-integrations/servicenow.json'
PLUGIN_DEF_ID_SERVICENOW = "servicenow-foundry"
PLUGIN_OP_ID_SERVICENOW_CREATE_INCIDENT = "create_incident"
PLUGIN_OP_ID_SERVICENOW_CREATE_SIR_INCIDENT = "create_sn_si_incident"

# Collection names
COLLECTION_NAME_TRACKED_ENTITIES = "tracked_entities"
COLLECTION_NAME_DEDUP_STORE = "dedup_store"

# Time bucket constants
TIME_BUCKET_FOREVER = "forever"
TIME_BUCKET_FIVE_MIN = "5 minutes"
TIME_BUCKET_THIRTY_MIN = "30 minutes"


class ITSMError(Exception):
    """Base exception for ITSM helper errors."""


class StorageError(ITSMError):
    """Exception for storage-related errors."""


class FalconClientError(ITSMError):
    """Exception for Falcon client errors."""


def create_falcon_clients(logger: Logger) -> Tuple[APIIntegrations, CustomStorage]:
    """
    Create Falcon API clients.

    In the Foundry runtime, authentication is handled automatically
    via environment variables (FALCON_CLIENT_ID, FALCON_CLIENT_SECRET).

    Args:
        logger: Logger instance for logging

    Returns:
        Tuple of (APIIntegrations client, CustomStorage client)

    Raises:
        FalconClientError: If client creation fails
    """
    try:
        api_integrations = APIIntegrations()
        collections = CustomStorage()

        return api_integrations, collections

    except Exception as e:
        logger.error(f"Error creating Falcon clients: {str(e)}", exc_info=True)
        raise FalconClientError(f"Failed to create Falcon clients: {str(e)}") from e


def sanitize_object_key(input_str: str) -> str:
    """
    Sanitize object key to meet custom storage requirements.

    Args:
        input_str: Input string to sanitize

    Returns:
        Sanitized object key

    Raises:
        ValueError: If sanitized key exceeds maximum length
    """
    # Replace disallowed characters with underscore
    sanitized = re.sub(r'[^a-zA-Z0-9._-]', '_', input_str)

    # Check length constraints
    if len(sanitized) > 1000:
        raise ValueError(f"Object key exceeds maximum length of 1000 characters: {len(sanitized)}")

    return sanitized


def create_tracked_entity_key(external_system_id: str, internal_entity_id: str) -> str:
    """
    Generate a unique key for tracked entities.

    Args:
        external_system_id: External system identifier
        internal_entity_id: Internal entity identifier

    Returns:
        Sanitized unique key
    """
    combined = f"{external_system_id}.{internal_entity_id}"
    return sanitize_object_key(combined)


def calculate_time_bucket(time_bucket: str) -> str:
    """
    Calculate the current time bucket based on the time bucket type.

    Args:
        time_bucket: Time bucket type (forever, 5 minutes, 30 minutes)

    Returns:
        Time bucket string
    """
    if time_bucket == TIME_BUCKET_FOREVER:
        return "forever_bucket"

    now = datetime.now(timezone.utc)

    if time_bucket == TIME_BUCKET_FIVE_MIN:
        # Round down to the nearest 5-minute interval
        minutes = (now.minute // 5) * 5
        # Format as "YYYY-MM-DD_HH:MM"
        date_part = now.strftime("%Y-%m-%d")
        minute_part = f"{now.hour:02d}:{minutes:02d}"
        return f"{date_part}_{minute_part}"
    if time_bucket == TIME_BUCKET_THIRTY_MIN:
        # Round down to the nearest 30-minute interval
        minutes = (now.minute // 30) * 30
        # Format as "YYYY-MM-DD_HH:MM"
        date_part = now.strftime("%Y-%m-%d")
        minute_part = f"{now.hour:02d}:{minutes:02d}"
        return f"{date_part}_{minute_part}"

    raise ValueError(f"Unsupported time bucket: {time_bucket}")


def check_external_entity_exists(
    collections: CustomStorage,
    _logger: Logger,
    internal_entity_id: str,
    external_system_id: str
) -> Tuple[bool, Optional[Dict[str, str]]]:
    """
    Check if an external entity mapping exists.

    Args:
        collections: CustomStorage client
        logger: Logger instance
        internal_entity_id: Internal entity ID
        external_system_id: External system ID

    Returns:
        Tuple of (exists, external_entity_record)
    """
    key = create_tracked_entity_key(external_system_id, internal_entity_id)

    response = collections.GetObject(
        collection_name=COLLECTION_NAME_TRACKED_ENTITIES,
        object_key=key
    )

    # GetObject returns bytes on success, dict on error.
    try:
        ext_record = json.loads(response.decode('utf-8'))
    except (AttributeError, UnicodeDecodeError, json.JSONDecodeError):
        return False, None

    # Verify external_system_id matches if provided
    if external_system_id and ext_record.get("external_system_id") != external_system_id:
        return False, None

    return True, ext_record


def create_or_update_external_entity_mapping(
    collections: CustomStorage,
    logger: Logger,
    internal_entity_id: str,
    external_entity_id: str,
    external_system_id: str
) -> None:
    """
    Create or update an external entity mapping.

    Args:
        collections: CustomStorage client
        logger: Logger instance
        internal_entity_id: Internal entity ID
        external_entity_id: External entity ID
        external_system_id: External system ID
    """
    try:
        key = create_tracked_entity_key(external_system_id, internal_entity_id)

        record = {
            "internal_entity_id": internal_entity_id,
            "external_entity_id": external_entity_id,
            "external_system_id": external_system_id
        }

        response = collections.PutObject(
            collection_name=COLLECTION_NAME_TRACKED_ENTITIES,
            object_key=key,
            body=record
        )

        if response["status_code"] not in [200, 201]:
            logger.error(f"Failed to store entity mapping: {response}")
            raise StorageError(f"Failed to store entity mapping: {response}")

        logger.info(
            f"Successfully stored entity mapping - "
            f"internal_id: {internal_entity_id}, "
            f"external_id: {external_entity_id}, "
            f"system_id: {external_system_id}"
        )

    except StorageError:
        raise
    except Exception as e:
        logger.error(f"Error storing entity mapping: {str(e)}", exc_info=True)
        raise StorageError(f"Failed to store entity mapping: {str(e)}") from e


def check_throttling_store(  # pylint: disable=too-many-arguments,too-many-positional-arguments
    collections: CustomStorage,
    logger: Logger,
    internal_entity_id: str,
    dedup_obj_type: str,
    dedup_obj_id: str,
    time_bucket: str
) -> bool:
    """
    Check if a combination of IDs already exists in throttling store.

    Args:
        collections: CustomStorage client
        logger: Logger instance
        internal_entity_id: Internal entity ID
        dedup_obj_type: Deduplication object type
        dedup_obj_id: Deduplication object ID
        time_bucket: Time bucket (forever, 5m, 30m)

    Returns:
        True if duplicate (exists), False if new
    """
    # Validate time bucket
    if time_bucket not in [TIME_BUCKET_FOREVER, TIME_BUCKET_FIVE_MIN, TIME_BUCKET_THIRTY_MIN]:
        raise ValueError(
            f"Unsupported time bucket value: {time_bucket} "
            f"(must be one of: {TIME_BUCKET_FOREVER}, {TIME_BUCKET_FIVE_MIN}, {TIME_BUCKET_THIRTY_MIN})"
        )

    # Calculate current bucket
    current_bucket = calculate_time_bucket(time_bucket)

    # Create dedup key
    combined = f"{internal_entity_id}:{dedup_obj_type}:{dedup_obj_id}:{current_bucket}"
    dedup_key = hashlib.md5(combined.encode()).hexdigest()

    # Try to get the object — success means duplicate exists
    response = collections.GetObject(
        collection_name=COLLECTION_NAME_DEDUP_STORE,
        object_key=dedup_key
    )

    # GetObject returns bytes on success, dict on error.
    try:
        json.loads(response.decode('utf-8'))
        return True  # Record exists, it's a duplicate
    except (AttributeError, UnicodeDecodeError, json.JSONDecodeError):
        pass

    # Object doesn't exist, create it.
    logger.info(f"No dedup record found for key: {dedup_key}, creating new entry")

    # Store new dedup record
    new_record = {"time_bucket": time_bucket}
    response = collections.PutObject(
        collection_name=COLLECTION_NAME_DEDUP_STORE,
        object_key=dedup_key,
        body=new_record
    )

    if response["status_code"] not in [200, 201]:
        logger.error(f"Failed to store dedup record: {response}")
        raise StorageError(f"Failed to store dedup record: {response}")

    return False  # New entry, not a duplicate


def build_request_payload(body: Dict[str, Any]) -> Dict[str, Any]:
    """
    Build ServiceNow API request payload from incident request.

    Args:
        body: Request body

    Returns:
        Request payload dictionary
    """
    request_payload = {
        "short_description": body.get("short_description", "")
    }

    # Add optional fields if provided
    optional_fields = [
        "assignment_group", "category", "description", "impact",
        "severity", "state", "urgency", "work_notes"
    ]

    for field in optional_fields:
        if body.get(field):
            request_payload[field] = body[field]

    # Handle custom fields
    custom_fields_str = body.get("custom_fields")
    if custom_fields_str:
        try:
            custom_fields = json.loads(custom_fields_str) if isinstance(custom_fields_str, str) else custom_fields_str
            request_payload.update(custom_fields)
        except json.JSONDecodeError:
            # Log but don't fail - custom fields are optional
            pass

    return request_payload


@FUNC.handler(path="/check_if_ext_entity_exists", method="POST")
def check_if_ext_entity_exists_handler(req: Request, _config: Optional[Dict[str, object]], logger: Logger) -> Response:
    """
    Check if an external entity mapping exists.

    Request body:
        - internal_entity_id: Internal entity ID
        - external_system_id: External system ID
    """
    try:
        logger.info(f"check_if_ext_entity_exists called - trace_id: {req.trace_id}")

        body = req.body
        internal_entity_id = body.get("internal_entity_id")
        external_system_id = body.get("external_system_id")

        if not internal_entity_id or not external_system_id:
            return Response(
                code=HTTPStatus.BAD_REQUEST,
                errors=[APIError(
                    code=HTTPStatus.BAD_REQUEST,
                    message="Missing required fields: internal_entity_id, external_system_id"
                )]
            )

        _, api_client = create_falcon_clients(logger)

        exists, ext_record = check_external_entity_exists(
            api_client, logger, internal_entity_id, external_system_id
        )

        if not exists:
            return Response(
                code=HTTPStatus.OK,
                body={"exists": False}
            )

        return Response(
            code=HTTPStatus.OK,
            body={
                "exists": True,
                "ext_id": ext_record.get("external_entity_id"),
                "ext_system_id": ext_record.get("external_system_id")
            }
        )

    except Exception as e:
        logger.error(f"Error in check_if_ext_entity_exists: {str(e)}", exc_info=True)
        return Response(
            code=HTTPStatus.INTERNAL_SERVER_ERROR,
            errors=[APIError(
                code=HTTPStatus.INTERNAL_SERVER_ERROR,
                message=f"Internal error: {str(e)}"
            )]
        )


@FUNC.handler(path="/create_entity_mapping", method="POST")
def create_entity_mapping_handler(req: Request, _config: Optional[Dict[str, object]], logger: Logger) -> Response:
    """
    Create an entity mapping between internal and external systems.

    Request body:
        - internal_entity_id: Internal entity ID
        - external_entity_id: External entity ID
        - external_system_id: External system ID
    """
    try:
        logger.info(f"create_entity_mapping called - trace_id: {req.trace_id}")

        body = req.body
        internal_entity_id = body.get("internal_entity_id")
        external_entity_id = body.get("external_entity_id")
        external_system_id = body.get("external_system_id")

        if not all([internal_entity_id, external_entity_id, external_system_id]):
            return Response(
                code=HTTPStatus.BAD_REQUEST,
                errors=[APIError(
                    code=HTTPStatus.BAD_REQUEST,
                    message="Missing required fields: internal_entity_id, external_entity_id, external_system_id"
                )]
            )

        _, api_client = create_falcon_clients(logger)

        create_or_update_external_entity_mapping(
            api_client, logger,
            internal_entity_id, external_entity_id, external_system_id
        )

        return Response(
            code=HTTPStatus.CREATED,
            body={
                "internal_entity_id": internal_entity_id,
                "external_entity_id": external_entity_id,
                "external_system_id": external_system_id
            }
        )

    except Exception as e:
        logger.error(f"Error in create_entity_mapping: {str(e)}", exc_info=True)
        return Response(
            code=HTTPStatus.INTERNAL_SERVER_ERROR,
            errors=[APIError(
                code=HTTPStatus.INTERNAL_SERVER_ERROR,
                message=f"Internal error: {str(e)}"
            )]
        )


def create_incident_impl(  # pylint: disable=too-many-locals,too-many-return-statements
    req: Request,
    logger: Logger,
    operation_id: str,
    ticket_type: str,
    external_system_id: str
) -> Response:
    """
    Common implementation for creating incidents.

    Args:
        req: Request object
        logger: Logger instance
        operation_id: ServiceNow operation ID
        ticket_type: Ticket type name
        external_system_id: External system identifier

    Returns:
        Response object
    """
    try:
        # Log workflow context if present
        logger.info(
            f"Creating {ticket_type} - trace_id: {req.trace_id}, workflow_ctx: {req.context}"
        )

        body = req.body
        config_id = body.get("config_id")
        entity_id = body.get("entity_id")

        if not config_id or not entity_id:
            return Response(
                code=HTTPStatus.BAD_REQUEST,
                errors=[APIError(
                    code=HTTPStatus.BAD_REQUEST,
                    message="Missing required fields: config_id, entity_id"
                )]
            )

        api_integrations, api_client = create_falcon_clients(logger)

        # Check if ticket already exists
        exists, ext_record = check_external_entity_exists(
            api_client, logger, entity_id, external_system_id
        )

        if exists:
            logger.info(f"Ticket already exists for entity {entity_id}: {ext_record.get('external_entity_id')}")
            return Response(
                code=HTTPStatus.OK,
                body={
                    "exists": True,
                    "ticket_id": ext_record.get("external_entity_id"),
                    "ticket_type": ticket_type
                }
            )

        # Build request payload
        request_payload = build_request_payload(body)

        # Execute API integration command
        command_body = {
            "resources": [{
                "definition_id": PLUGIN_DEF_ID_SERVICENOW,
                "operation_id": operation_id,
                "config_id": config_id,
                "request": {
                    "json": request_payload
                }
            }]
        }

        exec_response = api_integrations.execute_command(body=command_body)

        status_code = exec_response.get("status_code")
        logger.info(f"Plugin execution completed - status: {status_code}")

        if status_code not in [200, 201]:
            errors = exec_response.get("body", {}).get("errors", [])
            error_msg = errors[0].get("message", "Unknown error") if errors else f"Status {status_code}"
            logger.error(f"Failed to execute command: {error_msg}")
            return Response(
                code=HTTPStatus.INTERNAL_SERVER_ERROR,
                errors=[APIError(
                    code=HTTPStatus.INTERNAL_SERVER_ERROR,
                    message=f"Failed to execute command: {error_msg}"
                )]
            )

        # Parse response
        resources = exec_response.get("body", {}).get("resources", [])
        if not resources:
            return Response(
                code=HTTPStatus.INTERNAL_SERVER_ERROR,
                errors=[APIError(
                    code=HTTPStatus.INTERNAL_SERVER_ERROR,
                    message="Empty response from ServiceNow"
                )]
            )

        resource = resources[0]
        response_body = resource.get("response_body", {})

        # Check for errors
        if "error" in response_body:
            error_text = response_body["error"]
            if isinstance(error_text, dict):
                error_text = json.dumps(error_text)
            logger.error(f"ServiceNow error: {error_text}")
            return Response(
                code=HTTPStatus.INTERNAL_SERVER_ERROR,
                errors=[APIError(
                    code=HTTPStatus.INTERNAL_SERVER_ERROR,
                    message=f"ServiceNow Error: {error_text}"
                )]
            )

        # Extract ticket information
        result = response_body.get("result", {})
        snow_sys_class_name = result.get("sys_class_name", "")
        snow_sys_id = result.get("sys_id", "")

        logger.info(f"Received response from ITSM - ticket_id: {snow_sys_id}, ticket_type: {snow_sys_class_name}")

        # Store entity mapping
        if snow_sys_id:
            create_or_update_external_entity_mapping(
                api_client, logger,
                entity_id, snow_sys_id, external_system_id
            )

        return Response(
            code=HTTPStatus.CREATED,
            body={
                "exists": False,
                "ticket_id": snow_sys_id,
                "ticket_type": snow_sys_class_name
            }
        )

    except Exception as e:
        logger.error(f"Error creating {ticket_type}: {str(e)}", exc_info=True)
        return Response(
            code=HTTPStatus.INTERNAL_SERVER_ERROR,
            errors=[APIError(
                code=HTTPStatus.INTERNAL_SERVER_ERROR,
                message=f"Internal error: {str(e)}"
            )]
        )


@FUNC.handler(path="/create_incident", method="POST")
def create_incident_handler(req: Request, _config: Optional[Dict[str, object]], logger: Logger) -> Response:
    """
    Create a ServiceNow incident.

    Request body:
        - config_id: ServiceNow config ID
        - entity_id: Entity ID to track
        - short_description: Short description (required)
        - assignment_group, category, description, impact, severity, state, urgency, work_notes: Optional fields
        - custom_fields: JSON string of custom fields
    """
    return create_incident_impl(
        req, logger,
        PLUGIN_OP_ID_SERVICENOW_CREATE_INCIDENT,
        "incident",
        EXTERNAL_SYSTEM_ID_SERVICENOW_INCIDENT
    )


@FUNC.handler(path="/create_sir_incident", method="POST")
def create_sir_incident_handler(req: Request, _config: Optional[Dict[str, object]], logger: Logger) -> Response:
    """
    Create a ServiceNow SIR incident.

    Request body:
        - config_id: ServiceNow config ID
        - entity_id: Entity ID to track
        - short_description: Short description (required)
        - assignment_group, category, description, impact, severity, state, urgency, work_notes: Optional fields
        - custom_fields: JSON string of custom fields
    """
    return create_incident_impl(
        req, logger,
        PLUGIN_OP_ID_SERVICENOW_CREATE_SIR_INCIDENT,
        "sn_si_incident",
        EXTERNAL_SYSTEM_ID_SERVICENOW_SIR_INCIDENT
    )


@FUNC.handler(path="/throttle", method="POST")
def throttle_handler(req: Request, _config: Optional[Dict[str, object]], logger: Logger) -> Response:
    """
    Check if an action should be throttled based on deduplication store.

    Request body:
        - internal_entity_id: Internal entity ID
        - dedup_obj_type: Deduplication object type
        - dedup_obj_id: Deduplication object ID
        - time_bucket: Time bucket (forever, 5 minutes, 30 minutes)
    """
    try:
        logger.info(f"throttle called - trace_id: {req.trace_id}")

        body = req.body
        internal_entity_id = body.get("internal_entity_id")
        dedup_obj_type = body.get("dedup_obj_type")
        dedup_obj_id = body.get("dedup_obj_id")
        time_bucket = body.get("time_bucket")

        if not all([internal_entity_id, dedup_obj_type, dedup_obj_id, time_bucket]):
            return Response(
                code=HTTPStatus.BAD_REQUEST,
                errors=[APIError(
                    code=HTTPStatus.BAD_REQUEST,
                    message="Missing required fields: internal_entity_id, dedup_obj_type, dedup_obj_id, time_bucket"
                )]
            )

        _, api_client = create_falcon_clients(logger)

        is_duplicate = check_throttling_store(
            api_client, logger,
            internal_entity_id, dedup_obj_type, dedup_obj_id, time_bucket
        )

        return Response(
            code=HTTPStatus.OK,
            body={"allowed": not is_duplicate}
        )

    except Exception as e:
        logger.error(f"Error in throttle: {str(e)}", exc_info=True)
        return Response(
            code=HTTPStatus.INTERNAL_SERVER_ERROR,
            errors=[APIError(
                code=HTTPStatus.INTERNAL_SERVER_ERROR,
                message=f"Internal error: {str(e)}"
            )]
        )

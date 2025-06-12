# Components

## Actions

### 1. Check If External Entity Exists
**Name**: `ITSM Helper - Entities - Check if external entity exists`  
**Handler**: `HandleCheckIfExtEntityExists`  
**API Path**: `/check_if_ext_entity_exists`  
**Workflow Integration ID**: `61143666563e4735a79cb423c34bcdb4`

**Description**:  
This action checks if an external entity mapping exists for a given internal entity ID in a specified external system. It's primarily used to determine if a ServiceNow ticket already exists for a specific CrowdStrike entity.

**Schema Files**:
- Request Schema: [check_if_ext_entity_exists_req_schema.json](functions/itsmhelper/schemas/check_if_ext_entity_exists_req_schema.json)
- Response Schema: [check_if_ext_entity_exists_resp_schema.json](functions/itsmhelper/schemas/check_if_ext_entity_exists_resp_schema.json)

**Request Parameters**:
- `internal_entity_id` (string, required): The internal identifier for the entity in CrowdStrike
- `external_system_id` (string, required): The identifier for the external system (e.g., "servicenow_incident")

**Response**:
- `exists` (boolean): Indicates whether the mapping exists
- `ext_id` (string, optional): The external entity ID if the mapping exists
- `ext_system_id` (string, optional): The external system ID if the mapping exists

### 2. Create Entity Mapping
**Name**: `ITSM Helper - Entities - Establish mapping`  
**Handler**: `HandleCreateEntityMapping`  
**API Path**: `/create_entity_mapping`  
**Workflow Integration ID**: `018e9b3fc2324b78af5d1352d058d7cf`

**Description**:  
This action creates or updates a mapping between internal CrowdStrike entities and external entities (like ServiceNow tickets). It stores the relationship in the custom storage.

**Schema Files**:
- Request Schema: [create_entity_mapping_req_schema.json](functions/itsmhelper/schemas/create_entity_mapping_req_schema.json)
- Response Schema: [create_entity_mapping_resp_schema.json](functions/itsmhelper/schemas/create_entity_mapping_resp_schema.json)

**Request Parameters**:
- `internal_entity_id` (string, required): The internal identifier for the entity in CrowdStrike
- `external_entity_id` (string, required): The identifier for the entity in the external system
- `external_system_id` (string, required): The identifier for the external system

**Response**:
- Status information about the created mapping
- Error information if something went wrong

### 3. Create Incident
**Name**: `ITSM Helper - Create Incident`  
**Handler**: `HandleCreateIncident`  
**API Path**: `/create_incident`  
**Workflow Integration ID**: `0c7b883a19594810934f99b6def5246b`

**Description**:  
This action creates a standard incident in ServiceNow. It first checks if a ticket already exists for the entity, and if not, creates a new one using the ServiceNow API integration. It then stores the mapping between the CrowdStrike entity and the ServiceNow ticket.

**Schema Files**:
- Request Schema: [create_incident_req_schema.json](functions/itsmhelper/schemas/create_incident_req_schema.json)
- Response Schema: [create_incident_resp_schema.json](functions/itsmhelper/schemas/create_incident_resp_schema.json)

**Request Parameters**:
- `entity_id` (string, required): The internal entity ID in CrowdStrike
- `short_description` (string, required): Brief description of the incident
- `config_id` (string, required): Configuration ID for the ServiceNow integration
- `assignment_group` (string, optional): Group to assign the incident to
- `category` (string, optional): Incident category
- `description` (string, optional): Detailed description
- `impact` (string, optional): Impact level
- `severity` (string, optional): Severity level
- `state` (string, optional): Incident state
- `urgency` (string, optional): Urgency level
- `work_notes` (string, optional): Additional notes
- `custom_fields` (string, optional): JSON string containing custom ServiceNow fields as key-value pairs (e.g., `{"u_custom_field1": "value1", "u_affected_systems": 3}`)

**Response**:
- `exists` (boolean): Indicates if the ticket already existed
- `ticket_id` (string): The ServiceNow ticket ID
- `ticket_type` (string): The type of ticket created (typically "incident")

### 4. Create SIR Incident
**Name**: `ITSM Helper - Create SIR Incident`  
**Handler**: `HandleCreateSIRIncident`  
**API Path**: `/create_sir_incident`  
**Workflow Integration ID**: `e98c73fca1394b0ebd8981dcc4c65c74`

**Description**:  
This action creates a Security Incident Response (SIR) incident in ServiceNow. It functions similarly to the standard incident creation handler but creates a different type of ticket (sn_si_incident) and uses a different external system ID for tracking.

**Schema Files**:
- Request Schema: [create_sir_incident_req_schema.json](functions/itsmhelper/schemas/create_sir_incident_req_schema.json)
- Response Schema: [create_sir_incident_resp_schema.json](functions/itsmhelper/schemas/create_sir_incident_resp_schema.json)

**Request Parameters**:
- Same as Create Incident, but with different category and severity options specific to SIR incidents

**Response**:
- Same structure as Create Incident response

### 5. Throttle
**Name**: `ITSM Helper - Throttle`  
**Handler**: `HandleThrottle`  
**API Path**: `/throttle`  
**Workflow Integration ID**: `27aeedcb470e4cfda776d140dc5626e0`

**Description**:  
This action provides throttling functionality to control the flow of updates in workflows. It checks if an action should be allowed based on deduplication logic, helping to prevent duplicate tickets or actions within a specified time bucket.

**Schema Files**:
- Request Schema: [throttle_req_schema.json](functions/itsmhelper/schemas/throttle_req_schema.json)
- Response Schema: [throttle_resp_schema.json](functions/itsmhelper/schemas/throttle_resp_schema.json)

**Request Parameters**:
- `internal_entity_id` (string, required): Internal system identifier (e.g., CVE ID)
- `dedup_obj_type` (string, required): Type of object for deduplication (e.g., "Host", "User")
- `dedup_obj_id` (string, required): ID of the specific object for deduplication
- `time_bucket` (string, required): Time bucket for time-based deduping ("forever", "5 minutes", "30 minutes")

**Response**:
- `allowed` (boolean): Indicates whether further processing is allowed

## Workflow Integration

All actions are part of a single function called `itsm_helper` with ID `aff9a77fd6a845bf89d0a581410756dc`. This function is exposed to Workflow through the integrations listed above.

The ServiceNow ITSM Helper App uses the ServiceNow API integration (ID: `425a02a359bd49ed92be2075a98898bc`, defined in [api-integrations/servicenow.json](api-integrations/servicenow.json)) to communicate with ServiceNow. It supports two main operations:
- `create_incident`: Creates a standard incident in ServiceNow
- `create_sn_si_incident`: Creates a Security Incident Response (SIR) incident in ServiceNow

The app also uses two custom collections for storage:
1. `tracked_entities`: Stores mappings between CrowdStrike entities and ServiceNow tickets
2. `dedup_store`: Stores information for throttling and deduplication

## API Integration

The handlers in this project connect to ServiceNow through the OpenAPI Specification (OAS) defined in [api-integrations/servicenow.json](api-integrations/servicenow.json). Here's how the connection works:

1. In the manifest.yml file, the ServiceNow API integration is defined with the name `servicenow-foundry`.

2. The handlers reference this integration by name when communicating with ServiceNow through the Falcon API Integrations client.

3. When creating incidents, the handlers specify which operation to execute:
   - For standard incidents: `create_incident`
   - For SIR incidents: `create_sn_si_incident`

4. These operations are defined in the OAS file and map to specific ServiceNow API endpoints:
   - Standard incidents: `/api/now/table/incident` (POST)
   - SIR incidents: `/api/now/table/sn_si_incident` (POST)

This integration allows the app to create and manage tickets in ServiceNow while maintaining mappings between CrowdStrike entities and ServiceNow tickets in the custom storage.

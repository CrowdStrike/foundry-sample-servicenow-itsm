## ServiceNow ITSM Helper Foundry App

The ServiceNow ITSM Helper App is a Foundry application that facilitates integration between CrowdStrike and ServiceNow ITSM systems. It provides functionality for creating and managing incidents in ServiceNow, tracking entity mappings between systems, and implementing throttling mechanisms to control workflow execution.

## Key Capabilities
- **One-Way Alert Synchronization**: Provides building blocks to automatically create/update ServiceNow incidents from CrowdStrike alerts
- **Entity Mapping**: Tracks relationships between CrowdStrike entities and ServiceNow tickets
- **Multiple Ticket Type Support**: Creates standard incidents or Security Incident Response (SIR) tickets
- **Ticket as Container**: Associates multiple security objects with a single ServiceNow ticket
- **Time-Based Throttling**: Controls update flow to prevent duplicate tickets
- **Customizable Fields**: Maps CrowdStrike data to ServiceNow fields with support for custom fields

## Authentication Support
- Basic Authentication
- OAuth 2.0 Client Credentials grant

"""
Unit tests for ServiceNow ITSM Integration Function - Python Implementation
"""
# pylint: disable=missing-function-docstring,too-many-lines,protected-access,wrong-import-position,import-outside-toplevel,unused-argument

import json
import unittest
from unittest.mock import Mock, patch
from http import HTTPStatus

# Patch the FUNC.handler decorator to be a passthrough before importing main,
# so handler functions remain callable in tests.
import crowdstrike.foundry.function as _fdk

_orig_handler = _fdk.Function.handler

def _passthrough_handler(self, method, path):
    def decorator(func):
        _orig_handler(self, method=method, path=path)(func)
        return func
    return decorator

_fdk.Function.handler = _passthrough_handler

# Import the module under test (after patching)
import main


class TestSanitizeObjectKey(unittest.TestCase):
    """Test sanitize_object_key function"""

    def test_sanitize_valid_key(self):
        result = main.sanitize_object_key("test-key.123_value")
        self.assertEqual(result, "test-key.123_value")

    def test_sanitize_invalid_characters(self):
        result = main.sanitize_object_key("test@key#with$special%chars")
        self.assertEqual(result, "test_key_with_special_chars")

    def test_sanitize_exceeds_max_length(self):
        long_key = "a" * 1001
        with self.assertRaises(ValueError):
            main.sanitize_object_key(long_key)


class TestCreateTrackedEntityKey(unittest.TestCase):
    """Test create_tracked_entity_key function"""

    def test_create_key(self):
        result = main.create_tracked_entity_key("servicenow", "entity123")
        self.assertEqual(result, "servicenow.entity123")

    def test_create_key_with_special_chars(self):
        result = main.create_tracked_entity_key("service now", "entity@123")
        self.assertEqual(result, "service_now.entity_123")


class TestCalculateTimeBucket(unittest.TestCase):
    """Test calculate_time_bucket function"""

    def test_forever_bucket(self):
        result = main.calculate_time_bucket(main.TIME_BUCKET_FOREVER)
        self.assertEqual(result, "forever_bucket")

    @patch('main.datetime')
    def test_five_min_bucket(self, mock_datetime):
        from datetime import datetime, timezone
        # Mock current time to 2024-01-15 10:23:45
        mock_now = datetime(2024, 1, 15, 10, 23, 45, tzinfo=timezone.utc)
        mock_datetime.now.return_value = mock_now

        result = main.calculate_time_bucket(main.TIME_BUCKET_FIVE_MIN)
        # Should round down to 10:20 and format as "2024-01-15_10:20"
        self.assertEqual(result, "2024-01-15_10:20")

    @patch('main.datetime')
    def test_thirty_min_bucket(self, mock_datetime):
        from datetime import datetime, timezone
        # Mock current time to 2024-01-15 10:23:45
        mock_now = datetime(2024, 1, 15, 10, 23, 45, tzinfo=timezone.utc)
        mock_datetime.now.return_value = mock_now

        result = main.calculate_time_bucket(main.TIME_BUCKET_THIRTY_MIN)
        # Should round down to 10:00 and format as "2024-01-15_10:00"
        self.assertEqual(result, "2024-01-15_10:00")

    def test_invalid_bucket(self):
        with self.assertRaises(ValueError):
            main.calculate_time_bucket("invalid_bucket")


class TestCheckExternalEntityExists(unittest.TestCase):
    """Test check_external_entity_exists function"""

    def test_entity_exists(self):
        mock_client = Mock()
        mock_logger = Mock()

        # Mock successful response (bytes on success)
        entity_record = {
            "internal_entity_id": "entity123",
            "external_entity_id": "ext123",
            "external_system_id": "servicenow_incident"
        }
        mock_client.GetObject.return_value = json.dumps(entity_record).encode('utf-8')

        exists, record = main.check_external_entity_exists(
            mock_client, mock_logger, "entity123", "servicenow_incident"
        )

        self.assertTrue(exists)
        self.assertEqual(record["external_entity_id"], "ext123")
        mock_client.GetObject.assert_called_once_with(
            collection_name=main.COLLECTION_NAME_TRACKED_ENTITIES,
            object_key="servicenow_incident.entity123"
        )

    def test_entity_not_exists_404(self):
        """GetObject returns error dict on 404 — treated as not found."""
        mock_client = Mock()
        mock_logger = Mock()

        mock_client.GetObject.return_value = {"status_code": 404, "headers": {}, "body": {"errors": [], "resources": []}}

        exists, record = main.check_external_entity_exists(
            mock_client, mock_logger, "entity123", "servicenow_incident"
        )

        self.assertFalse(exists)
        self.assertIsNone(record)

    def test_entity_not_exists_500(self):
        """GetObject returns error dict on 500 — treated as not found."""
        mock_client = Mock()
        mock_logger = Mock()

        mock_client.GetObject.return_value = {
            "status_code": 500, "headers": {},
            "body": {"errors": [{"message": "server error"}], "resources": []}
        }

        exists, record = main.check_external_entity_exists(
            mock_client, mock_logger, "entity123", "servicenow_incident"
        )

        self.assertFalse(exists)
        self.assertIsNone(record)

    def test_entity_exists_but_wrong_system_id(self):
        mock_client = Mock()
        mock_logger = Mock()

        # Mock response with different system ID
        entity_record = {
            "internal_entity_id": "entity123",
            "external_entity_id": "ext123",
            "external_system_id": "other_system"
        }
        mock_client.GetObject.return_value = json.dumps(entity_record).encode('utf-8')

        exists, record = main.check_external_entity_exists(
            mock_client, mock_logger, "entity123", "servicenow_incident"
        )

        self.assertFalse(exists)
        self.assertIsNone(record)


class TestCreateOrUpdateExternalEntityMapping(unittest.TestCase):
    """Test create_or_update_external_entity_mapping function"""

    def test_create_mapping_success(self):
        mock_client = Mock()
        mock_logger = Mock()

        # Mock successful PutObject response
        mock_client.PutObject.return_value = {"status_code": 201}

        main.create_or_update_external_entity_mapping(
            mock_client, mock_logger,
            "entity123", "ext123", "servicenow_incident"
        )

        # Verify PutObject was called with correct args
        mock_client.PutObject.assert_called_once()
        call_args = mock_client.PutObject.call_args
        self.assertEqual(call_args[1]["body"]["internal_entity_id"], "entity123")
        self.assertEqual(call_args[1]["body"]["external_entity_id"], "ext123")
        self.assertEqual(call_args[1]["body"]["external_system_id"], "servicenow_incident")

    def test_create_mapping_failure(self):
        mock_client = Mock()
        mock_logger = Mock()

        # Mock failed PutObject response
        mock_client.PutObject.return_value = {"status_code": 500}

        with self.assertRaises(main.StorageError):
            main.create_or_update_external_entity_mapping(
                mock_client, mock_logger,
                "entity123", "ext123", "servicenow_incident"
            )


class TestCheckThrottlingStore(unittest.TestCase):
    """Test check_throttling_store function"""

    def test_first_occurrence_creates_record(self):
        mock_client = Mock()
        mock_logger = Mock()

        # GetObject returns error dict (not found), PutObject succeeds
        mock_client.GetObject.return_value = {"status_code": 404, "headers": {}, "body": {"errors": [], "resources": []}}
        mock_client.PutObject.return_value = {"status_code": 201}

        is_duplicate = main.check_throttling_store(
            mock_client, mock_logger,
            "entity123", "detection", "det456", main.TIME_BUCKET_FOREVER
        )

        self.assertFalse(is_duplicate)
        mock_client.GetObject.assert_called_once()
        mock_client.PutObject.assert_called_once()

    def test_first_occurrence_500_creates_record(self):
        """GetObject returns 500 error — still creates new record."""
        mock_client = Mock()
        mock_logger = Mock()

        # GetObject returns 500 error dict, PutObject succeeds
        mock_client.GetObject.return_value = {
            "status_code": 500, "headers": {},
            "body": {"errors": [{"message": "server error"}], "resources": []}
        }
        mock_client.PutObject.return_value = {"status_code": 201}

        is_duplicate = main.check_throttling_store(
            mock_client, mock_logger,
            "entity123", "detection", "det456", main.TIME_BUCKET_FOREVER
        )

        self.assertFalse(is_duplicate)
        mock_client.GetObject.assert_called_once()
        mock_client.PutObject.assert_called_once()

    def test_duplicate_occurrence(self):
        mock_client = Mock()
        mock_logger = Mock()

        # GetObject succeeds (record exists as bytes)
        dedup_record = {"time_bucket": main.TIME_BUCKET_FOREVER}
        mock_client.GetObject.return_value = json.dumps(dedup_record).encode('utf-8')

        is_duplicate = main.check_throttling_store(
            mock_client, mock_logger,
            "entity123", "detection", "det456", main.TIME_BUCKET_FOREVER
        )

        self.assertTrue(is_duplicate)
        # Should only call GetObject, not PutObject
        mock_client.GetObject.assert_called_once()
        mock_client.PutObject.assert_not_called()

    def test_invalid_time_bucket(self):
        mock_client = Mock()
        mock_logger = Mock()

        with self.assertRaises(ValueError):
            main.check_throttling_store(
                mock_client, mock_logger,
                "entity123", "detection", "det456", "invalid"
            )


class TestBuildRequestPayload(unittest.TestCase):
    """Test build_request_payload function"""

    def test_minimal_payload(self):
        body = {"short_description": "Test incident"}
        result = main.build_request_payload(body)

        self.assertEqual(result["short_description"], "Test incident")
        self.assertEqual(len(result), 1)

    def test_full_payload(self):
        body = {
            "short_description": "Test incident",
            "assignment_group": "IT Support",
            "category": "Software",
            "description": "Full description",
            "impact": "2",
            "severity": "3",
            "state": "1",
            "urgency": "2",
            "work_notes": "Initial notes"
        }
        result = main.build_request_payload(body)

        self.assertEqual(len(result), 9)
        self.assertEqual(result["assignment_group"], "IT Support")
        self.assertEqual(result["impact"], "2")

    def test_payload_with_custom_fields(self):
        body = {
            "short_description": "Test incident",
            "custom_fields": '{"custom_field1": "value1", "custom_field2": "value2"}'
        }
        result = main.build_request_payload(body)

        self.assertEqual(result["short_description"], "Test incident")
        self.assertEqual(result["custom_field1"], "value1")
        self.assertEqual(result["custom_field2"], "value2")


class TestCheckIfExtEntityExistsHandler(unittest.TestCase):
    """Test check_if_ext_entity_exists_handler"""

    @patch('main.create_falcon_clients')
    @patch('main.check_external_entity_exists')
    def test_entity_exists(self, mock_check, mock_create_clients):
        mock_logger = Mock()
        mock_client = Mock()
        mock_create_clients.return_value = (Mock(), mock_client)

        # Mock entity exists
        mock_check.return_value = (True, {
            "external_entity_id": "ext123",
            "external_system_id": "servicenow_incident"
        })

        mock_request = Mock()
        mock_request.trace_id = "trace-123"
        mock_request.body = {
            "internal_entity_id": "entity123",
            "external_system_id": "servicenow_incident"
        }

        response = main.check_if_ext_entity_exists_handler(mock_request, None, mock_logger)

        self.assertEqual(response.code, HTTPStatus.OK)
        self.assertTrue(response.body["exists"])
        self.assertEqual(response.body["ext_id"], "ext123")

    @patch('main.create_falcon_clients')
    def test_entity_not_exists_404(self, mock_create_clients):
        """Entity doesn't exist (404) returns exists=false with HTTP 200."""
        mock_logger = Mock()
        mock_client = Mock()
        mock_create_clients.return_value = (Mock(), mock_client)

        # GetObject returns error dict (404)
        mock_client.GetObject.return_value = {"status_code": 404, "headers": {}, "body": {"errors": [], "resources": []}}

        mock_request = Mock()
        mock_request.trace_id = "trace-123"
        mock_request.body = {
            "internal_entity_id": "entity123",
            "external_system_id": "servicenow_incident"
        }

        response = main.check_if_ext_entity_exists_handler(mock_request, None, mock_logger)

        self.assertEqual(response.code, HTTPStatus.OK)
        self.assertFalse(response.body["exists"])

    @patch('main.create_falcon_clients')
    @patch('main.check_external_entity_exists')
    def test_entity_not_exists(self, mock_check, mock_create_clients):
        mock_logger = Mock()
        mock_client = Mock()
        mock_create_clients.return_value = (Mock(), mock_client)

        # Mock entity does not exist
        mock_check.return_value = (False, None)

        mock_request = Mock()
        mock_request.trace_id = "trace-123"
        mock_request.body = {
            "internal_entity_id": "entity123",
            "external_system_id": "servicenow_incident"
        }

        response = main.check_if_ext_entity_exists_handler(mock_request, None, mock_logger)

        self.assertEqual(response.code, HTTPStatus.OK)
        self.assertFalse(response.body["exists"])

    def test_missing_required_fields(self):
        mock_logger = Mock()
        mock_request = Mock()
        mock_request.trace_id = "trace-123"
        mock_request.body = {"internal_entity_id": "entity123"}  # Missing external_system_id

        response = main.check_if_ext_entity_exists_handler(mock_request, None, mock_logger)

        self.assertEqual(response.code, HTTPStatus.BAD_REQUEST)
        self.assertTrue(len(response.errors) > 0)


class TestCreateEntityMappingHandler(unittest.TestCase):
    """Test create_entity_mapping_handler"""

    @patch('main.create_falcon_clients')
    @patch('main.create_or_update_external_entity_mapping')
    def test_create_mapping_success(self, mock_create_mapping, mock_create_clients):
        mock_logger = Mock()
        mock_client = Mock()
        mock_create_clients.return_value = (Mock(), mock_client)

        mock_request = Mock()
        mock_request.trace_id = "trace-123"
        mock_request.body = {
            "internal_entity_id": "entity123",
            "external_entity_id": "ext123",
            "external_system_id": "servicenow_incident"
        }

        response = main.create_entity_mapping_handler(mock_request, None, mock_logger)

        self.assertEqual(response.code, HTTPStatus.CREATED)
        self.assertEqual(response.body["internal_entity_id"], "entity123")
        self.assertEqual(response.body["external_entity_id"], "ext123")
        mock_create_mapping.assert_called_once()

    def test_missing_required_fields(self):
        mock_logger = Mock()
        mock_request = Mock()
        mock_request.trace_id = "trace-123"
        mock_request.body = {
            "internal_entity_id": "entity123",
            "external_entity_id": "ext123"
            # Missing external_system_id
        }

        response = main.create_entity_mapping_handler(mock_request, None, mock_logger)

        self.assertEqual(response.code, HTTPStatus.BAD_REQUEST)


class TestCreateIncidentHandlers(unittest.TestCase):
    """Test create_incident_handler and create_sir_incident_handler"""

    @patch('main.create_falcon_clients')
    @patch('main.check_external_entity_exists')
    def test_incident_already_exists(self, mock_check, mock_create_clients):
        mock_logger = Mock()
        mock_api_integrations = Mock()
        mock_client = Mock()
        mock_create_clients.return_value = (mock_api_integrations, mock_client)

        # Mock that ticket already exists
        mock_check.return_value = (True, {
            "external_entity_id": "existing-ticket-123"
        })

        mock_request = Mock()
        mock_request.trace_id = "trace-123"
        mock_request.body = {
            "config_id": "config123",
            "entity_id": "entity123",
            "short_description": "Test incident"
        }

        response = main.create_incident_handler(mock_request, None, mock_logger)

        self.assertEqual(response.code, HTTPStatus.OK)
        self.assertTrue(response.body["exists"])
        self.assertEqual(response.body["ticket_id"], "existing-ticket-123")
        # Should not call API integration if ticket exists
        mock_api_integrations.execute_command.assert_not_called()

    @patch('main.create_falcon_clients')
    @patch('main.check_external_entity_exists')
    @patch('main.create_or_update_external_entity_mapping')
    def test_create_new_incident(self, mock_create_mapping, mock_check, mock_create_clients):
        mock_logger = Mock()
        mock_api_integrations = Mock()
        mock_client = Mock()
        mock_create_clients.return_value = (mock_api_integrations, mock_client)

        # Mock that ticket does not exist
        mock_check.return_value = (False, None)

        # Mock successful API integration response
        mock_api_integrations.execute_command.return_value = {
            "status_code": 200,
            "body": {
                "resources": [{
                    "response_body": {
                        "result": {
                            "sys_id": "new-ticket-123",
                            "sys_class_name": "incident"
                        }
                    }
                }]
            }
        }

        mock_request = Mock()
        mock_request.trace_id = "trace-123"
        mock_request.body = {
            "config_id": "config123",
            "entity_id": "entity123",
            "short_description": "Test incident",
            "impact": "2",
            "urgency": "2"
        }

        response = main.create_incident_handler(mock_request, None, mock_logger)

        self.assertEqual(response.code, HTTPStatus.CREATED)
        self.assertFalse(response.body["exists"])
        self.assertEqual(response.body["ticket_id"], "new-ticket-123")
        self.assertEqual(response.body["ticket_type"], "incident")

        # Verify API integration was called
        mock_api_integrations.execute_command.assert_called_once()
        call_args = mock_api_integrations.execute_command.call_args[1]
        self.assertEqual(call_args["body"]["resources"][0]["definition_id"], main.PLUGIN_DEF_ID_SERVICENOW)

        # Verify entity mapping was created
        mock_create_mapping.assert_called_once()

    @patch('main.create_falcon_clients')
    @patch('main.create_or_update_external_entity_mapping')
    def test_create_new_incident_entity_404(self, mock_create_mapping, mock_create_clients):
        """Entity check returns 404, triggers new ticket creation via ServiceNow."""
        mock_logger = Mock()
        mock_api_integrations = Mock()
        mock_client = Mock()
        mock_create_clients.return_value = (mock_api_integrations, mock_client)

        # GetObject returns 404 on entity check
        mock_client.GetObject.return_value = {"status_code": 404, "body": {}}

        # Mock successful ServiceNow response
        mock_api_integrations.execute_command.return_value = {
            "status_code": 200,
            "body": {
                "resources": [{
                    "response_body": {
                        "result": {
                            "sys_id": "new-ticket-456",
                            "sys_class_name": "incident"
                        }
                    }
                }]
            }
        }

        mock_request = Mock()
        mock_request.trace_id = "trace-123"
        mock_request.context = {}
        mock_request.body = {
            "config_id": "config123",
            "entity_id": "entity123",
            "short_description": "Test incident"
        }

        response = main.create_incident_impl(
            mock_request, mock_logger,
            main.PLUGIN_OP_ID_SERVICENOW_CREATE_INCIDENT,
            "incident",
            main.EXTERNAL_SYSTEM_ID_SERVICENOW_INCIDENT
        )

        self.assertEqual(response.code, HTTPStatus.CREATED)
        self.assertFalse(response.body["exists"])
        self.assertEqual(response.body["ticket_id"], "new-ticket-456")
        mock_api_integrations.execute_command.assert_called_once()

    @patch('main.create_falcon_clients')
    @patch('main.check_external_entity_exists')
    def test_execute_command_failure(self, mock_check, mock_create_clients):
        mock_logger = Mock()
        mock_api_integrations = Mock()
        mock_client = Mock()
        mock_create_clients.return_value = (mock_api_integrations, mock_client)

        mock_check.return_value = (False, None)

        # Mock failed execute_command response
        mock_api_integrations.execute_command.return_value = {
            "status_code": 403,
            "body": {
                "errors": [{"message": "received invalid status code from plugin: 403"}]
            }
        }

        mock_request = Mock()
        mock_request.trace_id = "trace-123"
        mock_request.body = {
            "config_id": "config123",
            "entity_id": "entity123",
            "short_description": "Test incident"
        }

        response = main.create_incident_handler(mock_request, None, mock_logger)

        self.assertEqual(response.code, HTTPStatus.INTERNAL_SERVER_ERROR)
        self.assertIn("403", response.errors[0].message)

    @patch('main.create_falcon_clients')
    @patch('main.check_external_entity_exists')
    def test_servicenow_error_response(self, mock_check, mock_create_clients):
        mock_logger = Mock()
        mock_api_integrations = Mock()
        mock_client = Mock()
        mock_create_clients.return_value = (mock_api_integrations, mock_client)

        # Mock that ticket does not exist
        mock_check.return_value = (False, None)

        # Mock ServiceNow error response
        mock_api_integrations.execute_command.return_value = {
            "status_code": 200,
            "body": {
                "resources": [{
                    "response_body": {
                        "error": "ServiceNow validation error"
                    }
                }]
            }
        }

        mock_request = Mock()
        mock_request.trace_id = "trace-123"
        mock_request.body = {
            "config_id": "config123",
            "entity_id": "entity123",
            "short_description": "Test incident"
        }

        response = main.create_incident_handler(mock_request, None, mock_logger)

        self.assertEqual(response.code, HTTPStatus.INTERNAL_SERVER_ERROR)
        self.assertTrue(len(response.errors) > 0)
        self.assertIn("ServiceNow Error", response.errors[0].message)

    def test_missing_required_fields(self):
        mock_logger = Mock()
        mock_request = Mock()
        mock_request.trace_id = "trace-123"
        mock_request.body = {
            "config_id": "config123"
            # Missing entity_id
        }

        response = main.create_incident_handler(mock_request, None, mock_logger)

        self.assertEqual(response.code, HTTPStatus.BAD_REQUEST)


class TestThrottleHandler(unittest.TestCase):
    """Test throttle_handler"""

    @patch('main.create_falcon_clients')
    @patch('main.check_throttling_store')
    def test_action_allowed(self, mock_check_throttle, mock_create_clients):
        mock_logger = Mock()
        mock_client = Mock()
        mock_create_clients.return_value = (Mock(), mock_client)

        # Mock that it's not a duplicate
        mock_check_throttle.return_value = False

        mock_request = Mock()
        mock_request.trace_id = "trace-123"
        mock_request.body = {
            "internal_entity_id": "entity123",
            "dedup_obj_type": "detection",
            "dedup_obj_id": "det456",
            "time_bucket": "forever"
        }

        response = main.throttle_handler(mock_request, None, mock_logger)

        self.assertEqual(response.code, HTTPStatus.OK)
        self.assertTrue(response.body["allowed"])

    @patch('main.create_falcon_clients')
    def test_action_allowed_on_404(self, mock_create_clients):
        """Record doesn't exist (404) creates new record and allows action."""
        mock_logger = Mock()
        mock_client = Mock()
        mock_create_clients.return_value = (Mock(), mock_client)

        # GetObject returns 404, PutObject succeeds
        mock_client.GetObject.return_value = {"status_code": 404, "body": {}}
        mock_client.PutObject.return_value = {"status_code": 201}

        mock_request = Mock()
        mock_request.trace_id = "trace-123"
        mock_request.body = {
            "internal_entity_id": "entity123",
            "dedup_obj_type": "detection",
            "dedup_obj_id": "det456",
            "time_bucket": "forever"
        }

        response = main.throttle_handler(mock_request, None, mock_logger)

        self.assertEqual(response.code, HTTPStatus.OK)
        self.assertTrue(response.body["allowed"])
        # Should have called GetObject then PutObject
        mock_client.GetObject.assert_called_once()
        mock_client.PutObject.assert_called_once()

    @patch('main.create_falcon_clients')
    @patch('main.check_throttling_store')
    def test_action_not_allowed(self, mock_check_throttle, mock_create_clients):
        mock_logger = Mock()
        mock_client = Mock()
        mock_create_clients.return_value = (Mock(), mock_client)

        # Mock that it's a duplicate
        mock_check_throttle.return_value = True

        mock_request = Mock()
        mock_request.trace_id = "trace-123"
        mock_request.body = {
            "internal_entity_id": "entity123",
            "dedup_obj_type": "detection",
            "dedup_obj_id": "det456",
            "time_bucket": "forever"
        }

        response = main.throttle_handler(mock_request, None, mock_logger)

        self.assertEqual(response.code, HTTPStatus.OK)
        self.assertFalse(response.body["allowed"])

    def test_missing_required_fields(self):
        mock_logger = Mock()
        mock_request = Mock()
        mock_request.trace_id = "trace-123"
        mock_request.body = {
            "internal_entity_id": "entity123",
            "dedup_obj_type": "detection"
            # Missing dedup_obj_id and time_bucket
        }

        response = main.throttle_handler(mock_request, None, mock_logger)

        self.assertEqual(response.code, HTTPStatus.BAD_REQUEST)


if __name__ == '__main__':
    unittest.main()

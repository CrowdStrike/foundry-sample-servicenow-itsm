package storage

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"regexp"
	"strings"

	"github.com/crowdstrike/gofalcon/falcon/client/custom_storage"
)

const (
	CollectionNameTrackedEntities = "tracked_entities"
	CollectionNameDedupStore      = "dedup_store"
)

type StorageService interface {
	GetObject(params *custom_storage.GetObjectParams, writer io.Writer, opts ...custom_storage.ClientOption) (*custom_storage.GetObjectOK, error)
	PutObject(params *custom_storage.PutObjectParams, opts ...custom_storage.ClientOption) (*custom_storage.PutObjectOK, error)
}

// CheckThrottlingStore check if a combination of ids is already known.
// Returns true if already exists, false if it doesn't
func CheckThrottlingStore(ctx context.Context, storageService StorageService, logger *slog.Logger, internalEntityID, dedupObjType, dedupObjId, timeBucket string) (bool, error) {
	// Convert timeBucket string to TimeBucket type
	tb := TimeBucket(timeBucket)

	// Validate timeBucket against supported enum values
	if tb != TimeBucketForever && tb != TimeBucketFiveMin && tb != TimeBucketThirtyMin {
		return false, fmt.Errorf("unsupported time bucket value: %s (must be one of: %s, %s, %s)",
			timeBucket, TimeBucketForever, TimeBucketFiveMin, TimeBucketThirtyMin)
	}

	// Calculate the current bucket
	currentBucket, err := calculateTimeBucket(tb)
	if err != nil {
		return false, fmt.Errorf("failed to calculate time bucket: %w", err)
	}

	combined := strings.Join([]string{internalEntityID, dedupObjType, dedupObjId, currentBucket}, ":")
	hasher := md5.New()
	hasher.Write([]byte(combined))
	dedupKey := hex.EncodeToString(hasher.Sum(nil))

	getCommand := &custom_storage.GetObjectParams{
		CollectionName: CollectionNameDedupStore,
		ObjectKey:      dedupKey,
		Context:        ctx,
	}

	buf := new(bytes.Buffer)
	_, err = storageService.GetObject(getCommand, buf)
	if err != nil {
		// Check if object doesn't exist
		if strings.Contains(err.Error(), "status 404") {
			// Record doesn't exist, create a new one
			newDedupStoreRecord := DedupStoreRecord{TimeBucket: tb}
			var uploadBuf bytes.Buffer
			if err := json.NewEncoder(&uploadBuf).Encode(newDedupStoreRecord); err != nil {
				logger.Error("failed to encode dedup record", "error", err)
				return false, fmt.Errorf("failed to encode dedup record: %w", err)
			}

			_, err = storageService.PutObject(&custom_storage.PutObjectParams{
				CollectionName: CollectionNameDedupStore,
				ObjectKey:      dedupKey,
				Body:           io.NopCloser(&uploadBuf),
				Context:        ctx,
			})
			if err != nil {
				logger.Error("failed to store dedup record", "error", err)
				return false, fmt.Errorf("failed to store dedup record: %w", err)
			}

			return false, nil
		}

		return false, fmt.Errorf("failed to check dedup record: %w", err)
	}

	// Record exists, unmarshal for validation/logging if needed
	var dedupStoreRecord DedupStoreRecord
	if err := json.Unmarshal(buf.Bytes(), &dedupStoreRecord); err != nil {
		return false, fmt.Errorf("failed to unmarshal dedup record: %w", err)
	}

	// Record exists, return true
	return true, nil
}

// CreateTrackedEntityKey generates a unique key for tracked entities by combining
// the external system ID and internal entity ID
func CreateTrackedEntityKey(externalSystemID, internalEntityID string) (string, error) {
	combined := fmt.Sprintf("%s.%s", externalSystemID, internalEntityID)
	return sanitizeObjectKey(combined)
}

// CheckExternalEntityExists checks if an external entity mapping exists for the given internal entity ID
// If externalSystemID is provided, it will also check if the external system ID matches
func CheckExternalEntityExists(ctx context.Context, storageService StorageService, logger *slog.Logger, internalEntityID string, externalSystemID string) (bool, *ExternalEntityRecord, error) {
	key, err := CreateTrackedEntityKey(externalSystemID, internalEntityID)
	if err != nil {
		return false, nil, fmt.Errorf("failed to create tracked entity key: %w", err)
	}

	getCommand := &custom_storage.GetObjectParams{
		CollectionName: CollectionNameTrackedEntities,
		ObjectKey:      key,
		Context:        ctx,
	}

	buf := new(bytes.Buffer)
	_, err = storageService.GetObject(getCommand, buf)
	if err != nil {
		if strings.Contains(err.Error(), "status 404") {
			return false, nil, nil
		}

		return false, nil, fmt.Errorf("failed to check if external entity exists: %w", err)
	}

	var extRecord ExternalEntityRecord
	if err := json.Unmarshal(buf.Bytes(), &extRecord); err != nil {
		return true, nil, fmt.Errorf("failed to unmarshal external entity record: %w", err)
	}

	// If externalSystemID is provided, check if it matches
	if externalSystemID != "" && extRecord.ExternalSystemID != externalSystemID {
		return false, nil, nil
	}

	// Record exists, matches the external system ID (if provided), and was successfully unmarshaled
	return true, &extRecord, nil
}

// CreateOrUpdateExternalEntityMapping stores a mapping between internal and external entities in custom storage
func CreateOrUpdateExternalEntityMapping(ctx context.Context, storageService StorageService, logger *slog.Logger, record ExternalEntityRecord) error {
	// Store the mapping in the custom storage using the Upload method
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(record); err != nil {
		logger.Error("failed to encode entity record", "error", err)
		return fmt.Errorf("error encoding entity record: %w", err)
	}

	key, err := CreateTrackedEntityKey(record.ExternalSystemID, record.InternalEntityID)
	if err != nil {
		logger.Error("failed to create tracked entity key", "error", err)
		return fmt.Errorf("error creating tracked entity key: %w", err)
	}

	_, err = storageService.PutObject(&custom_storage.PutObjectParams{
		CollectionName: CollectionNameTrackedEntities,
		ObjectKey:      key,
		Body:           io.NopCloser(&buf),
		Context:        ctx,
	})
	if err != nil {
		logger.Error("failed to upload entity mapping", "error", err)
		return fmt.Errorf("error storing entity mapping in collection: %w", err)
	}

	logger.Info("successfully stored entity mapping",
		"internal_id", record.InternalEntityID,
		"external_id", record.ExternalEntityID,
		"system_id", record.ExternalSystemID)

	return nil
}

func sanitizeObjectKey(input string) (string, error) {
	// Replace disallowed characters with underscore
	re := regexp.MustCompile("[^a-zA-Z0-9._-]")
	sanitized := re.ReplaceAllString(input, "_")

	// Check length constraints
	if len(sanitized) > 1000 {
		return "", fmt.Errorf("object key exceeds maximum length of 1000 characters: %d", len(sanitized))
	}

	return sanitized, nil
}

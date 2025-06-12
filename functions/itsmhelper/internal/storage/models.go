package storage

// ExternalEntityRecord represents a mapping between internal entities and external ITSM system entities
type ExternalEntityRecord struct {
	InternalEntityID string `json:"internal_entity_id"`

	ExternalEntityID string `json:"external_entity_id"`
	ExternalSystemID string `json:"external_system_id"`
}

// TimeBucket represents time interval for time-based deduping
type TimeBucket string

const (
	TimeBucketForever   TimeBucket = "forever"
	TimeBucketFiveMin   TimeBucket = "5 minutes"
	TimeBucketThirtyMin TimeBucket = "30 minutes"
)

type DedupStoreRecord struct {
	TimeBucket TimeBucket `json:"time_bucket"`
}

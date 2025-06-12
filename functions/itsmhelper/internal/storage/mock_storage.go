package storage

import (
	"io"

	"github.com/crowdstrike/gofalcon/falcon/client/custom_storage"
	"github.com/go-openapi/runtime"
)

// MockStorageService is a mock implementation of the StorageService interface for testing
type MockStorageService struct {
	GetObjectFunc                  func(*custom_storage.GetObjectParams, io.Writer, ...custom_storage.ClientOption) (*custom_storage.GetObjectOK, error)
	PutObjectFunc                  func(*custom_storage.PutObjectParams, ...custom_storage.ClientOption) (*custom_storage.PutObjectOK, error)
	DeleteFunc                     func(*custom_storage.DeleteObjectParams, ...custom_storage.ClientOption) (*custom_storage.DeleteObjectOK, error)
	DeleteVersionedObjectFunc      func(*custom_storage.DeleteVersionedObjectParams, ...custom_storage.ClientOption) (*custom_storage.DeleteVersionedObjectOK, error)
	GetVersionedObjectFunc         func(*custom_storage.GetVersionedObjectParams, io.Writer, ...custom_storage.ClientOption) (*custom_storage.GetVersionedObjectOK, error)
	GetVersionedObjectMetadataFunc func(*custom_storage.GetVersionedObjectMetadataParams, ...custom_storage.ClientOption) (*custom_storage.GetVersionedObjectMetadataOK, error)
	ListObjectsFunc                func(*custom_storage.ListObjectsParams, ...custom_storage.ClientOption) (*custom_storage.ListObjectsOK, error)
	ListObjectsByVersionFunc       func(*custom_storage.ListObjectsByVersionParams, ...custom_storage.ClientOption) (*custom_storage.ListObjectsByVersionOK, error)
	MetadataFunc                   func(*custom_storage.GetObjectMetadataParams, ...custom_storage.ClientOption) (*custom_storage.GetObjectMetadataOK, error)
	PutObjectByVersionFunc         func(*custom_storage.PutObjectByVersionParams, ...custom_storage.ClientOption) (*custom_storage.PutObjectByVersionOK, error)
	SearchObjectsFunc              func(*custom_storage.SearchObjectsParams, ...custom_storage.ClientOption) (*custom_storage.SearchObjectsOK, error)
	SearchObjectsByVersionFunc     func(*custom_storage.SearchObjectsByVersionParams, ...custom_storage.ClientOption) (*custom_storage.SearchObjectsByVersionOK, error)
}

// GetObject implements the GetObject method for the mock
func (m *MockStorageService) GetObject(params *custom_storage.GetObjectParams, writer io.Writer, opts ...custom_storage.ClientOption) (*custom_storage.GetObjectOK, error) {
	if m.GetObjectFunc != nil {
		return m.GetObjectFunc(params, writer, opts...)
	}
	return nil, nil
}

// PutObject implements the PutObject method for the mock
func (m *MockStorageService) PutObject(params *custom_storage.PutObjectParams, opts ...custom_storage.ClientOption) (*custom_storage.PutObjectOK, error) {
	if m.PutObjectFunc != nil {
		return m.PutObjectFunc(params, opts...)
	}
	return nil, nil
}

func (m *MockStorageService) DeleteObject(params *custom_storage.DeleteObjectParams, opts ...custom_storage.ClientOption) (*custom_storage.DeleteObjectOK, error) {
	panic("not implemented")
}

func (m *MockStorageService) DeleteVersionedObject(params *custom_storage.DeleteVersionedObjectParams, opts ...custom_storage.ClientOption) (*custom_storage.DeleteVersionedObjectOK, error) {
	panic("not implemented")
}

func (m *MockStorageService) DescribeCollection(params *custom_storage.DescribeCollectionParams, opts ...custom_storage.ClientOption) (*custom_storage.DescribeCollectionOK, error) {
	panic("not implemented")
}

func (m *MockStorageService) DescribeCollections(params *custom_storage.DescribeCollectionsParams, opts ...custom_storage.ClientOption) (*custom_storage.DescribeCollectionsOK, error) {
	panic("not implemented")
}

func (m *MockStorageService) GetObjectMetadata(params *custom_storage.GetObjectMetadataParams, opts ...custom_storage.ClientOption) (*custom_storage.GetObjectMetadataOK, error) {
	panic("not implemented")
}

func (m *MockStorageService) GetSchema(params *custom_storage.GetSchemaParams, writer io.Writer, opts ...custom_storage.ClientOption) (*custom_storage.GetSchemaOK, error) {
	panic("not implemented")
}

func (m *MockStorageService) GetSchemaMetadata(params *custom_storage.GetSchemaMetadataParams, opts ...custom_storage.ClientOption) (*custom_storage.GetSchemaMetadataOK, error) {
	panic("not implemented")
}

func (m *MockStorageService) GetVersionedObject(params *custom_storage.GetVersionedObjectParams, writer io.Writer, opts ...custom_storage.ClientOption) (*custom_storage.GetVersionedObjectOK, error) {
	panic("not implemented")
}

func (m *MockStorageService) GetVersionedObjectMetadata(params *custom_storage.GetVersionedObjectMetadataParams, opts ...custom_storage.ClientOption) (*custom_storage.GetVersionedObjectMetadataOK, error) {
	panic("not implemented")
}

func (m *MockStorageService) ListCollections(params *custom_storage.ListCollectionsParams, opts ...custom_storage.ClientOption) (*custom_storage.ListCollectionsOK, error) {
	panic("not implemented")
}

func (m *MockStorageService) ListObjects(params *custom_storage.ListObjectsParams, opts ...custom_storage.ClientOption) (*custom_storage.ListObjectsOK, error) {
	panic("not implemented")
}

func (m *MockStorageService) ListObjectsByVersion(params *custom_storage.ListObjectsByVersionParams, opts ...custom_storage.ClientOption) (*custom_storage.ListObjectsByVersionOK, error) {
	panic("not implemented")
}

func (m *MockStorageService) ListSchemas(params *custom_storage.ListSchemasParams, opts ...custom_storage.ClientOption) (*custom_storage.ListSchemasOK, error) {
	panic("not implemented")
}

func (m *MockStorageService) PutObjectByVersion(params *custom_storage.PutObjectByVersionParams, opts ...custom_storage.ClientOption) (*custom_storage.PutObjectByVersionOK, error) {
	panic("not implemented")
}

func (m *MockStorageService) SearchObjects(params *custom_storage.SearchObjectsParams, opts ...custom_storage.ClientOption) (*custom_storage.SearchObjectsOK, error) {
	panic("not implemented")
}

func (m *MockStorageService) SearchObjectsByVersion(params *custom_storage.SearchObjectsByVersionParams, opts ...custom_storage.ClientOption) (*custom_storage.SearchObjectsByVersionOK, error) {
	panic("not implemented")
}

func (m *MockStorageService) SetTransport(transport runtime.ClientTransport) {
	panic("not implemented")
}

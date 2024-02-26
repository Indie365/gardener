// Code generated by MockGen. DO NOT EDIT.
// Source: k8s.io/client-go/discovery (interfaces: DiscoveryInterface)
//
// Generated by this command:
//
//	mockgen -package discovery -destination=mocks.go k8s.io/client-go/discovery DiscoveryInterface
//

// Package discovery is a generated GoMock package.
package discovery

import (
	reflect "reflect"

	openapi_v2 "github.com/google/gnostic-models/openapiv2"
	gomock "go.uber.org/mock/gomock"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	version "k8s.io/apimachinery/pkg/version"
	discovery "k8s.io/client-go/discovery"
	openapi "k8s.io/client-go/openapi"
	rest "k8s.io/client-go/rest"
)

// MockDiscoveryInterface is a mock of DiscoveryInterface interface.
type MockDiscoveryInterface struct {
	ctrl     *gomock.Controller
	recorder *MockDiscoveryInterfaceMockRecorder
}

// MockDiscoveryInterfaceMockRecorder is the mock recorder for MockDiscoveryInterface.
type MockDiscoveryInterfaceMockRecorder struct {
	mock *MockDiscoveryInterface
}

// NewMockDiscoveryInterface creates a new mock instance.
func NewMockDiscoveryInterface(ctrl *gomock.Controller) *MockDiscoveryInterface {
	mock := &MockDiscoveryInterface{ctrl: ctrl}
	mock.recorder = &MockDiscoveryInterfaceMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockDiscoveryInterface) EXPECT() *MockDiscoveryInterfaceMockRecorder {
	return m.recorder
}

// OpenAPISchema mocks base method.
func (m *MockDiscoveryInterface) OpenAPISchema() (*openapi_v2.Document, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "OpenAPISchema")
	ret0, _ := ret[0].(*openapi_v2.Document)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// OpenAPISchema indicates an expected call of OpenAPISchema.
func (mr *MockDiscoveryInterfaceMockRecorder) OpenAPISchema() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "OpenAPISchema", reflect.TypeOf((*MockDiscoveryInterface)(nil).OpenAPISchema))
}

// OpenAPIV3 mocks base method.
func (m *MockDiscoveryInterface) OpenAPIV3() openapi.Client {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "OpenAPIV3")
	ret0, _ := ret[0].(openapi.Client)
	return ret0
}

// OpenAPIV3 indicates an expected call of OpenAPIV3.
func (mr *MockDiscoveryInterfaceMockRecorder) OpenAPIV3() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "OpenAPIV3", reflect.TypeOf((*MockDiscoveryInterface)(nil).OpenAPIV3))
}

// RESTClient mocks base method.
func (m *MockDiscoveryInterface) RESTClient() rest.Interface {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "RESTClient")
	ret0, _ := ret[0].(rest.Interface)
	return ret0
}

// RESTClient indicates an expected call of RESTClient.
func (mr *MockDiscoveryInterfaceMockRecorder) RESTClient() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "RESTClient", reflect.TypeOf((*MockDiscoveryInterface)(nil).RESTClient))
}

// ServerGroups mocks base method.
func (m *MockDiscoveryInterface) ServerGroups() (*v1.APIGroupList, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ServerGroups")
	ret0, _ := ret[0].(*v1.APIGroupList)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ServerGroups indicates an expected call of ServerGroups.
func (mr *MockDiscoveryInterfaceMockRecorder) ServerGroups() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ServerGroups", reflect.TypeOf((*MockDiscoveryInterface)(nil).ServerGroups))
}

// ServerGroupsAndResources mocks base method.
func (m *MockDiscoveryInterface) ServerGroupsAndResources() ([]*v1.APIGroup, []*v1.APIResourceList, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ServerGroupsAndResources")
	ret0, _ := ret[0].([]*v1.APIGroup)
	ret1, _ := ret[1].([]*v1.APIResourceList)
	ret2, _ := ret[2].(error)
	return ret0, ret1, ret2
}

// ServerGroupsAndResources indicates an expected call of ServerGroupsAndResources.
func (mr *MockDiscoveryInterfaceMockRecorder) ServerGroupsAndResources() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ServerGroupsAndResources", reflect.TypeOf((*MockDiscoveryInterface)(nil).ServerGroupsAndResources))
}

// ServerPreferredNamespacedResources mocks base method.
func (m *MockDiscoveryInterface) ServerPreferredNamespacedResources() ([]*v1.APIResourceList, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ServerPreferredNamespacedResources")
	ret0, _ := ret[0].([]*v1.APIResourceList)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ServerPreferredNamespacedResources indicates an expected call of ServerPreferredNamespacedResources.
func (mr *MockDiscoveryInterfaceMockRecorder) ServerPreferredNamespacedResources() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ServerPreferredNamespacedResources", reflect.TypeOf((*MockDiscoveryInterface)(nil).ServerPreferredNamespacedResources))
}

// ServerPreferredResources mocks base method.
func (m *MockDiscoveryInterface) ServerPreferredResources() ([]*v1.APIResourceList, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ServerPreferredResources")
	ret0, _ := ret[0].([]*v1.APIResourceList)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ServerPreferredResources indicates an expected call of ServerPreferredResources.
func (mr *MockDiscoveryInterfaceMockRecorder) ServerPreferredResources() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ServerPreferredResources", reflect.TypeOf((*MockDiscoveryInterface)(nil).ServerPreferredResources))
}

// ServerResourcesForGroupVersion mocks base method.
func (m *MockDiscoveryInterface) ServerResourcesForGroupVersion(arg0 string) (*v1.APIResourceList, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ServerResourcesForGroupVersion", arg0)
	ret0, _ := ret[0].(*v1.APIResourceList)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ServerResourcesForGroupVersion indicates an expected call of ServerResourcesForGroupVersion.
func (mr *MockDiscoveryInterfaceMockRecorder) ServerResourcesForGroupVersion(arg0 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ServerResourcesForGroupVersion", reflect.TypeOf((*MockDiscoveryInterface)(nil).ServerResourcesForGroupVersion), arg0)
}

// ServerVersion mocks base method.
func (m *MockDiscoveryInterface) ServerVersion() (*version.Info, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ServerVersion")
	ret0, _ := ret[0].(*version.Info)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ServerVersion indicates an expected call of ServerVersion.
func (mr *MockDiscoveryInterfaceMockRecorder) ServerVersion() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ServerVersion", reflect.TypeOf((*MockDiscoveryInterface)(nil).ServerVersion))
}

// WithLegacy mocks base method.
func (m *MockDiscoveryInterface) WithLegacy() discovery.DiscoveryInterface {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "WithLegacy")
	ret0, _ := ret[0].(discovery.DiscoveryInterface)
	return ret0
}

// WithLegacy indicates an expected call of WithLegacy.
func (mr *MockDiscoveryInterfaceMockRecorder) WithLegacy() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "WithLegacy", reflect.TypeOf((*MockDiscoveryInterface)(nil).WithLegacy))
}

// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/gardener/gardener/pkg/client/kubernetes (interfaces: Interface)
//
// Generated by this command:
//
//	mockgen -package mock -destination=mocks.go github.com/gardener/gardener/pkg/client/kubernetes Interface
//

// Package mock is a generated GoMock package.
package mock

import (
	context "context"
	reflect "reflect"

	chartrenderer "github.com/gardener/gardener/pkg/chartrenderer"
	kubernetes "github.com/gardener/gardener/pkg/client/kubernetes"
	gomock "go.uber.org/mock/gomock"
	version "k8s.io/apimachinery/pkg/version"
	kubernetes0 "k8s.io/client-go/kubernetes"
	rest "k8s.io/client-go/rest"
	cache "sigs.k8s.io/controller-runtime/pkg/cache"
	client "sigs.k8s.io/controller-runtime/pkg/client"
)

// MockInterface is a mock of Interface interface.
type MockInterface struct {
	ctrl     *gomock.Controller
	recorder *MockInterfaceMockRecorder
}

// MockInterfaceMockRecorder is the mock recorder for MockInterface.
type MockInterfaceMockRecorder struct {
	mock *MockInterface
}

// NewMockInterface creates a new mock instance.
func NewMockInterface(ctrl *gomock.Controller) *MockInterface {
	mock := &MockInterface{ctrl: ctrl}
	mock.recorder = &MockInterfaceMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockInterface) EXPECT() *MockInterfaceMockRecorder {
	return m.recorder
}

// APIReader mocks base method.
func (m *MockInterface) APIReader() client.Reader {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "APIReader")
	ret0, _ := ret[0].(client.Reader)
	return ret0
}

// APIReader indicates an expected call of APIReader.
func (mr *MockInterfaceMockRecorder) APIReader() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "APIReader", reflect.TypeOf((*MockInterface)(nil).APIReader))
}

// Applier mocks base method.
func (m *MockInterface) Applier() kubernetes.Applier {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Applier")
	ret0, _ := ret[0].(kubernetes.Applier)
	return ret0
}

// Applier indicates an expected call of Applier.
func (mr *MockInterfaceMockRecorder) Applier() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Applier", reflect.TypeOf((*MockInterface)(nil).Applier))
}

// Cache mocks base method.
func (m *MockInterface) Cache() cache.Cache {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Cache")
	ret0, _ := ret[0].(cache.Cache)
	return ret0
}

// Cache indicates an expected call of Cache.
func (mr *MockInterfaceMockRecorder) Cache() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Cache", reflect.TypeOf((*MockInterface)(nil).Cache))
}

// ChartApplier mocks base method.
func (m *MockInterface) ChartApplier() kubernetes.ChartApplier {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ChartApplier")
	ret0, _ := ret[0].(kubernetes.ChartApplier)
	return ret0
}

// ChartApplier indicates an expected call of ChartApplier.
func (mr *MockInterfaceMockRecorder) ChartApplier() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ChartApplier", reflect.TypeOf((*MockInterface)(nil).ChartApplier))
}

// ChartRenderer mocks base method.
func (m *MockInterface) ChartRenderer() chartrenderer.Interface {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ChartRenderer")
	ret0, _ := ret[0].(chartrenderer.Interface)
	return ret0
}

// ChartRenderer indicates an expected call of ChartRenderer.
func (mr *MockInterfaceMockRecorder) ChartRenderer() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ChartRenderer", reflect.TypeOf((*MockInterface)(nil).ChartRenderer))
}

// Client mocks base method.
func (m *MockInterface) Client() client.Client {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Client")
	ret0, _ := ret[0].(client.Client)
	return ret0
}

// Client indicates an expected call of Client.
func (mr *MockInterfaceMockRecorder) Client() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Client", reflect.TypeOf((*MockInterface)(nil).Client))
}

// DiscoverVersion mocks base method.
func (m *MockInterface) DiscoverVersion() (*version.Info, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DiscoverVersion")
	ret0, _ := ret[0].(*version.Info)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// DiscoverVersion indicates an expected call of DiscoverVersion.
func (mr *MockInterfaceMockRecorder) DiscoverVersion() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DiscoverVersion", reflect.TypeOf((*MockInterface)(nil).DiscoverVersion))
}

// Kubernetes mocks base method.
func (m *MockInterface) Kubernetes() kubernetes0.Interface {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Kubernetes")
	ret0, _ := ret[0].(kubernetes0.Interface)
	return ret0
}

// Kubernetes indicates an expected call of Kubernetes.
func (mr *MockInterfaceMockRecorder) Kubernetes() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Kubernetes", reflect.TypeOf((*MockInterface)(nil).Kubernetes))
}

// RESTClient mocks base method.
func (m *MockInterface) RESTClient() rest.Interface {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "RESTClient")
	ret0, _ := ret[0].(rest.Interface)
	return ret0
}

// RESTClient indicates an expected call of RESTClient.
func (mr *MockInterfaceMockRecorder) RESTClient() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "RESTClient", reflect.TypeOf((*MockInterface)(nil).RESTClient))
}

// RESTConfig mocks base method.
func (m *MockInterface) RESTConfig() *rest.Config {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "RESTConfig")
	ret0, _ := ret[0].(*rest.Config)
	return ret0
}

// RESTConfig indicates an expected call of RESTConfig.
func (mr *MockInterfaceMockRecorder) RESTConfig() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "RESTConfig", reflect.TypeOf((*MockInterface)(nil).RESTConfig))
}

// Start mocks base method.
func (m *MockInterface) Start(arg0 context.Context) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "Start", arg0)
}

// Start indicates an expected call of Start.
func (mr *MockInterfaceMockRecorder) Start(arg0 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Start", reflect.TypeOf((*MockInterface)(nil).Start), arg0)
}

// Version mocks base method.
func (m *MockInterface) Version() string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Version")
	ret0, _ := ret[0].(string)
	return ret0
}

// Version indicates an expected call of Version.
func (mr *MockInterfaceMockRecorder) Version() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Version", reflect.TypeOf((*MockInterface)(nil).Version))
}

// WaitForCacheSync mocks base method.
func (m *MockInterface) WaitForCacheSync(arg0 context.Context) bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "WaitForCacheSync", arg0)
	ret0, _ := ret[0].(bool)
	return ret0
}

// WaitForCacheSync indicates an expected call of WaitForCacheSync.
func (mr *MockInterfaceMockRecorder) WaitForCacheSync(arg0 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "WaitForCacheSync", reflect.TypeOf((*MockInterface)(nil).WaitForCacheSync), arg0)
}
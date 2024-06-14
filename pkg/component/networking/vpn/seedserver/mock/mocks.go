// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/gardener/gardener/pkg/component/networking/vpn/seedserver (interfaces: Interface)
//
// Generated by this command:
//
//	mockgen -package mock -destination=mocks.go github.com/gardener/gardener/pkg/component/networking/vpn/seedserver Interface
//

// Package mock is a generated GoMock package.
package mock

import (
	context "context"
	net "net"
	reflect "reflect"

	seedserver "github.com/gardener/gardener/pkg/component/networking/vpn/seedserver"
	gomock "go.uber.org/mock/gomock"
	types "k8s.io/apimachinery/pkg/types"
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

// Deploy mocks base method.
func (m *MockInterface) Deploy(arg0 context.Context) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Deploy", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// Deploy indicates an expected call of Deploy.
func (mr *MockInterfaceMockRecorder) Deploy(arg0 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Deploy", reflect.TypeOf((*MockInterface)(nil).Deploy), arg0)
}

// Destroy mocks base method.
func (m *MockInterface) Destroy(arg0 context.Context) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Destroy", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// Destroy indicates an expected call of Destroy.
func (mr *MockInterfaceMockRecorder) Destroy(arg0 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Destroy", reflect.TypeOf((*MockInterface)(nil).Destroy), arg0)
}

// GetValues mocks base method.
func (m *MockInterface) GetValues() seedserver.Values {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetValues")
	ret0, _ := ret[0].(seedserver.Values)
	return ret0
}

// GetValues indicates an expected call of GetValues.
func (mr *MockInterfaceMockRecorder) GetValues() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetValues", reflect.TypeOf((*MockInterface)(nil).GetValues))
}

// SetNodeNetworkCIDRs mocks base method.
func (m *MockInterface) SetNodeNetworkCIDRs(arg0 []net.IPNet) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "SetNodeNetworkCIDRs", arg0)
}

// SetNodeNetworkCIDRs indicates an expected call of SetNodeNetworkCIDRs.
func (mr *MockInterfaceMockRecorder) SetNodeNetworkCIDRs(arg0 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SetNodeNetworkCIDRs", reflect.TypeOf((*MockInterface)(nil).SetNodeNetworkCIDRs), arg0)
}

// SetPodNetworkCIDRs mocks base method.
func (m *MockInterface) SetPodNetworkCIDRs(arg0 []net.IPNet) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "SetPodNetworkCIDRs", arg0)
}

// SetPodNetworkCIDRs indicates an expected call of SetPodNetworkCIDRs.
func (mr *MockInterfaceMockRecorder) SetPodNetworkCIDRs(arg0 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SetPodNetworkCIDRs", reflect.TypeOf((*MockInterface)(nil).SetPodNetworkCIDRs), arg0)
}

// SetSeedNamespaceObjectUID mocks base method.
func (m *MockInterface) SetSeedNamespaceObjectUID(arg0 types.UID) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "SetSeedNamespaceObjectUID", arg0)
}

// SetSeedNamespaceObjectUID indicates an expected call of SetSeedNamespaceObjectUID.
func (mr *MockInterfaceMockRecorder) SetSeedNamespaceObjectUID(arg0 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SetSeedNamespaceObjectUID", reflect.TypeOf((*MockInterface)(nil).SetSeedNamespaceObjectUID), arg0)
}

// SetServiceNetworkCIDRs mocks base method.
func (m *MockInterface) SetServiceNetworkCIDRs(arg0 []net.IPNet) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "SetServiceNetworkCIDRs", arg0)
}

// SetServiceNetworkCIDRs indicates an expected call of SetServiceNetworkCIDRs.
func (mr *MockInterfaceMockRecorder) SetServiceNetworkCIDRs(arg0 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SetServiceNetworkCIDRs", reflect.TypeOf((*MockInterface)(nil).SetServiceNetworkCIDRs), arg0)
}

// Wait mocks base method.
func (m *MockInterface) Wait(arg0 context.Context) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Wait", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// Wait indicates an expected call of Wait.
func (mr *MockInterfaceMockRecorder) Wait(arg0 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Wait", reflect.TypeOf((*MockInterface)(nil).Wait), arg0)
}

// WaitCleanup mocks base method.
func (m *MockInterface) WaitCleanup(arg0 context.Context) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "WaitCleanup", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// WaitCleanup indicates an expected call of WaitCleanup.
func (mr *MockInterfaceMockRecorder) WaitCleanup(arg0 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "WaitCleanup", reflect.TypeOf((*MockInterface)(nil).WaitCleanup), arg0)
}

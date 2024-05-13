// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/gardener/gardener/pkg/client/core/clientset/versioned (interfaces: Interface)
//
// Generated by this command:
//
//	mockgen -destination=mocks.go -package=mock github.com/gardener/gardener/pkg/client/core/clientset/versioned Interface
//

// Package mock is a generated GoMock package.
package mock

import (
	reflect "reflect"

	v1 "github.com/gardener/gardener/pkg/client/core/clientset/versioned/typed/core/v1"
	v1beta1 "github.com/gardener/gardener/pkg/client/core/clientset/versioned/typed/core/v1beta1"
	gomock "go.uber.org/mock/gomock"
	discovery "k8s.io/client-go/discovery"
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

// CoreV1 mocks base method.
func (m *MockInterface) CoreV1() v1.CoreV1Interface {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CoreV1")
	ret0, _ := ret[0].(v1.CoreV1Interface)
	return ret0
}

// CoreV1 indicates an expected call of CoreV1.
func (mr *MockInterfaceMockRecorder) CoreV1() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CoreV1", reflect.TypeOf((*MockInterface)(nil).CoreV1))
}

// CoreV1beta1 mocks base method.
func (m *MockInterface) CoreV1beta1() v1beta1.CoreV1beta1Interface {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CoreV1beta1")
	ret0, _ := ret[0].(v1beta1.CoreV1beta1Interface)
	return ret0
}

// CoreV1beta1 indicates an expected call of CoreV1beta1.
func (mr *MockInterfaceMockRecorder) CoreV1beta1() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CoreV1beta1", reflect.TypeOf((*MockInterface)(nil).CoreV1beta1))
}

// Discovery mocks base method.
func (m *MockInterface) Discovery() discovery.DiscoveryInterface {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Discovery")
	ret0, _ := ret[0].(discovery.DiscoveryInterface)
	return ret0
}

// Discovery indicates an expected call of Discovery.
func (mr *MockInterfaceMockRecorder) Discovery() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Discovery", reflect.TypeOf((*MockInterface)(nil).Discovery))
}

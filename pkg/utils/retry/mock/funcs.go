// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/gardener/gardener/pkg/utils/retry/mock (interfaces: WaitFunc,Func)

// Package mock is a generated GoMock package.
package mock

import (
	context "context"
	reflect "reflect"

	gomock "go.uber.org/mock/gomock"
)

// MockWaitFunc is a mock of WaitFunc interface.
type MockWaitFunc struct {
	ctrl     *gomock.Controller
	recorder *MockWaitFuncMockRecorder
}

// MockWaitFuncMockRecorder is the mock recorder for MockWaitFunc.
type MockWaitFuncMockRecorder struct {
	mock *MockWaitFunc
}

// NewMockWaitFunc creates a new mock instance.
func NewMockWaitFunc(ctrl *gomock.Controller) *MockWaitFunc {
	mock := &MockWaitFunc{ctrl: ctrl}
	mock.recorder = &MockWaitFuncMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockWaitFunc) EXPECT() *MockWaitFuncMockRecorder {
	return m.recorder
}

// Do mocks base method.
func (m *MockWaitFunc) Do(arg0 context.Context) (context.Context, context.CancelFunc) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Do", arg0)
	ret0, _ := ret[0].(context.Context)
	ret1, _ := ret[1].(context.CancelFunc)
	return ret0, ret1
}

// Do indicates an expected call of Do.
func (mr *MockWaitFuncMockRecorder) Do(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Do", reflect.TypeOf((*MockWaitFunc)(nil).Do), arg0)
}

// MockFunc is a mock of Func interface.
type MockFunc struct {
	ctrl     *gomock.Controller
	recorder *MockFuncMockRecorder
}

// MockFuncMockRecorder is the mock recorder for MockFunc.
type MockFuncMockRecorder struct {
	mock *MockFunc
}

// NewMockFunc creates a new mock instance.
func NewMockFunc(ctrl *gomock.Controller) *MockFunc {
	mock := &MockFunc{ctrl: ctrl}
	mock.recorder = &MockFuncMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockFunc) EXPECT() *MockFuncMockRecorder {
	return m.recorder
}

// Do mocks base method.
func (m *MockFunc) Do(arg0 context.Context) (bool, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Do", arg0)
	ret0, _ := ret[0].(bool)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Do indicates an expected call of Do.
func (mr *MockFuncMockRecorder) Do(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Do", reflect.TypeOf((*MockFunc)(nil).Do), arg0)
}

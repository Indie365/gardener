// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/gardener/gardener/extensions/pkg/webhook (interfaces: Mutator)

// Package mock is a generated GoMock package.
package mock

import (
	context "context"
	reflect "reflect"

	gomock "go.uber.org/mock/gomock"
	client "sigs.k8s.io/controller-runtime/pkg/client"
)

// MockMutator is a mock of Mutator interface.
type MockMutator struct {
	ctrl     *gomock.Controller
	recorder *MockMutatorMockRecorder
}

// MockMutatorMockRecorder is the mock recorder for MockMutator.
type MockMutatorMockRecorder struct {
	mock *MockMutator
}

// NewMockMutator creates a new mock instance.
func NewMockMutator(ctrl *gomock.Controller) *MockMutator {
	mock := &MockMutator{ctrl: ctrl}
	mock.recorder = &MockMutatorMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockMutator) EXPECT() *MockMutatorMockRecorder {
	return m.recorder
}

// Mutate mocks base method.
func (m *MockMutator) Mutate(arg0 context.Context, arg1, arg2 client.Object) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Mutate", arg0, arg1, arg2)
	ret0, _ := ret[0].(error)
	return ret0
}

// Mutate indicates an expected call of Mutate.
func (mr *MockMutatorMockRecorder) Mutate(arg0, arg1, arg2 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Mutate", reflect.TypeOf((*MockMutator)(nil).Mutate), arg0, arg1, arg2)
}
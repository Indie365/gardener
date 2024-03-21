// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/gardener/gardener/pkg/utils/retry (interfaces: ErrorAggregator,ErrorAggregatorFactory,IntervalFactory)
//
// Generated by this command:
//
//	mockgen -destination=mocks.go -package=mock github.com/gardener/gardener/pkg/utils/retry ErrorAggregator,ErrorAggregatorFactory,IntervalFactory
//

// Package mock is a generated GoMock package.
package mock

import (
	reflect "reflect"
	time "time"

	retry "github.com/gardener/gardener/pkg/utils/retry"
	gomock "go.uber.org/mock/gomock"
)

// MockErrorAggregator is a mock of ErrorAggregator interface.
type MockErrorAggregator struct {
	ctrl     *gomock.Controller
	recorder *MockErrorAggregatorMockRecorder
}

// MockErrorAggregatorMockRecorder is the mock recorder for MockErrorAggregator.
type MockErrorAggregatorMockRecorder struct {
	mock *MockErrorAggregator
}

// NewMockErrorAggregator creates a new mock instance.
func NewMockErrorAggregator(ctrl *gomock.Controller) *MockErrorAggregator {
	mock := &MockErrorAggregator{ctrl: ctrl}
	mock.recorder = &MockErrorAggregatorMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockErrorAggregator) EXPECT() *MockErrorAggregatorMockRecorder {
	return m.recorder
}

// Error mocks base method.
func (m *MockErrorAggregator) Error() error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Error")
	ret0, _ := ret[0].(error)
	return ret0
}

// Error indicates an expected call of Error.
func (mr *MockErrorAggregatorMockRecorder) Error() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Error", reflect.TypeOf((*MockErrorAggregator)(nil).Error))
}

// Minor mocks base method.
func (m *MockErrorAggregator) Minor(arg0 error) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "Minor", arg0)
}

// Minor indicates an expected call of Minor.
func (mr *MockErrorAggregatorMockRecorder) Minor(arg0 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Minor", reflect.TypeOf((*MockErrorAggregator)(nil).Minor), arg0)
}

// Severe mocks base method.
func (m *MockErrorAggregator) Severe(arg0 error) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "Severe", arg0)
}

// Severe indicates an expected call of Severe.
func (mr *MockErrorAggregatorMockRecorder) Severe(arg0 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Severe", reflect.TypeOf((*MockErrorAggregator)(nil).Severe), arg0)
}

// MockErrorAggregatorFactory is a mock of ErrorAggregatorFactory interface.
type MockErrorAggregatorFactory struct {
	ctrl     *gomock.Controller
	recorder *MockErrorAggregatorFactoryMockRecorder
}

// MockErrorAggregatorFactoryMockRecorder is the mock recorder for MockErrorAggregatorFactory.
type MockErrorAggregatorFactoryMockRecorder struct {
	mock *MockErrorAggregatorFactory
}

// NewMockErrorAggregatorFactory creates a new mock instance.
func NewMockErrorAggregatorFactory(ctrl *gomock.Controller) *MockErrorAggregatorFactory {
	mock := &MockErrorAggregatorFactory{ctrl: ctrl}
	mock.recorder = &MockErrorAggregatorFactoryMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockErrorAggregatorFactory) EXPECT() *MockErrorAggregatorFactoryMockRecorder {
	return m.recorder
}

// New mocks base method.
func (m *MockErrorAggregatorFactory) New() retry.ErrorAggregator {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "New")
	ret0, _ := ret[0].(retry.ErrorAggregator)
	return ret0
}

// New indicates an expected call of New.
func (mr *MockErrorAggregatorFactoryMockRecorder) New() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "New", reflect.TypeOf((*MockErrorAggregatorFactory)(nil).New))
}

// MockIntervalFactory is a mock of IntervalFactory interface.
type MockIntervalFactory struct {
	ctrl     *gomock.Controller
	recorder *MockIntervalFactoryMockRecorder
}

// MockIntervalFactoryMockRecorder is the mock recorder for MockIntervalFactory.
type MockIntervalFactoryMockRecorder struct {
	mock *MockIntervalFactory
}

// NewMockIntervalFactory creates a new mock instance.
func NewMockIntervalFactory(ctrl *gomock.Controller) *MockIntervalFactory {
	mock := &MockIntervalFactory{ctrl: ctrl}
	mock.recorder = &MockIntervalFactoryMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockIntervalFactory) EXPECT() *MockIntervalFactoryMockRecorder {
	return m.recorder
}

// New mocks base method.
func (m *MockIntervalFactory) New(arg0 time.Duration) retry.WaitFunc {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "New", arg0)
	ret0, _ := ret[0].(retry.WaitFunc)
	return ret0
}

// New indicates an expected call of New.
func (mr *MockIntervalFactoryMockRecorder) New(arg0 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "New", reflect.TypeOf((*MockIntervalFactory)(nil).New), arg0)
}
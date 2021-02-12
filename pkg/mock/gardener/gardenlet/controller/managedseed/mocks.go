// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/gardener/gardener/pkg/gardenlet/controller/managedseed (interfaces: Actuator,ValuesHelper)

// Package managedseed is a generated GoMock package.
package managedseed

import (
	context "context"
	reflect "reflect"

	v1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	v1alpha1 "github.com/gardener/gardener/pkg/apis/seedmanagement/v1alpha1"
	v1alpha10 "github.com/gardener/gardener/pkg/gardenlet/apis/config/v1alpha1"
	gomock "github.com/golang/mock/gomock"
)

// MockActuator is a mock of Actuator interface.
type MockActuator struct {
	ctrl     *gomock.Controller
	recorder *MockActuatorMockRecorder
}

// MockActuatorMockRecorder is the mock recorder for MockActuator.
type MockActuatorMockRecorder struct {
	mock *MockActuator
}

// NewMockActuator creates a new mock instance.
func NewMockActuator(ctrl *gomock.Controller) *MockActuator {
	mock := &MockActuator{ctrl: ctrl}
	mock.recorder = &MockActuatorMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockActuator) EXPECT() *MockActuatorMockRecorder {
	return m.recorder
}

// Delete mocks base method.
func (m *MockActuator) Delete(arg0 context.Context, arg1 *v1alpha1.ManagedSeed, arg2 *v1beta1.Shoot) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Delete", arg0, arg1, arg2)
	ret0, _ := ret[0].(error)
	return ret0
}

// Delete indicates an expected call of Delete.
func (mr *MockActuatorMockRecorder) Delete(arg0, arg1, arg2 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Delete", reflect.TypeOf((*MockActuator)(nil).Delete), arg0, arg1, arg2)
}

// Reconcile mocks base method.
func (m *MockActuator) Reconcile(arg0 context.Context, arg1 *v1alpha1.ManagedSeed, arg2 *v1beta1.Shoot) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Reconcile", arg0, arg1, arg2)
	ret0, _ := ret[0].(error)
	return ret0
}

// Reconcile indicates an expected call of Reconcile.
func (mr *MockActuatorMockRecorder) Reconcile(arg0, arg1, arg2 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Reconcile", reflect.TypeOf((*MockActuator)(nil).Reconcile), arg0, arg1, arg2)
}

// MockValuesHelper is a mock of ValuesHelper interface.
type MockValuesHelper struct {
	ctrl     *gomock.Controller
	recorder *MockValuesHelperMockRecorder
}

// MockValuesHelperMockRecorder is the mock recorder for MockValuesHelper.
type MockValuesHelperMockRecorder struct {
	mock *MockValuesHelper
}

// NewMockValuesHelper creates a new mock instance.
func NewMockValuesHelper(ctrl *gomock.Controller) *MockValuesHelper {
	mock := &MockValuesHelper{ctrl: ctrl}
	mock.recorder = &MockValuesHelperMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockValuesHelper) EXPECT() *MockValuesHelperMockRecorder {
	return m.recorder
}

// GetGardenletChartValues mocks base method.
func (m *MockValuesHelper) GetGardenletChartValues(arg0 *v1alpha1.GardenletDeployment, arg1 *v1alpha10.GardenletConfiguration, arg2 string) (map[string]interface{}, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetGardenletChartValues", arg0, arg1, arg2)
	ret0, _ := ret[0].(map[string]interface{})
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetGardenletChartValues indicates an expected call of GetGardenletChartValues.
func (mr *MockValuesHelperMockRecorder) GetGardenletChartValues(arg0, arg1, arg2 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetGardenletChartValues", reflect.TypeOf((*MockValuesHelper)(nil).GetGardenletChartValues), arg0, arg1, arg2)
}

// MergeGardenletConfiguration mocks base method.
func (m *MockValuesHelper) MergeGardenletConfiguration(arg0 *v1alpha10.GardenletConfiguration) (*v1alpha10.GardenletConfiguration, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "MergeGardenletConfiguration", arg0)
	ret0, _ := ret[0].(*v1alpha10.GardenletConfiguration)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// MergeGardenletConfiguration indicates an expected call of MergeGardenletConfiguration.
func (mr *MockValuesHelperMockRecorder) MergeGardenletConfiguration(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "MergeGardenletConfiguration", reflect.TypeOf((*MockValuesHelper)(nil).MergeGardenletConfiguration), arg0)
}

// MergeGardenletDeployment mocks base method.
func (m *MockValuesHelper) MergeGardenletDeployment(arg0 *v1alpha1.GardenletDeployment, arg1 *v1beta1.Shoot) (*v1alpha1.GardenletDeployment, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "MergeGardenletDeployment", arg0, arg1)
	ret0, _ := ret[0].(*v1alpha1.GardenletDeployment)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// MergeGardenletDeployment indicates an expected call of MergeGardenletDeployment.
func (mr *MockValuesHelperMockRecorder) MergeGardenletDeployment(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "MergeGardenletDeployment", reflect.TypeOf((*MockValuesHelper)(nil).MergeGardenletDeployment), arg0, arg1)
}

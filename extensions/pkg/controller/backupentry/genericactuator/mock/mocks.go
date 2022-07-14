// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/gardener/gardener/extensions/pkg/controller/backupentry/genericactuator (interfaces: BackupEntryDelegate)

// Package mock is a generated GoMock package.
package mock

import (
	context "context"
	reflect "reflect"

	v1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	logr "github.com/go-logr/logr"
	gomock "github.com/golang/mock/gomock"
)

// MockBackupEntryDelegate is a mock of BackupEntryDelegate interface.
type MockBackupEntryDelegate struct {
	ctrl     *gomock.Controller
	recorder *MockBackupEntryDelegateMockRecorder
}

// MockBackupEntryDelegateMockRecorder is the mock recorder for MockBackupEntryDelegate.
type MockBackupEntryDelegateMockRecorder struct {
	mock *MockBackupEntryDelegate
}

// NewMockBackupEntryDelegate creates a new mock instance.
func NewMockBackupEntryDelegate(ctrl *gomock.Controller) *MockBackupEntryDelegate {
	mock := &MockBackupEntryDelegate{ctrl: ctrl}
	mock.recorder = &MockBackupEntryDelegateMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockBackupEntryDelegate) EXPECT() *MockBackupEntryDelegateMockRecorder {
	return m.recorder
}

// Delete mocks base method.
func (m *MockBackupEntryDelegate) Delete(arg0 context.Context, arg1 logr.Logger, arg2 *v1alpha1.BackupEntry) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Delete", arg0, arg1, arg2)
	ret0, _ := ret[0].(error)
	return ret0
}

// Delete indicates an expected call of Delete.
func (mr *MockBackupEntryDelegateMockRecorder) Delete(arg0, arg1, arg2 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Delete", reflect.TypeOf((*MockBackupEntryDelegate)(nil).Delete), arg0, arg1, arg2)
}

// GetETCDSecretData mocks base method.
func (m *MockBackupEntryDelegate) GetETCDSecretData(arg0 context.Context, arg1 logr.Logger, arg2 *v1alpha1.BackupEntry, arg3 map[string][]byte) (map[string][]byte, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetETCDSecretData", arg0, arg1, arg2, arg3)
	ret0, _ := ret[0].(map[string][]byte)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetETCDSecretData indicates an expected call of GetETCDSecretData.
func (mr *MockBackupEntryDelegateMockRecorder) GetETCDSecretData(arg0, arg1, arg2, arg3 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetETCDSecretData", reflect.TypeOf((*MockBackupEntryDelegate)(nil).GetETCDSecretData), arg0, arg1, arg2, arg3)
}

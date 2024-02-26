// Code generated by MockGen. DO NOT EDIT.
// Source: sigs.k8s.io/controller-runtime/pkg/cache (interfaces: Cache)
//
// Generated by this command:
//
//	mockgen -package cache -destination=mocks.go sigs.k8s.io/controller-runtime/pkg/cache Cache
//

// Package cache is a generated GoMock package.
package cache

import (
	context "context"
	reflect "reflect"

	gomock "go.uber.org/mock/gomock"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	cache "sigs.k8s.io/controller-runtime/pkg/cache"
	client "sigs.k8s.io/controller-runtime/pkg/client"
)

// MockCache is a mock of Cache interface.
type MockCache struct {
	ctrl     *gomock.Controller
	recorder *MockCacheMockRecorder
}

// MockCacheMockRecorder is the mock recorder for MockCache.
type MockCacheMockRecorder struct {
	mock *MockCache
}

// NewMockCache creates a new mock instance.
func NewMockCache(ctrl *gomock.Controller) *MockCache {
	mock := &MockCache{ctrl: ctrl}
	mock.recorder = &MockCacheMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockCache) EXPECT() *MockCacheMockRecorder {
	return m.recorder
}

// Get mocks base method.
func (m *MockCache) Get(arg0 context.Context, arg1 types.NamespacedName, arg2 client.Object, arg3 ...client.GetOption) error {
	m.ctrl.T.Helper()
	varargs := []any{arg0, arg1, arg2}
	for _, a := range arg3 {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "Get", varargs...)
	ret0, _ := ret[0].(error)
	return ret0
}

// Get indicates an expected call of Get.
func (mr *MockCacheMockRecorder) Get(arg0, arg1, arg2 any, arg3 ...any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]any{arg0, arg1, arg2}, arg3...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Get", reflect.TypeOf((*MockCache)(nil).Get), varargs...)
}

// GetInformer mocks base method.
func (m *MockCache) GetInformer(arg0 context.Context, arg1 client.Object, arg2 ...cache.InformerGetOption) (cache.Informer, error) {
	m.ctrl.T.Helper()
	varargs := []any{arg0, arg1}
	for _, a := range arg2 {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "GetInformer", varargs...)
	ret0, _ := ret[0].(cache.Informer)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetInformer indicates an expected call of GetInformer.
func (mr *MockCacheMockRecorder) GetInformer(arg0, arg1 any, arg2 ...any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]any{arg0, arg1}, arg2...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetInformer", reflect.TypeOf((*MockCache)(nil).GetInformer), varargs...)
}

// GetInformerForKind mocks base method.
func (m *MockCache) GetInformerForKind(arg0 context.Context, arg1 schema.GroupVersionKind, arg2 ...cache.InformerGetOption) (cache.Informer, error) {
	m.ctrl.T.Helper()
	varargs := []any{arg0, arg1}
	for _, a := range arg2 {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "GetInformerForKind", varargs...)
	ret0, _ := ret[0].(cache.Informer)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetInformerForKind indicates an expected call of GetInformerForKind.
func (mr *MockCacheMockRecorder) GetInformerForKind(arg0, arg1 any, arg2 ...any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]any{arg0, arg1}, arg2...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetInformerForKind", reflect.TypeOf((*MockCache)(nil).GetInformerForKind), varargs...)
}

// IndexField mocks base method.
func (m *MockCache) IndexField(arg0 context.Context, arg1 client.Object, arg2 string, arg3 client.IndexerFunc) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "IndexField", arg0, arg1, arg2, arg3)
	ret0, _ := ret[0].(error)
	return ret0
}

// IndexField indicates an expected call of IndexField.
func (mr *MockCacheMockRecorder) IndexField(arg0, arg1, arg2, arg3 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "IndexField", reflect.TypeOf((*MockCache)(nil).IndexField), arg0, arg1, arg2, arg3)
}

// List mocks base method.
func (m *MockCache) List(arg0 context.Context, arg1 client.ObjectList, arg2 ...client.ListOption) error {
	m.ctrl.T.Helper()
	varargs := []any{arg0, arg1}
	for _, a := range arg2 {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "List", varargs...)
	ret0, _ := ret[0].(error)
	return ret0
}

// List indicates an expected call of List.
func (mr *MockCacheMockRecorder) List(arg0, arg1 any, arg2 ...any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]any{arg0, arg1}, arg2...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "List", reflect.TypeOf((*MockCache)(nil).List), varargs...)
}

// Start mocks base method.
func (m *MockCache) Start(arg0 context.Context) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Start", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// Start indicates an expected call of Start.
func (mr *MockCacheMockRecorder) Start(arg0 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Start", reflect.TypeOf((*MockCache)(nil).Start), arg0)
}

// WaitForCacheSync mocks base method.
func (m *MockCache) WaitForCacheSync(arg0 context.Context) bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "WaitForCacheSync", arg0)
	ret0, _ := ret[0].(bool)
	return ret0
}

// WaitForCacheSync indicates an expected call of WaitForCacheSync.
func (mr *MockCacheMockRecorder) WaitForCacheSync(arg0 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "WaitForCacheSync", reflect.TypeOf((*MockCache)(nil).WaitForCacheSync), arg0)
}

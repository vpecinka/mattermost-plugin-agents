// Code generated by mockery; DO NOT EDIT.
// github.com/vektra/mockery
// template: testify

package mocks

import (
	"context"

	"github.com/mattermost/mattermost-plugin-ai/embeddings"
	mock "github.com/stretchr/testify/mock"
)

// NewMockEmbeddingSearch creates a new instance of MockEmbeddingSearch. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewMockEmbeddingSearch(t interface {
	mock.TestingT
	Cleanup(func())
}) *MockEmbeddingSearch {
	mock := &MockEmbeddingSearch{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}

// MockEmbeddingSearch is an autogenerated mock type for the EmbeddingSearch type
type MockEmbeddingSearch struct {
	mock.Mock
}

type MockEmbeddingSearch_Expecter struct {
	mock *mock.Mock
}

func (_m *MockEmbeddingSearch) EXPECT() *MockEmbeddingSearch_Expecter {
	return &MockEmbeddingSearch_Expecter{mock: &_m.Mock}
}

// Clear provides a mock function for the type MockEmbeddingSearch
func (_mock *MockEmbeddingSearch) Clear(ctx context.Context) error {
	ret := _mock.Called(ctx)

	if len(ret) == 0 {
		panic("no return value specified for Clear")
	}

	var r0 error
	if returnFunc, ok := ret.Get(0).(func(context.Context) error); ok {
		r0 = returnFunc(ctx)
	} else {
		r0 = ret.Error(0)
	}
	return r0
}

// MockEmbeddingSearch_Clear_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Clear'
type MockEmbeddingSearch_Clear_Call struct {
	*mock.Call
}

// Clear is a helper method to define mock.On call
//   - ctx
func (_e *MockEmbeddingSearch_Expecter) Clear(ctx interface{}) *MockEmbeddingSearch_Clear_Call {
	return &MockEmbeddingSearch_Clear_Call{Call: _e.mock.On("Clear", ctx)}
}

func (_c *MockEmbeddingSearch_Clear_Call) Run(run func(ctx context.Context)) *MockEmbeddingSearch_Clear_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context))
	})
	return _c
}

func (_c *MockEmbeddingSearch_Clear_Call) Return(err error) *MockEmbeddingSearch_Clear_Call {
	_c.Call.Return(err)
	return _c
}

func (_c *MockEmbeddingSearch_Clear_Call) RunAndReturn(run func(ctx context.Context) error) *MockEmbeddingSearch_Clear_Call {
	_c.Call.Return(run)
	return _c
}

// Delete provides a mock function for the type MockEmbeddingSearch
func (_mock *MockEmbeddingSearch) Delete(ctx context.Context, postIDs []string) error {
	ret := _mock.Called(ctx, postIDs)

	if len(ret) == 0 {
		panic("no return value specified for Delete")
	}

	var r0 error
	if returnFunc, ok := ret.Get(0).(func(context.Context, []string) error); ok {
		r0 = returnFunc(ctx, postIDs)
	} else {
		r0 = ret.Error(0)
	}
	return r0
}

// MockEmbeddingSearch_Delete_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Delete'
type MockEmbeddingSearch_Delete_Call struct {
	*mock.Call
}

// Delete is a helper method to define mock.On call
//   - ctx
//   - postIDs
func (_e *MockEmbeddingSearch_Expecter) Delete(ctx interface{}, postIDs interface{}) *MockEmbeddingSearch_Delete_Call {
	return &MockEmbeddingSearch_Delete_Call{Call: _e.mock.On("Delete", ctx, postIDs)}
}

func (_c *MockEmbeddingSearch_Delete_Call) Run(run func(ctx context.Context, postIDs []string)) *MockEmbeddingSearch_Delete_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].([]string))
	})
	return _c
}

func (_c *MockEmbeddingSearch_Delete_Call) Return(err error) *MockEmbeddingSearch_Delete_Call {
	_c.Call.Return(err)
	return _c
}

func (_c *MockEmbeddingSearch_Delete_Call) RunAndReturn(run func(ctx context.Context, postIDs []string) error) *MockEmbeddingSearch_Delete_Call {
	_c.Call.Return(run)
	return _c
}

// Search provides a mock function for the type MockEmbeddingSearch
func (_mock *MockEmbeddingSearch) Search(ctx context.Context, query string, opts embeddings.SearchOptions) ([]embeddings.SearchResult, error) {
	ret := _mock.Called(ctx, query, opts)

	if len(ret) == 0 {
		panic("no return value specified for Search")
	}

	var r0 []embeddings.SearchResult
	var r1 error
	if returnFunc, ok := ret.Get(0).(func(context.Context, string, embeddings.SearchOptions) ([]embeddings.SearchResult, error)); ok {
		return returnFunc(ctx, query, opts)
	}
	if returnFunc, ok := ret.Get(0).(func(context.Context, string, embeddings.SearchOptions) []embeddings.SearchResult); ok {
		r0 = returnFunc(ctx, query, opts)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]embeddings.SearchResult)
		}
	}
	if returnFunc, ok := ret.Get(1).(func(context.Context, string, embeddings.SearchOptions) error); ok {
		r1 = returnFunc(ctx, query, opts)
	} else {
		r1 = ret.Error(1)
	}
	return r0, r1
}

// MockEmbeddingSearch_Search_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Search'
type MockEmbeddingSearch_Search_Call struct {
	*mock.Call
}

// Search is a helper method to define mock.On call
//   - ctx
//   - query
//   - opts
func (_e *MockEmbeddingSearch_Expecter) Search(ctx interface{}, query interface{}, opts interface{}) *MockEmbeddingSearch_Search_Call {
	return &MockEmbeddingSearch_Search_Call{Call: _e.mock.On("Search", ctx, query, opts)}
}

func (_c *MockEmbeddingSearch_Search_Call) Run(run func(ctx context.Context, query string, opts embeddings.SearchOptions)) *MockEmbeddingSearch_Search_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(string), args[2].(embeddings.SearchOptions))
	})
	return _c
}

func (_c *MockEmbeddingSearch_Search_Call) Return(searchResults []embeddings.SearchResult, err error) *MockEmbeddingSearch_Search_Call {
	_c.Call.Return(searchResults, err)
	return _c
}

func (_c *MockEmbeddingSearch_Search_Call) RunAndReturn(run func(ctx context.Context, query string, opts embeddings.SearchOptions) ([]embeddings.SearchResult, error)) *MockEmbeddingSearch_Search_Call {
	_c.Call.Return(run)
	return _c
}

// Store provides a mock function for the type MockEmbeddingSearch
func (_mock *MockEmbeddingSearch) Store(ctx context.Context, docs []embeddings.PostDocument) error {
	ret := _mock.Called(ctx, docs)

	if len(ret) == 0 {
		panic("no return value specified for Store")
	}

	var r0 error
	if returnFunc, ok := ret.Get(0).(func(context.Context, []embeddings.PostDocument) error); ok {
		r0 = returnFunc(ctx, docs)
	} else {
		r0 = ret.Error(0)
	}
	return r0
}

// MockEmbeddingSearch_Store_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Store'
type MockEmbeddingSearch_Store_Call struct {
	*mock.Call
}

// Store is a helper method to define mock.On call
//   - ctx
//   - docs
func (_e *MockEmbeddingSearch_Expecter) Store(ctx interface{}, docs interface{}) *MockEmbeddingSearch_Store_Call {
	return &MockEmbeddingSearch_Store_Call{Call: _e.mock.On("Store", ctx, docs)}
}

func (_c *MockEmbeddingSearch_Store_Call) Run(run func(ctx context.Context, docs []embeddings.PostDocument)) *MockEmbeddingSearch_Store_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].([]embeddings.PostDocument))
	})
	return _c
}

func (_c *MockEmbeddingSearch_Store_Call) Return(err error) *MockEmbeddingSearch_Store_Call {
	_c.Call.Return(err)
	return _c
}

func (_c *MockEmbeddingSearch_Store_Call) RunAndReturn(run func(ctx context.Context, docs []embeddings.PostDocument) error) *MockEmbeddingSearch_Store_Call {
	_c.Call.Return(run)
	return _c
}

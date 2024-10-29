// Code generated by mockery v2.46.3. DO NOT EDIT.

package mocks

import (
	context "context"

	event "github.com/davidsbond/autopgo/internal/event"
	mock "github.com/stretchr/testify/mock"
)

// MockEventWriter is an autogenerated mock type for the EventWriter type
type MockEventWriter struct {
	mock.Mock
}

type MockEventWriter_Expecter struct {
	mock *mock.Mock
}

func (_m *MockEventWriter) EXPECT() *MockEventWriter_Expecter {
	return &MockEventWriter_Expecter{mock: &_m.Mock}
}

// Write provides a mock function with given fields: ctx, evt
func (_m *MockEventWriter) Write(ctx context.Context, evt event.Payload) error {
	ret := _m.Called(ctx, evt)

	if len(ret) == 0 {
		panic("no return value specified for Write")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, event.Payload) error); ok {
		r0 = rf(ctx, evt)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// MockEventWriter_Write_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Write'
type MockEventWriter_Write_Call struct {
	*mock.Call
}

// Write is a helper method to define mock.On call
//   - ctx context.Context
//   - evt event.Payload
func (_e *MockEventWriter_Expecter) Write(ctx interface{}, evt interface{}) *MockEventWriter_Write_Call {
	return &MockEventWriter_Write_Call{Call: _e.mock.On("Write", ctx, evt)}
}

func (_c *MockEventWriter_Write_Call) Run(run func(ctx context.Context, evt event.Payload)) *MockEventWriter_Write_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(event.Payload))
	})
	return _c
}

func (_c *MockEventWriter_Write_Call) Return(_a0 error) *MockEventWriter_Write_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *MockEventWriter_Write_Call) RunAndReturn(run func(context.Context, event.Payload) error) *MockEventWriter_Write_Call {
	_c.Call.Return(run)
	return _c
}

// NewMockEventWriter creates a new instance of MockEventWriter. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewMockEventWriter(t interface {
	mock.TestingT
	Cleanup(func())
}) *MockEventWriter {
	mock := &MockEventWriter{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}

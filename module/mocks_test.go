package module_test

import (
	"io"

	"github.com/stretchr/testify/mock"
	"golang.org/x/mod/module"
)

type MockClient struct {
	mock.Mock
}

func (c *MockClient) Version(path, ver string) (module.Version, error) {
	args := c.Called(path, ver)
	return args.Get(0).(module.Version), args.Error(1)
}

func (c *MockClient) Download(m module.Version) (io.ReadCloser, error) {
	args := c.Called(m)
	return args.Get(0).(io.ReadCloser), args.Error(1)
}

type MockUI struct {
	mock.Mock
}

func (m *MockUI) LoadCSS(css string) error {
	args := m.Called(css)
	return args.Error(0)
}

func (m *MockUI) LoadHTML(html string) error {
	args := m.Called(html)
	return args.Error(0)
}

func (m *MockUI) Bind(name string, fun interface{}) error {
	args := m.Called(name, fun)
	return args.Error(0)
}

func (m *MockUI) Eval(cmd string, ctx ...interface{}) (interface{}, error) {
	params := append([]interface{}{cmd}, ctx...)
	args := m.Called(params...)
	return args.Get(0), args.Error(0)
}

type MockLogger struct {
	mock.Mock
}

func (m *MockLogger) Info(msg string, ctx ...interface{}) {
	params := append([]interface{}{msg}, ctx...)
	_ = m.Called(params)
}

func (m *MockLogger) Error(msg string, ctx ...interface{}) {
	params := append([]interface{}{msg}, ctx...)
	_ = m.Called(params)
}

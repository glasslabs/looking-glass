package glass

import (
	"encoding/json"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/hamba/logger/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/zserge/lorca"
)

func TestNewUI(t *testing.T) {
	log := logger.New(io.Discard, logger.LogfmtFormat(), logger.Error)

	cfg := UIConfig{
		Width:      1024,
		Height:     764,
		Fullscreen: true,
		CustomCSS: []string{
			"testdata/custom.css",
		},
	}
	ui := &MockLorcaUI{}
	ui.On("Eval", mock.MatchedBy(func(js string) bool {
		return strings.HasPrefix(js, "loadCSS(`fonts`")
	})).Once().Return(NewValue("", nil))
	ui.On("Eval", "loadCSS(`customCSS1`, `custom css`);").Once().Return(NewValue("", nil))
	ui.On("Bind", mock.AnythingOfType("string"), mock.Anything).Return(nil)

	oldNewFunc := newFunc
	t.Cleanup(func() {
		newFunc = oldNewFunc
	})
	newFunc = func(url, dir string, width, height int, customArgs ...string) (lorca.UI, error) {
		assert.Equal(t, 1024, width)
		assert.Equal(t, 764, height)
		assert.Equal(t, 1024, width)
		assert.Contains(t, customArgs, "--start-fullscreen")

		return ui, nil
	}

	got, err := NewUI(cfg, log)

	require.NoError(t, err)
	assert.IsType(t, (*UI)(nil), got)
	ui.AssertExpectations(t)
}

func TestNewUI_HandlesWindowError(t *testing.T) {
	log := logger.New(io.Discard, logger.LogfmtFormat(), logger.Error)

	cfg := UIConfig{
		Width:  1024,
		Height: 764,
	}

	oldNewFunc := newFunc
	t.Cleanup(func() {
		newFunc = oldNewFunc
	})
	newFunc = func(url, dir string, width, height int, customArgs ...string) (lorca.UI, error) {
		return nil, errors.New("test error")
	}

	_, err := NewUI(cfg, log)

	assert.Error(t, err)
	assert.EqualError(t, err, "could not create window: test error")
}

func TestUI_Done(t *testing.T) {
	ch := make(chan struct{})
	t.Cleanup(func() {
		close(ch)
	})
	win := &MockLorcaUI{}
	win.On("Done").Return(ch)
	ui := &UI{win: win}

	got := ui.Done()

	if ch != got {
		assert.Fail(t, "incorrect channel")
		return
	}
	win.AssertExpectations(t)
}

func TestUI_Close(t *testing.T) {
	win := &MockLorcaUI{}
	win.On("Close").Return(nil)
	ui := &UI{win: win}

	err := ui.Close()

	assert.NoError(t, err)
	win.AssertExpectations(t)
}

type MockLorcaUI struct {
	mock.Mock
}

func (m *MockLorcaUI) Load(url string) error {
	args := m.Called(url)
	return args.Error(0)
}

func (m *MockLorcaUI) Bounds() (lorca.Bounds, error) {
	args := m.Called()
	return args.Get(0).(lorca.Bounds), args.Error(1)
}

func (m *MockLorcaUI) SetBounds(bounds lorca.Bounds) error {
	args := m.Called(bounds)
	return args.Error(0)
}

func (m *MockLorcaUI) Bind(name string, f any) error {
	args := m.Called(name, f)
	return args.Error(0)
}

func (m *MockLorcaUI) Eval(js string) lorca.Value {
	args := m.Called(js)
	return args.Get(0).(lorca.Value)
}

func (m *MockLorcaUI) Done() <-chan struct{} {
	args := m.Called()
	return args.Get(0).(chan struct{})
}

func (m *MockLorcaUI) Close() error {
	args := m.Called()
	return args.Error(0)
}

type Value struct {
	err error
	raw json.RawMessage
}

func NewValue(val string, err error) Value {
	v := json.RawMessage{}
	if val != "" {
		v = json.RawMessage(val)
	}

	return Value{
		raw: v,
		err: err,
	}
}

func (v Value) Err() error         { return v.err }
func (v Value) To(x any) error     { return json.Unmarshal(v.raw, x) }
func (v Value) Float() (f float32) { v.To(&f); return f }
func (v Value) Int() (i int)       { v.To(&i); return i }
func (v Value) String() (s string) { v.To(&s); return s }
func (v Value) Bool() (b bool)     { v.To(&b); return b }
func (v Value) Bytes() []byte      { return v.raw }
func (v Value) Array() (values []lorca.Value) {
	array := []json.RawMessage{}
	_ = v.To(&array)
	for _, el := range array {
		values = append(values, Value{raw: el})
	}
	return values
}
func (v Value) Object() (object map[string]lorca.Value) {
	object = map[string]lorca.Value{}
	kv := map[string]json.RawMessage{}
	_ = v.To(&kv)
	for k, v := range kv {
		object[k] = Value{raw: v}
	}
	return object
}

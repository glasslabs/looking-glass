package module

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/glasslabs/client-go"
	"github.com/hamba/logger/v2"
	lctx "github.com/hamba/logger/v2/ctx"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
)

// streamStore manages open HTTP response streams keyed by a numeric handle.
// Handles are issued with a monotonically increasing counter so they are
// unique for the lifetime of the process.
type streamStore struct {
	counter atomic.Int32

	mu      sync.Mutex
	streams map[int32]*openStream
}

func newStreamStore() *streamStore {
	return &streamStore{streams: make(map[int32]*openStream)}
}

// openStream holds an in-flight HTTP response body.
type openStream struct {
	resp *http.Response
}

func (s *streamStore) add(stream *openStream) int32 {
	id := s.counter.Add(1)
	s.mu.Lock()
	s.streams[id] = stream
	s.mu.Unlock()
	return id
}

func (s *streamStore) get(id int32) (*openStream, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	sc, ok := s.streams[id]
	return sc, ok
}

func (s *streamStore) remove(id int32) *openStream {
	s.mu.Lock()
	sc := s.streams[id]
	delete(s.streams, id)
	s.mu.Unlock()
	return sc
}

// buildHostModule registers the looking-glass host functions into rt.
//
// render(ptr uint32, length uint32)
//
//	ptr and length describe a slice in the plugin's linear memory containing
//	"moduleName\x00{widgetXML}". The host splits on the null byte, looks up
//	the named module's WidgetPusher, and forwards the XML bytes to it.
//
// http_stream_open(method_ptr, method_len, url_ptr, url_len, hdr_ptr, hdr_len, body_ptr, body_len) -> handle
//
//	Opens an HTTP request and returns a handle for the response stream.
//	hdr_ptr points to newline-separated "Key: Value\n" header lines (hdr_len=0
//	for no headers). body_ptr/body_len describe a request body (0/0 for none).
//	Returns a non-negative handle on success, or a negative errno on failure.
//
// http_stream_status(handle) -> status_code
//
//	Returns the HTTP status code for the response identified by handle.
//
// http_stream_read(handle, buf_ptr, buf_len) -> n
//
//	Reads up to buf_len bytes from the response body into buf_ptr.
//	Returns the number of bytes read, 0 on EOF, or a negative errno on error.
//	Blocks until data is available, the stream ends, or the module context is
//	cancelled.
//
// http_stream_close(handle)
//
//	Closes the response body and releases the handle.
func buildHostModule(ctx context.Context, rt wazero.Runtime, ui UIProvider, log *logger.Logger) error {
	store := newStreamStore()

	_, err := rt.NewHostModuleBuilder("looking-glass").
		NewFunctionBuilder().
		WithGoModuleFunction(
			renderFunc(ui, log),
			[]api.ValueType{api.ValueTypeI32, api.ValueTypeI32},
			[]api.ValueType{},
		).
		Export("render").
		NewFunctionBuilder().
		WithGoModuleFunction(
			httpStreamOpenFunc(store, log),
			[]api.ValueType{
				api.ValueTypeI32, api.ValueTypeI32, // method
				api.ValueTypeI32, api.ValueTypeI32, // url
				api.ValueTypeI32, api.ValueTypeI32, // headers
				api.ValueTypeI32, api.ValueTypeI32, // body
			},
			[]api.ValueType{api.ValueTypeI32},
		).
		Export("http_stream_open").
		NewFunctionBuilder().
		WithGoModuleFunction(
			// http_stream_status: returns the HTTP status code for a handle.
			httpStreamStatusFunc(store),
			[]api.ValueType{api.ValueTypeI32},
			[]api.ValueType{api.ValueTypeI32},
		).
		Export("http_stream_status").
		NewFunctionBuilder().
		WithGoModuleFunction(
			httpStreamReadFunc(store),
			[]api.ValueType{api.ValueTypeI32, api.ValueTypeI32, api.ValueTypeI32},
			[]api.ValueType{api.ValueTypeI32},
		).
		Export("http_stream_read").
		NewFunctionBuilder().
		WithGoModuleFunction(
			httpStreamCloseFunc(store),
			[]api.ValueType{api.ValueTypeI32},
			[]api.ValueType{},
		).
		Export("http_stream_close").
		Instantiate(ctx)
	return err
}

func renderFunc(ui UIProvider, log *logger.Logger) api.GoModuleFunc {
	return func(_ context.Context, mod api.Module, stack []uint64) {
		ptr := uint32(stack[0])
		length := uint32(stack[1])

		data, ok := mod.Memory().Read(ptr, length)
		if !ok {
			log.Error("render: out-of-bounds memory read")
			return
		}

		before, after, ok := bytes.Cut(data, []byte{0})
		if !ok {
			log.Error("render: missing null separator in payload")
			return
		}

		name := string(before)
		widgetXML := after

		log.Trace("render: widget received", lctx.Str("module", name), lctx.Int("bytes", len(widgetXML)))

		w, err := client.DecodeWidget(widgetXML)
		if err != nil {
			log.Error("render: could not decode widget", lctx.Str("module", name), lctx.Err(err))
			return
		}

		pusher := ui.ModuleUI(name)
		if pusher == nil {
			log.Error("render: module not registered", lctx.Str("module", name))
			return
		}

		if err = pusher.Update(w); err != nil {
			log.Error("render: could not update widget", lctx.Str("module", name), lctx.Err(err))
			return
		}

		log.Trace("render: widget pushed to render node", lctx.Str("module", name))
	}
}

// httpStreamOpenFunc opens an HTTP request, blocks until response headers
// are received, and returns a handle for streaming the body.
func httpStreamOpenFunc(store *streamStore, log *logger.Logger) api.GoModuleFunc {
	return func(ctx context.Context, mod api.Module, stack []uint64) {
		methodPtr := uint32(stack[0])
		methodLen := uint32(stack[1])
		urlPtr := uint32(stack[2])
		urlLen := uint32(stack[3])
		hdrPtr := uint32(stack[4])
		hdrLen := uint32(stack[5])
		bodyPtr := uint32(stack[6])
		bodyLen := uint32(stack[7])

		methodBytes, ok := mod.Memory().Read(methodPtr, methodLen)
		if !ok {
			stack[0] = 0xFFFFFFFF // i32(-1): invalid argument
			return
		}

		urlBytes, ok := mod.Memory().Read(urlPtr, urlLen)
		if !ok {
			stack[0] = 0xFFFFFFFF
			return
		}

		// Assemble optional request body.
		var bodyReader io.Reader
		if bodyLen > 0 {
			bodyBytes, ok := mod.Memory().Read(bodyPtr, bodyLen)
			if !ok {
				stack[0] = 0xFFFFFFFF
				return
			}
			bodyReader = bytes.NewReader(bodyBytes)
		}

		req, err := http.NewRequestWithContext(ctx, string(methodBytes), string(urlBytes), bodyReader)
		if err != nil {
			log.Error("http_stream_open: building request failed",
				lctx.Str("url", string(urlBytes)), lctx.Err(err))
			stack[0] = 0xFFFFFFFF
			return
		}

		// Parse and attach headers ("Key: Value\n" lines).
		if hdrLen > 0 {
			hdrBytes, ok := mod.Memory().Read(hdrPtr, hdrLen)
			if !ok {
				stack[0] = 0xFFFFFFFF
				return
			}
			for line := range strings.SplitSeq(strings.TrimRight(string(hdrBytes), "\n"), "\n") {
				if k, v, found := strings.Cut(line, ": "); found {
					req.Header.Set(k, v)
				}
			}
		}

		//nolint:bodyclose // Closed in `http_stream_close`.
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			log.Error("http_stream_open: request failed",
				lctx.Str("url", string(urlBytes)), lctx.Err(err))
			stack[0] = 0xFFFFFFFE // i32(-2): I/O error
			return
		}

		handle := store.add(&openStream{resp: resp})
		stack[0] = uint64(handle)
	}
}

func httpStreamStatusFunc(store *streamStore) api.GoModuleFunc {
	return func(_ context.Context, _ api.Module, stack []uint64) {
		handle := int32(stack[0])
		stream, ok := store.get(handle)
		if !ok {
			stack[0] = 0
			return
		}
		stack[0] = uint64(stream.resp.StatusCode)
	}
}

// httpStreamReadFunc reads up to buf_len bytes from the response body.
// Blocks until data arrives. Returns bytes read, 0 on EOF, or negative on error.
func httpStreamReadFunc(store *streamStore) api.GoModuleFunc {
	return func(ctx context.Context, mod api.Module, stack []uint64) {
		handle := int32(stack[0])
		bufPtr := uint32(stack[1])
		bufLen := uint32(stack[2])

		stream, ok := store.get(handle)
		if !ok {
			stack[0] = 0xFFFFFFFF // i32(-1): bad handle
			return
		}

		buf, ok := mod.Memory().Read(bufPtr, bufLen)
		if !ok {
			stack[0] = 0xFFFFFFFF
			return
		}

		for {
			n, err := stream.resp.Body.Read(buf)
			// Always return data before signalling EOF or an error.
			// io.Reader is permitted to return n>0 alongside a non-nil err
			// (including io.EOF) in the same call; discarding n here would
			// truncate the response and cause "unexpected EOF" in decoders.
			if n > 0 {
				stack[0] = uint64(int32(n))
				return
			}
			if err == io.EOF {
				stack[0] = 0
				return
			}
			if err != nil {
				stack[0] = 0xFFFFFFFE // i32(-2): I/O error
				return
			}
			// n == 0, err == nil: the underlying reader has no data ready yet
			// (e.g. a gzip decoder working through a chunk header, or a
			// bufio boundary). Retry until bytes or EOF arrive.
			select {
			case <-ctx.Done():
				stack[0] = 0xFFFFFFFE
				return
			default:
			}
		}
	}
}

// httpStreamCloseFunc closes the response body and releases the handle.
// Does not drain first: draining an SSE stream that never sends EOF
// would block indefinitely. Closing directly aborts the connection.
func httpStreamCloseFunc(store *streamStore) api.GoModuleFunc {
	return func(_ context.Context, _ api.Module, stack []uint64) {
		handle := int32(stack[0])
		stream := store.remove(handle)
		if stream != nil {
			_ = stream.resp.Body.Close()
		}
	}
}

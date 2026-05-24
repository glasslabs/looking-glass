---
description: 'Rules for writing Go documentation comments'
applyTo: '**/*.go'
---

# Go Documentation Rules

Follow the documentation style of the Go standard library. The goal is concise,
factual comments that tell the reader something the code itself cannot. Do not
restate what is already obvious from the signature, the type, or the name.

## General rules

- Write comments in complete English sentences.
- Start every doc comment with the name of the symbol it documents.
- Document *what* something is or *what* it returns/does, not *how* it works
  internally (implementation detail belongs in inline comments, not doc comments).
- Omit obvious information: "Error implements the error interface." is sufficient
  for an `Error() string` method — do not describe what the method does beyond
  that.
- Keep doc comments to the minimum that tells the reader something useful.

## Packages

```go
// Package httputil provides HTTP utility functions.
package httputil
```

Starts with `Package <name>`. Describes what the package
*provides*, not what it contains.

## Types

```go
// Server serves HTTP requests.
type Server struct { ... }
```

One declarative sentence starting with the type name. Describes what the type
*is*. Do not describe lifecycle, usage patterns, or method behaviour here.

## Constructors

```go
// NewServer returns a new Server.
func NewServer(addr string, handler http.Handler) *Server { ... }
```

Starts with the function name. States what it returns. Parameters are
self-documenting through their names and types — do not enumerate them in
prose unless a parameter has a non-obvious constraint or side effect.

## Methods

```go
// ServeHTTP handles an HTTP request.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) { ... }
```

Starts with the method name. States what the method does. Describe
non-obvious behaviour (error conditions, fallback logic, concurrency
guarantees) in subsequent sentences. Do not repeat information already
visible in the signature.

## Interfaces

```go
// Handler responds to an HTTP request.
type Handler interface {
    // ServeHTTP writes a reply to w for the request r.
    ServeHTTP(w ResponseWriter, r *Request)
}
```

The interface doc describes what the interface *represents*. Each method doc
states what an implementor must do, not how an existing implementation does it.

## Fields

```go
type Config struct {
    // Timeout is the maximum duration for a request.
    Timeout time.Duration
    // TLSConfig optionally specifies a TLS configuration.
    // If nil, TLS is not used.
    TLSConfig *tls.Config
}
```

One line per field where possible. Document constraints, zero-value semantics,
or optionality when they are not obvious. Do not restate the field name or type.

## Constants and variables

```go
const (
    // ModeSync fetches from the VCS on listing and on cache miss.
    ModeSync Mode = "sync"
)
```

One line starting with the constant or variable name. State what it means or
when it is used. Do not write generic descriptions like "ModeSync is the sync mode."

## Unexported symbols

Document unexported symbols only when their purpose is not obvious from the
name, or when they have non-obvious constraints that callers of unexported
functions within the same package need to know.

## What to avoid

- Do not restate the signature in prose:
  `// Versions returns the versions.` adds nothing over the name.
- Do not enumerate parameters in the constructor doc.
- Do not put behaviour or usage guidance on a type or constructor — put it on
  the methods where the behaviour actually lives.
- Do not use phrases like "This function..." or "This type..."; start with the
  symbol name directly.
- Do not end doc comments with punctuation that implies a continuation
  (`...`) unless there genuinely is one.


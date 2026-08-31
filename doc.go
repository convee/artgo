// Package artgo is a small HTTP framework built around the standard library
// net/http handler interface. It provides tree-based routing, route groups,
// middleware, request binding, validation, and common renderers.
//
// An Engine is intended to be configured before it starts serving requests.
// Route registration and middleware mutation are not synchronized for
// concurrent use.
package artgo

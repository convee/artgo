package artgo_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/convee/artgo"
)

func TestStaticRouteTakesPrecedenceOverParameter(t *testing.T) {
	router := artgo.New()
	router.GET("/users/:id", func(c *artgo.Context) {
		c.String(http.StatusOK, "user=%s", c.Param("id"))
	})
	router.GET("/users/new", func(c *artgo.Context) {
		c.String(http.StatusOK, "new-user")
	})

	assertResponse(t, router, http.MethodGet, "/users/new", http.StatusOK, "new-user")
	assertResponse(t, router, http.MethodGet, "/users/42", http.StatusOK, "user=42")
}

func TestCatchAllRouteCapturesRemainingPath(t *testing.T) {
	router := artgo.New()
	router.GET("/assets/*filepath", func(c *artgo.Context) {
		c.String(http.StatusOK, "filepath=%s", c.Param("filepath"))
	})

	assertResponse(t, router, http.MethodGet, "/assets/css/site.css", http.StatusOK, "filepath=css/site.css")
	assertResponse(t, router, http.MethodGet, "/assets/", http.StatusOK, "filepath=")
}

func TestMalformedRequestPathDoesNotPanic(t *testing.T) {
	router := artgo.New()
	router.GET("/users/:id", func(c *artgo.Context) {
		c.String(http.StatusOK, "user=%s", c.Param("id"))
	})

	assertResponse(t, router, http.MethodGet, "//users//42", http.StatusOK, "user=42")
	assertResponse(t, router, http.MethodGet, "//missing", http.StatusNotFound, "404 NOT FOUND: //missing\n")
}

func TestRouteMiddlewareIsBoundToRouteNotPathPrefix(t *testing.T) {
	var order []string
	record := func(name string) artgo.HandlerFunc {
		return func(c *artgo.Context) {
			order = append(order, name+"-before")
			c.Next()
			order = append(order, name+"-after")
		}
	}

	router := artgo.New()
	router.Use(record("root"))
	api := router.Group("/api")
	api.Use(record("api"))
	v1 := api.Group("/v1")
	v1.Use(record("v1"))
	v1.GET("/users", func(c *artgo.Context) {
		order = append(order, "route")
		c.Status(http.StatusOK)
	})
	router.GET("/apiary", func(c *artgo.Context) {
		c.Status(http.StatusOK)
	})

	assertResponse(t, router, http.MethodGet, "/api/v1/users", http.StatusOK, "")
	want := []string{"root-before", "api-before", "v1-before", "route", "v1-after", "api-after", "root-after"}
	if strings.Join(order, ",") != strings.Join(want, ",") {
		t.Fatalf("middleware order = %v, want %v", order, want)
	}

	order = nil
	assertResponse(t, router, http.MethodGet, "/apiary", http.StatusOK, "")
	if strings.Join(order, ",") != "root-before,root-after" {
		t.Fatalf("/apiary unexpectedly ran API middleware: %v", order)
	}
}

func TestGlobalMiddlewareRunsForNotFound(t *testing.T) {
	var observedPath string
	var observedStatus int
	router := artgo.New()
	router.Use(func(c *artgo.Context) {
		c.Next()
		observedPath = c.Path
		observedStatus = c.StatusCode
	})

	assertResponse(t, router, http.MethodGet, "/missing", http.StatusNotFound, "404 NOT FOUND: /missing\n")
	if observedPath != "/missing" || observedStatus != http.StatusNotFound {
		t.Fatalf("not-found middleware observation = path %q status %d", observedPath, observedStatus)
	}
}

func TestContextAbortStopsRemainingRouteHandlers(t *testing.T) {
	var routeRan bool
	var observedAborted bool
	router := artgo.New()
	router.Use(func(c *artgo.Context) {
		c.Next()
		observedAborted = c.IsAborted()
	})
	router.Use(func(c *artgo.Context) {
		c.Abort()
		c.Status(http.StatusForbidden)
	})
	router.GET("/private", func(c *artgo.Context) {
		routeRan = true
	})

	assertResponse(t, router, http.MethodGet, "/private", http.StatusForbidden, "")
	if routeRan {
		t.Fatal("route handler after Abort ran")
	}
	if !observedAborted {
		t.Fatal("earlier middleware could not observe Abort")
	}
}

func TestCompletedHandlerChainIsNotAborted(t *testing.T) {
	var observedAborted = true
	router := artgo.New()
	router.Use(func(c *artgo.Context) {
		c.Next()
		observedAborted = c.IsAborted()
	})
	router.GET("/completed", func(c *artgo.Context) { c.Status(http.StatusOK) })

	assertResponse(t, router, http.MethodGet, "/completed", http.StatusOK, "")
	if observedAborted {
		t.Fatal("a normally completed chain was reported as aborted")
	}
}

func TestInvalidAndDuplicateRoutesPanicAtRegistration(t *testing.T) {
	tests := []struct {
		name     string
		register func(router *artgo.Engine)
	}{
		{
			name: "nil middleware",
			register: func(router *artgo.Engine) {
				router.Use(nil)
			},
		},
		{
			name: "duplicate",
			register: func(router *artgo.Engine) {
				router.GET("/users", func(c *artgo.Context) {})
				router.GET("/users/", func(c *artgo.Context) {})
			},
		},
		{
			name: "conflicting wildcard",
			register: func(router *artgo.Engine) {
				router.GET("/users/:id", func(c *artgo.Context) {})
				router.GET("/users/:name", func(c *artgo.Context) {})
			},
		},
		{
			name: "non-final catch-all",
			register: func(router *artgo.Engine) {
				router.GET("/files/*rest/edit", func(c *artgo.Context) {})
			},
		},
		{
			name: "unnamed wildcard",
			register: func(router *artgo.Engine) {
				router.GET("/files/*", func(c *artgo.Context) {})
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Fatal("expected route registration to panic")
				}
			}()
			test.register(artgo.New())
		})
	}
}

func assertResponse(t *testing.T, handler http.Handler, method, target string, wantStatus int, wantBody string) {
	t.Helper()
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(method, target, nil))
	if recorder.Code != wantStatus {
		t.Fatalf("%s %s status = %d, want %d", method, target, recorder.Code, wantStatus)
	}
	if recorder.Body.String() != wantBody {
		t.Fatalf("%s %s body = %q, want %q", method, target, recorder.Body.String(), wantBody)
	}
}

func TestAllVerbsAreRoutable(t *testing.T) {
	router := artgo.New()
	register := map[string]func(string, artgo.HandlerFunc){
		http.MethodGet:     router.GET,
		http.MethodPost:    router.POST,
		http.MethodPut:     router.PUT,
		http.MethodDelete:  router.DELETE,
		http.MethodPatch:   router.PATCH,
		http.MethodHead:    router.HEAD,
		http.MethodOptions: router.OPTIONS,
	}
	for method, register := range register {
		method := method
		register("/things/"+strings.ToLower(method), func(c *artgo.Context) {
			c.String(http.StatusOK, "handled=%s", method)
		})
	}

	for method := range register {
		assertResponse(t, router, method, "/things/"+strings.ToLower(method),
			http.StatusOK, "handled="+method)
	}
}

func TestHandleRegistersArbitraryMethod(t *testing.T) {
	router := artgo.New()
	router.Handle("REPORT", "/docs", func(c *artgo.Context) {
		c.String(http.StatusOK, "report")
	})

	assertResponse(t, router, "REPORT", "/docs", http.StatusOK, "report")
}

func TestAnyRegistersEveryStandardVerb(t *testing.T) {
	router := artgo.New()
	router.Any("/ping", func(c *artgo.Context) {
		c.String(http.StatusOK, "pong")
	})

	for _, method := range []string{
		http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete,
		http.MethodPatch, http.MethodHead, http.MethodOptions,
	} {
		assertResponse(t, router, method, "/ping", http.StatusOK, "pong")
	}
}

func TestKnownPathWithWrongMethodReturns405WithAllowHeader(t *testing.T) {
	router := artgo.New()
	router.GET("/articles/:id", func(c *artgo.Context) { c.String(http.StatusOK, "get") })
	router.DELETE("/articles/:id", func(c *artgo.Context) { c.String(http.StatusOK, "delete") })

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/articles/7", nil))

	if recorder.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusMethodNotAllowed)
	}
	// Allow 头必须稳定排序：roots 是 map，未排序时该断言会随机失败
	if got := recorder.Header().Get("Allow"); got != "DELETE, GET" {
		t.Fatalf("Allow = %q, want %q", got, "DELETE, GET")
	}
}

func TestUnknownPathStillReturns404(t *testing.T) {
	router := artgo.New()
	router.GET("/articles/:id", func(c *artgo.Context) { c.String(http.StatusOK, "get") })

	assertResponse(t, router, http.MethodPost, "/nowhere",
		http.StatusNotFound, "404 NOT FOUND: /nowhere\n")
}

func TestGlobalMiddlewareStillRunsOnFallbackResponses(t *testing.T) {
	router := artgo.New()
	var ran bool
	router.Use(func(c *artgo.Context) {
		ran = true
		c.Next()
	})
	router.GET("/only-get", func(c *artgo.Context) { c.String(http.StatusOK, "ok") })

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/only-get", nil))

	if recorder.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want 405", recorder.Code)
	}
	if !ran {
		t.Fatal("global middleware did not run on the 405 path")
	}
}

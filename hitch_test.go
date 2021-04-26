package hitch

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHitch_path(t *testing.T) {
	type fields struct {
		basePath string
	}
	type args struct {
		p string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   string
	}{
		{
			name:   "No base path, no path",
			fields: fields{basePath: ""},
			args:   args{p: ""},
			want:   "",
		}, {
			name:   "No base path, some path",
			fields: fields{basePath: ""},
			args:   args{p: "/foo"},
			want:   "/foo",
		}, {
			name:   "No base path, some path without /",
			fields: fields{basePath: ""},
			args:   args{p: "foo"},
			want:   "/foo",
		}, {
			name:   "No base path, some path with ending /",
			fields: fields{basePath: ""},
			args:   args{p: "foo/"},
			want:   "/foo/",
		}, {
			name:   "No base path, some complex path",
			fields: fields{basePath: ""},
			args:   args{p: "/foo/bar"},
			want:   "/foo/bar",
		}, {
			name:   "Some base path, no path",
			fields: fields{basePath: "/base"},
			args:   args{p: ""},
			want:   "/base",
		}, {
			name:   "Some base path, some path",
			fields: fields{basePath: "/base"},
			args:   args{p: "/foo"},
			want:   "/base/foo",
		}, {
			name:   "Some base path, some path without /",
			fields: fields{basePath: "/base"},
			args:   args{p: "foo"},
			want:   "/base/foo",
		}, {
			name:   "Some base path, some path with ending /",
			fields: fields{basePath: "/base"},
			args:   args{p: "foo/"},
			want:   "/base/foo/",
		}, {
			name:   "Some base path, some complex path",
			fields: fields{basePath: "/base"},
			args:   args{p: "/foo/bar"},
			want:   "/base/foo/bar",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &Hitch{
				basePath: tt.fields.basePath,
			}
			if got := h.path(tt.args.p); got != tt.want {
				t.Errorf("path() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHitch_Get(t *testing.T) {
	t.Run("Single route, no param, no middleware", func(t *testing.T) {
		r := New()
		r.GET("/foo/bar", func(w http.ResponseWriter, r *http.Request) {
			if _, err := w.Write([]byte("OK")); err != nil {
				t.Fatal(err)
			}
		})

		srv := httptest.NewServer(r.Router())
		defer srv.Close()

		res, err := http.Get(srv.URL + "/foo/bar")
		assertNoError(t, err)
		assertResponse(t, res, 200, "", "", "OK")
	})

	t.Run("Single route, some param, no middleware", func(t *testing.T) {
		r := New()
		r.GET("/foo/bar/:id", func(w http.ResponseWriter, r *http.Request) {
			if _, err := w.Write([]byte("OK " + Params(r).ByName("id"))); err != nil {
				t.Fatal(err)
			}
		})

		srv := httptest.NewServer(r.Router())
		defer srv.Close()

		res, err := http.Get(srv.URL + "/foo/bar/john")
		assertNoError(t, err)
		assertResponse(t, res, 200, "", "", "OK john")
	})

	t.Run("Multiple routes, some param, no middleware", func(t *testing.T) {
		r := New()
		r.GET("/foo/bar/:id", func(w http.ResponseWriter, r *http.Request) {
			if _, err := w.Write([]byte("OK " + Params(r).ByName("id"))); err != nil {
				t.Fatal(err)
			}
		})
		r.POST("/foo/bar", func(w http.ResponseWriter, r *http.Request) {
			t.Error("not expected to be here")
		})
		r.GET("/foo/bar/:id/whatever", func(w http.ResponseWriter, r *http.Request) {
			t.Error("not expected to be here")
		})
		r.PUT("/foo/bar/:id", func(w http.ResponseWriter, r *http.Request) {
			t.Error("not expected to be here")
		})
		r.DELETE("/foo/bar/:id", func(w http.ResponseWriter, r *http.Request) {
			t.Error("not expected to be here")
		})

		srv := httptest.NewServer(r.Router())
		defer srv.Close()

		res, err := http.Get(srv.URL + "/foo/bar/john")
		assertNoError(t, err)
		assertResponse(t, res, 200, "", "", "OK john")
	})

	t.Run("Multiple routes, some param, some middleware", func(t *testing.T) {
		r := New()
		r = r.WithMiddleware(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				addHeader(w)
				next.ServeHTTP(w, r)
			})
		})
		r.GET("/foo/bar/:id", func(w http.ResponseWriter, r *http.Request) {
			if _, err := w.Write([]byte("OK " + Params(r).ByName("id"))); err != nil {
				t.Fatal(err)
			}
		})
		r.POST("/foo/bar", func(w http.ResponseWriter, r *http.Request) {
			t.Error("not expected to be here")
		})
		r.GET("/foo/bar/:id/whatever", func(w http.ResponseWriter, r *http.Request) {
			t.Error("not expected to be here")
		})
		r.PUT("/foo/bar/:id", func(w http.ResponseWriter, r *http.Request) {
			t.Error("not expected to be here")
		})
		r.DELETE("/foo/bar/:id", func(w http.ResponseWriter, r *http.Request) {
			t.Error("not expected to be here")
		})

		srv := httptest.NewServer(r.Router())
		defer srv.Close()

		res, err := http.Get(srv.URL + "/foo/bar/john")
		assertNoError(t, err)
		assertResponse(t, res, 200, "foo", "", "OK john")
	})

	t.Run("Multiple routes, some param, some middleware and inline middleware", func(t *testing.T) {
		r := New()
		r = r.WithHandlerMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Minirouter", "bar")
		}))
		r.GET("/foo/bar/:id", func(w http.ResponseWriter, r *http.Request) {
			if _, err := w.Write([]byte("OK " + Params(r).ByName("id"))); err != nil {
				t.Fatal(err)
			}
		}, func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				addHeaderID(w, r)
				next.ServeHTTP(w, r)
			})
		})
		r.POST("/foo/bar", func(w http.ResponseWriter, r *http.Request) {
			t.Error("not expected to be here")
		})
		r.GET("/foo/bar/:id/whatever", func(w http.ResponseWriter, r *http.Request) {
			t.Error("not expected to be here")
		})
		r.PUT("/foo/bar/:id", func(w http.ResponseWriter, r *http.Request) {
			t.Error("not expected to be here")
		})
		r.DELETE("/foo/bar/:id", func(w http.ResponseWriter, r *http.Request) {
			t.Error("not expected to be here")
		})

		srv := httptest.NewServer(r.Router())
		defer srv.Close()

		res, err := http.Get(srv.URL + "/foo/bar/john")
		assertNoError(t, err)
		assertResponse(t, res, 200, "bar", "john", "OK john")
	})
}

func addHeader(w http.ResponseWriter) {
	w.Header().Set("X-Minirouter", "foo")
}

func addHeaderID(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("X-Id", Params(r).ByName("id"))
}

func assertNoError(t *testing.T, err error) {
	if err != nil {
		t.Fatalf("Expected no error, got %s", err)
	}
}

func assertResponse(t *testing.T, res *http.Response, status int, minirouterHeader, idHeader, expectedBody string) {
	t.Helper()
	if res.StatusCode != status {
		t.Errorf("Wrong response status code. Expected %d, got %d", status, res.StatusCode)
	}
	v := res.Header.Get("X-Minirouter")
	if v != minirouterHeader {
		t.Errorf("Wrong response X-Minirouter. Expected %s, got %s", minirouterHeader, v)
	}
	v = res.Header.Get("X-Id")
	if v != idHeader {
		t.Errorf("Wrong response X-Id. Expected %s, got %s", idHeader, v)
	}

	body, err := io.ReadAll(res.Body)
	res.Body.Close()
	if err != nil {
		t.Fatal(err)
	}
	cleanBody := strings.TrimSpace(string(body))
	if cleanBody != expectedBody {
		t.Errorf("Response body is not as expected. Expected '%s', got '%s'", expectedBody, cleanBody)
	}
}

func TestHitch_Group(t *testing.T) {
	t.Run("Multiple routes, some param, some middleware and inline middleware", func(t *testing.T) {
		r := New()
		r = r.WithHandlerMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Minirouter", "bar")
		}))
		r.GET("/foo/bar/:id", func(w http.ResponseWriter, r *http.Request) {
			t.Error("not expected to be here")
		})
		r.POST("/foo/bar", func(w http.ResponseWriter, r *http.Request) {
			body, err := io.ReadAll(r.Body)
			r.Body.Close()
			if err != nil {
				t.Fatal(err)
			}
			w.WriteHeader(201)
			if _, err := w.Write([]byte("OK " + string(body))); err != nil {
				t.Fatal(err)
			}
		})

		g := r.SubPath("/sub")
		g = g.WithHandlerMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Minirouter", "override-bar")
		}))
		g.GET("/foo/bar/:id", func(w http.ResponseWriter, r *http.Request) {
			if _, err := w.Write([]byte("OK " + Params(r).ByName("id"))); err != nil {
				t.Fatal(err)
			}
		}, func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				addHeaderID(w, r)
				next.ServeHTTP(w, r)
			})
		})
		g.PUT("/foo/bar/:id", func(w http.ResponseWriter, r *http.Request) {
			t.Error("not expected to be here")
		})
		g.DELETE("/foo/bar/:id", func(w http.ResponseWriter, r *http.Request) {
			t.Error("not expected to be here")
		})

		srv := httptest.NewServer(r.Router())
		defer srv.Close()

		res, err := http.Get(srv.URL + "/sub/foo/bar/john")
		assertNoError(t, err)
		assertResponse(t, res, 200, "override-bar", "john", "OK john")

		res, err = http.Post(srv.URL+"/foo/bar", "text/plain", bytes.NewReader([]byte("X")))
		assertNoError(t, err)
		assertResponse(t, res, 201, "bar", "", "OK X")
	})
}

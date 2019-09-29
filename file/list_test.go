package file_test

import (
	"net/http"
	"net/url"
	"testing"

	assert "github.com/stretchr/testify/require"
	"github.com/uploadcare/uploadcare-go/file"
	"github.com/uploadcare/uploadcare-go/ucare"
)

func TestListParamsEncodeRequest(t *testing.T) {
	t.Parallel()

	cases := []struct {
		test string

		params        *file.ListParams
		expectedQuery url.Values
	}{{
		test: "full list of params",
		params: &file.ListParams{
			Removed:  ucare.Bool(true),
			Stored:   ucare.Bool(false),
			Limit:    ucare.Int64(500),
			Ordering: ucare.String(file.OrderBySizeAsc),
		},
		expectedQuery: url.Values{
			"removed":  []string{"true"},
			"stored":   []string{"false"},
			"limit":    []string{"500"},
			"ordering": []string{"size"},
		},
	}, {
		test:          "empty list",
		params:        &file.ListParams{},
		expectedQuery: url.Values{},
	}, {
		test: "part of the params are filled",
		params: &file.ListParams{
			Stored:   ucare.Bool(false),
			Ordering: ucare.String(file.OrderByUploadedAtDesc),
		},
		expectedQuery: url.Values{
			"stored":   []string{"false"},
			"ordering": []string{"-datetime_uploaded"},
		},
	}}

	for _, c := range cases {
		c := c
		t.Run(c.test, func(t *testing.T) {
			t.Parallel()

			req, _ := http.NewRequest("GET", "", nil)
			c.params.EncodeRequest(req)
			q := req.URL.Query()

			if len(c.expectedQuery) == 0 && req.URL.RawQuery != "" {
				t.Error("should have no qparams")
			}
			for k, v := range c.expectedQuery {
				assert.Equal(t, q[k], v)
			}
		})
	}
}
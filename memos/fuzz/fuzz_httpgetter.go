package fuzz

import (
	"strings"

	"github.com/AdamKorcz/go-118-fuzz-build/testing"
	"github.com/usememos/memos/plugin/httpgetter"
)

func FuzzURLValidation(f *testing.F) {

	f.Fuzz(func(t *testing.T, urlStr string) {
		_ = httpgetter.ValidateURL(urlStr)
	})
}

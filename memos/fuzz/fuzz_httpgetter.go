package fuzz

import (
	"strings"
	"testing"
	"github.com/usememos/memos/plugin/httpgetter"
)

func FuzzURLValidation(f *testing.F) {

	f.Fuzz(func(t *testing.T, urlStr string) {
		_ = httpgetter.ValidateURL(urlStr)
	})
}

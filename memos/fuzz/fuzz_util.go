package fuzz

import (

	"testing"
	"github.com/usememos/memos/internal/base"
	"github.com/usememos/memos/internal/util"
)

func FuzzValidateEmail(f *testing.F) {

	f.Fuzz(func(t *testing.T, email string) {
		_ = util.ValidateEmail(email)
	})
}


func FuzzUIDMatcher(f *testing.F) {

	f.Fuzz(func(t *testing.T, uid string) {
		_ = base.UIDMatcher.MatchString(uid)
	})
}

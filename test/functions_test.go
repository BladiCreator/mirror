package test

import (
	"testing"

	"github.com/mirror/mirror/internal/functions"
)

func TestBuiltinFunctions(t *testing.T) {
	fMap := functions.ResolveFuncs([]string{"strings:st"})
	
	toTitleRaw, ok := fMap["fn_st_toTitle"]
	if !ok {
		t.Fatal("fn_st_toTitle function not found")
	}
	
	toTitle := toTitleRaw.(func(string) string)
	res := toTitle("hello_world")
	if res != "Hello_world" {
		t.Errorf("expected Hello_world, got %s", res)
	}

	toUpperRaw := fMap["fn_st_toUpper"].(func(string) string)
	if toUpperRaw("abc") != "ABC" {
		t.Errorf("expected ABC, got %s", toUpperRaw("abc"))
	}
}

package db

import (
	"reflect"
	"testing"
)

func TestTextArrayGormValueUsesPostgresArrayExpression(t *testing.T) {
	expression := (TextArray{"plain", "comma,value"}).GormValue(nil, nil)
	if expression.SQL != "ARRAY[?,?]::text[]" {
		t.Fatalf("unexpected SQL expression: %s", expression.SQL)
	}
	if !reflect.DeepEqual(expression.Vars, []any{"plain", "comma,value"}) {
		t.Fatalf("unexpected array parameters: %#v", expression.Vars)
	}
	if empty := (TextArray{}).GormValue(nil, nil).SQL; empty != "ARRAY[]::text[]" {
		t.Fatalf("unexpected empty-array expression: %s", empty)
	}
}

func TestTextArrayScansPostgresArrayLiteral(t *testing.T) {
	var got TextArray
	if err := got.Scan(`{"plain","comma,value","quote\"value","slash\\value"}`); err != nil {
		t.Fatalf("Scan returned an error: %v", err)
	}
	want := TextArray{"plain", "comma,value", `quote"value`, `slash\value`}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Scan returned %#v, want %#v", got, want)
	}
}

func TestTextArrayRejectsNullElements(t *testing.T) {
	var got TextArray
	if err := got.Scan(`{NULL}`); err == nil {
		t.Fatal("NULL elements should be rejected")
	}
}

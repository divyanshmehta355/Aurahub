package db

import (
	"context"
	"fmt"
	"strings"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type TextArray []string

func (a TextArray) GormDataType() string {
	return "text[]"
}

func (a TextArray) GormValue(_ context.Context, _ *gorm.DB) clause.Expr {
	if len(a) == 0 {
		return clause.Expr{SQL: "ARRAY[]::text[]"}
	}
	placeholders := make([]string, len(a))
	values := make([]any, len(a))
	for index, value := range a {
		placeholders[index] = "?"
		values[index] = value
	}
	return clause.Expr{
		SQL:  "ARRAY[" + strings.Join(placeholders, ",") + "]::text[]",
		Vars: values,
	}
}

func (a *TextArray) Scan(value any) error {
	switch value := value.(type) {
	case nil:
		*a = nil
		return nil
	case []string:
		*a = append((*a)[:0], value...)
		return nil
	case string:
		return a.scanLiteral(value)
	case []byte:
		return a.scanLiteral(string(value))
	default:
		return fmt.Errorf("cannot scan PostgreSQL text array from %T", value)
	}
}

func (a *TextArray) scanLiteral(value string) error {
	if len(value) < 2 || value[0] != '{' || value[len(value)-1] != '}' {
		return fmt.Errorf("invalid PostgreSQL text array")
	}
	if len(value) == 2 {
		*a = TextArray{}
		return nil
	}

	result := make(TextArray, 0)
	for index := 1; index < len(value)-1; {
		var item strings.Builder
		quoted := value[index] == '"'
		if quoted {
			index++
		}
		closed := !quoted
		for index < len(value)-1 {
			character := value[index]
			if character == '\\' {
				index++
				if index >= len(value)-1 {
					return fmt.Errorf("invalid escape in PostgreSQL text array")
				}
				item.WriteByte(value[index])
				index++
				continue
			}
			if quoted && character == '"' {
				index++
				closed = true
				break
			}
			if !quoted && (character == ',' || character == '}') {
				break
			}
			item.WriteByte(character)
			index++
		}
		if !closed {
			return fmt.Errorf("unterminated quoted value in PostgreSQL text array")
		}
		if !quoted && item.String() == "NULL" {
			return fmt.Errorf("NULL elements are not supported in PostgreSQL text arrays")
		}
		result = append(result, item.String())
		if index == len(value)-1 {
			break
		}
		if value[index] != ',' {
			return fmt.Errorf("invalid delimiter in PostgreSQL text array")
		}
		index++
	}
	*a = result
	return nil
}

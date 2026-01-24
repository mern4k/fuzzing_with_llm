package fuzz

import (
	"context"

	"github.com/AdamKorcz/go-118-fuzz-build/testing"
	"github.com/usememos/memos/plugin/filter"
)

func FuzzFilterCompile(f *testing.F) {
	f.Fuzz(func(t *testing.T, filterExpr string) {
		
		schema := filter.NewSchema()
		engine, err := filter.NewEngine(schema)
		if err != nil {
			return
		}

		ctx := context.Background()

		program, err := engine.Compile(ctx, filterExpr)
		if err != nil {
			return
		}

		if program != nil {
			dialects := []filter.DialectName{
				filter.DialectSQLite,
				filter.DialectMySQL,
				filter.DialectPostgres,
			}

			for _, dialect := range dialects {
				opts := filter.RenderOptions{
					Dialect:		dialect,
					PlaceholderOffset:	0,
				}
				_, _ = program.Render(opts)
			}
		}
	})
}

func FuzzFilterNormalize(f *testing.F) {
	
	f.Fuzz(func(t *testing.T, expr string) {
		schema := filter.NewSchema()
		engine, err := filter.NewEngine(schema)
		if err != nil {
			return
		}
		ctx := context.Background()
		_, _ = engine.Compile(ctx, expr)
	})
}

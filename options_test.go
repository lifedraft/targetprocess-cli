package targetprocess

import "testing"

func TestResolveSearchOpts(t *testing.T) {
	t.Run("defaults", func(t *testing.T) {
		so := resolveSearchOpts(nil)
		if so.take != defaultTake {
			t.Errorf("take = %d, want %d", so.take, defaultTake)
		}
		if so.selectExpr != "" {
			t.Errorf("selectExpr = %q, want empty", so.selectExpr)
		}
		if so.orderBy != "" {
			t.Errorf("orderBy = %q, want empty", so.orderBy)
		}
	})

	t.Run("with_options", func(t *testing.T) {
		so := resolveSearchOpts([]SearchOption{
			WithSelect("id", "name", "entityState.name as state"),
			WithTake(50),
			WithOrderBy("createDate desc"),
		})
		if so.take != 50 {
			t.Errorf("take = %d, want 50", so.take)
		}
		if want := "id,name,entityState.name as state"; so.selectExpr != want {
			t.Errorf("selectExpr = %q, want %q", so.selectExpr, want)
		}
		if so.orderBy != "createDate desc" {
			t.Errorf("orderBy = %q, want %q", so.orderBy, "createDate desc")
		}
	})
}

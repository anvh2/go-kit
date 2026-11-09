package example

import (
	"errors"
	"strings"
	"testing"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/utils/tests"
)

// dryRun returns a database that builds SQL without executing it (no server needed).
func dryRun(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(tests.DummyDialector{}, &gorm.Config{DryRun: true, DisableAutomaticPing: true})
	if err != nil {
		t.Fatal(err)
	}
	return db
}

func sqlOf(t *testing.T, db *gorm.DB) (string, []interface{}) {
	t.Helper()
	if db.Error != nil {
		t.Fatalf("query error: %v", db.Error)
	}
	return db.Statement.SQL.String(), db.Statement.Vars
}

func TestFiltersOrderAndPaging(t *testing.T) {
	var plots []Plot
	q := NewPlotQuerySet(dryRun(t)).
		FarmIDEq(7).
		StageIn("growing", "harvesting").
		CodeLike("EF-%").
		RetiredAtIsNull().
		AreaM2Gte(100).
		OrderAscByCode().
		Limit(10).Offset(20)
	if err := q.All(&plots); err != nil {
		t.Fatal(err)
	}
	got, vars := sqlOf(t, q.GetDB().Find(&plots))
	want := "SELECT * FROM `plots` WHERE `plots`.`farm_id` = ? AND `plots`.`stage` IN (?,?) AND `plots`.`code` LIKE ? AND `plots`.`retired_at` IS NULL AND `plots`.`area_m2` >= ? ORDER BY `plots`.`code` LIMIT ? OFFSET ?"
	if got != want {
		t.Fatalf("SQL =\n%s\nwant\n%s", got, want)
	}
	if len(vars) != 7 || vars[0] != uint64(7) || vars[1] != Stage("growing") {
		t.Fatalf("vars = %v", vars)
	}
}

func TestNegationsAndDescendingOrder(t *testing.T) {
	var plots []Plot
	db := NewPlotQuerySet(dryRun(t)).IDNotIn(1, 2).CodeNotLike("X%").RetiredAtIsNotNull().ActiveNe(true).OrderDescByCreatedAt().GetDB().Find(&plots)
	got, _ := sqlOf(t, db)
	want := "SELECT * FROM `plots` WHERE `plots`.`id` NOT IN (?,?) AND `plots`.`code` NOT LIKE ? AND `plots`.`retired_at` IS NOT NULL AND `plots`.`active` <> ? ORDER BY `plots`.`created_at` DESC"
	if got != want {
		t.Fatalf("SQL =\n%s\nwant\n%s", got, want)
	}
}

func TestQuerySetsBranchWithoutSharingConditions(t *testing.T) {
	base := NewPlotQuerySet(dryRun(t)).FarmIDEq(1)
	_ = base.StageEq("growing")
	var plots []Plot
	got, _ := sqlOf(t, base.CodeEq("A").GetDB().Find(&plots))
	want := "SELECT * FROM `plots` WHERE `plots`.`farm_id` = ? AND `plots`.`code` = ?"
	if got != want {
		t.Fatalf("a sibling query leaked into its base:\n%s\nwant\n%s", got, want)
	}
}

func TestEmptyInIsAnError(t *testing.T) {
	var plots []Plot
	if err := NewPlotQuerySet(dryRun(t)).StageIn().All(&plots); err == nil {
		t.Fatal("StageIn() with no values must fail")
	}
}

func TestCountSelectOneAndUpdater(t *testing.T) {
	db := dryRun(t)
	if _, err := NewPlotQuerySet(db).FarmIDEq(1).Count(); err != nil {
		t.Fatal(err)
	}
	var p Plot
	got, _ := sqlOf(t, NewPlotQuerySet(db).Select(PlotDBSchema.ID, PlotDBSchema.Code).IDEq(3).GetDB().First(&p))
	if want := "SELECT `id`,`code` FROM `plots` WHERE `plots`.`id` = ? ORDER BY `plots`.`id` LIMIT ?"; got != want {
		t.Fatalf("SQL =\n%s\nwant\n%s", got, want)
	}

	upd := NewPlotQuerySet(db).IDEq(3).GetUpdater().SetStage("growing").SetRetiredAt(nil)
	res := upd.db.Updates(upd.fields)
	got, _ = sqlOf(t, res)
	if want := "UPDATE `plots` SET `retired_at`=?,`stage`=? WHERE `plots`.`id` = ?"; got != want {
		t.Fatalf("SQL =\n%s\nwant\n%s", got, want)
	}
}

func TestUnconditionedWritesAreRefused(t *testing.T) {
	db := dryRun(t)
	if err := NewPlotQuerySet(db).GetUpdater().SetCode("x").Update(); !errors.Is(err, gorm.ErrMissingWhereClause) {
		t.Fatalf("update without conditions: err = %v, want ErrMissingWhereClause", err)
	}
	if err := NewPlotQuerySet(db).Delete(); !errors.Is(err, gorm.ErrMissingWhereClause) {
		t.Fatalf("delete without conditions: err = %v, want ErrMissingWhereClause", err)
	}
	if err := (&Plot{}).Update(db); err == nil {
		t.Fatal("Update with no fields must fail")
	}
}

func TestSchemaNames(t *testing.T) {
	if PlotDBSchema.AreaM2.String() != "area_m2" || PlotDBSchema.CreatedAt != "created_at" {
		t.Fatalf("schema = %+v", PlotDBSchema)
	}
	_ = time.Now
}

// capture records the SQL of every statement the database would run.
func capture(t *testing.T) (*gorm.DB, *[]string) {
	t.Helper()
	db := dryRun(t)
	var sqls []string
	record := func(tx *gorm.DB) { sqls = append(sqls, tx.Statement.SQL.String()) }
	for name, register := range map[string]func(string, func(*gorm.DB)) error{
		"query": func(n string, f func(*gorm.DB)) error {
			return db.Callback().Query().After("gorm:query").Register(n, f)
		},
		"row": func(n string, f func(*gorm.DB)) error { return db.Callback().Row().After("gorm:row").Register(n, f) },
	} {
		if err := register("capture:"+name, record); err != nil {
			t.Fatal(err)
		}
	}
	return db, &sqls
}

func TestQuerySetCanBeReusedAfterTerminalCalls(t *testing.T) {
	db, sqls := capture(t)
	base := NewPlotQuerySet(db).FarmIDEq(1)

	var one Plot
	_ = base.One(&one)
	var all []Plot
	if err := base.All(&all); err != nil {
		t.Fatalf("All after One: %v", err)
	}
	if _, err := base.Count(); err != nil {
		t.Fatalf("Count after One: %v", err)
	}
	if got := *sqls; len(got) != 3 {
		t.Fatalf("captured %d statements, want 3: %q", len(got), got)
	}
	if first := (*sqls)[0]; !strings.Contains(first, "ORDER BY `plots`.`id` LIMIT") {
		t.Fatalf("One = %s, want ORDER BY id LIMIT", first)
	}
	for _, later := range (*sqls)[1:] {
		if strings.Contains(later, "LIMIT") || strings.Contains(later, "ORDER BY") {
			t.Fatalf("One leaked into the query set: %s", later)
		}
	}
}

func TestCountIgnoresPaging(t *testing.T) {
	db, sqls := capture(t)
	if _, err := NewPlotQuerySet(db).FarmIDEq(1).OrderAscByCode().Limit(10).Offset(20).Count(); err != nil {
		t.Fatal(err)
	}
	got := (*sqls)[0]
	if strings.Contains(got, "LIMIT") || strings.Contains(got, "OFFSET") {
		t.Fatalf("Count kept paging, which makes it return no row: %s", got)
	}
}

func TestUpdatersBranchWithoutSharingColumns(t *testing.T) {
	base := NewPlotQuerySet(dryRun(t)).IDEq(1).GetUpdater()
	_ = base.SetCode("a")
	b := base.SetStage("growing")
	if _, leaked := b.fields["code"]; leaked {
		t.Fatalf("a sibling updater leaked its column: %v", b.fields)
	}
	if len(base.fields) != 0 {
		t.Fatalf("the base updater changed: %v", base.fields)
	}
}

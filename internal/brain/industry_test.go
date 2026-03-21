package brain

import (
	"strings"
	"testing"
)

func TestLoadIndustries(t *testing.T) {
	if err := LoadIndustries(); err != nil {
		t.Fatalf("LoadIndustries: %v", err)
	}

	ids := ListIndustries()
	if len(ids) < 3 {
		t.Errorf("expected at least 3 industry templates, got %d", len(ids))
	}

	// Verify specific industries loaded
	for _, id := range []string{"saas", "manufacturing", "sales"} {
		if GetIndustry(id) == nil {
			t.Errorf("industry %q not loaded", id)
		}
	}
}

func TestMatchIndustry_ExactKeyword(t *testing.T) {
	LoadIndustries()

	tmpl := MatchIndustry("saas")
	if tmpl == nil {
		t.Fatal("expected saas match for 'saas'")
	}
	if tmpl.ID != "saas" {
		t.Errorf("got %q, want 'saas'", tmpl.ID)
	}
}

func TestMatchIndustry_ChineseKeyword(t *testing.T) {
	LoadIndustries()

	tmpl := MatchIndustry("我们是做软件的")
	if tmpl == nil {
		t.Fatal("expected saas match for Chinese keyword '软件'")
	}
	if tmpl.ID != "saas" {
		t.Errorf("got %q, want 'saas'", tmpl.ID)
	}
}

func TestMatchIndustry_Sentence(t *testing.T) {
	LoadIndustries()

	tmpl := MatchIndustry("We're a SaaS startup building tools")
	if tmpl == nil {
		t.Fatal("expected saas match for sentence")
	}
	if tmpl.ID != "saas" {
		t.Errorf("got %q, want 'saas'", tmpl.ID)
	}
}

func TestMatchIndustry_Manufacturing(t *testing.T) {
	LoadIndustries()

	tmpl := MatchIndustry("我们是一家制造工厂")
	if tmpl == nil {
		t.Fatal("expected manufacturing match")
	}
	if tmpl.ID != "manufacturing" {
		t.Errorf("got %q, want 'manufacturing'", tmpl.ID)
	}
}

func TestMatchIndustry_Sales(t *testing.T) {
	LoadIndustries()

	tmpl := MatchIndustry("外贸销售公司")
	if tmpl == nil {
		t.Fatal("expected sales match")
	}
	if tmpl.ID != "sales" {
		t.Errorf("got %q, want 'sales'", tmpl.ID)
	}
}

func TestMatchIndustry_NoMatch(t *testing.T) {
	LoadIndustries()

	tmpl := MatchIndustry("我们做水产养殖的")
	if tmpl != nil {
		t.Errorf("expected nil for unmatched industry, got %q", tmpl.ID)
	}
}

func TestMatchIndustry_CaseInsensitive(t *testing.T) {
	LoadIndustries()

	tmpl := MatchIndustry("SAAS")
	if tmpl == nil {
		t.Fatal("expected saas match for 'SAAS'")
	}
	if tmpl.ID != "saas" {
		t.Errorf("got %q, want 'saas'", tmpl.ID)
	}
}

func TestMatchIndustry_Empty(t *testing.T) {
	LoadIndustries()

	tmpl := MatchIndustry("")
	if tmpl != nil {
		t.Errorf("expected nil for empty string, got %v", tmpl)
	}
}

func TestBuildIndustryContext_WithTemplate(t *testing.T) {
	tmpl := &IndustryTemplate{
		Name:   "科技/SaaS",
		NameEn: "Technology / SaaS",
		KPIReference: []KPIRef{
			{Name: "MRR", Frequency: "monthly", Description: "Monthly recurring revenue"},
		},
		RoleReference: []RoleRef{
			{Title: "Tech Lead", Type: "human"},
		},
		QuestionRef: []string{"What did you deploy?"},
		AlertReference: []AlertRef{
			{Condition: "Churn spike", Action: "Notify CEO"},
		},
	}

	ctx := BuildIndustryContext(tmpl)

	if !strings.Contains(ctx, "Industry Context") {
		t.Error("should contain 'Industry Context' header")
	}
	if !strings.Contains(ctx, "MRR") {
		t.Error("should contain KPI reference")
	}
	if !strings.Contains(ctx, "Tech Lead") {
		t.Error("should contain role reference")
	}
	if !strings.Contains(ctx, "What did you deploy") {
		t.Error("should contain question reference")
	}
	if !strings.Contains(ctx, "Churn spike") {
		t.Error("should contain alert reference")
	}
	if !strings.Contains(ctx, "仅供参考") {
		t.Error("should contain disclaimer")
	}
}

func TestBuildIndustryContext_NilTemplate(t *testing.T) {
	ctx := BuildIndustryContext(nil)
	if ctx != "" {
		t.Errorf("expected empty string for nil template, got %q", ctx)
	}
}

func TestBuildPainPointsContext_WithTemplate(t *testing.T) {
	tmpl := &IndustryTemplate{
		Name: "科技/SaaS",
		CommonPainPoints: []string{
			"招聘困难",
			"技术债务",
		},
	}

	ctx := BuildPainPointsContext(tmpl)

	if !strings.Contains(ctx, "常见痛点") {
		t.Error("should contain pain points header")
	}
	if !strings.Contains(ctx, "招聘困难") {
		t.Error("should contain pain point")
	}
	if !strings.Contains(ctx, "主动询问") {
		t.Error("should contain instruction to ask proactively")
	}
}

func TestBuildPainPointsContext_NilTemplate(t *testing.T) {
	ctx := BuildPainPointsContext(nil)
	if ctx != "" {
		t.Errorf("expected empty string for nil template, got %q", ctx)
	}
}

func TestGetIndustry_NotFound(t *testing.T) {
	LoadIndustries()

	tmpl := GetIndustry("nonexistent")
	if tmpl != nil {
		t.Errorf("expected nil for unknown industry ID, got %v", tmpl)
	}
}

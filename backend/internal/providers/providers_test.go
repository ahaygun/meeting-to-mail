package providers

import (
	"strings"
	"testing"

	"meeting-to-mail/internal/domain"
)

func TestSanitizeSummary_BlanksGenericOwner(t *testing.T) {
	in := domain.SummaryContent{
		ActionItems: []domain.ActionItem{
			{Task: "İş A", Owner: "gündeminde tutacak kişi", Due: ""},
			{Task: "İş B", Owner: "görevli kişi", Due: ""},
			{Task: "İş C", Owner: "Ali Yılmaz", Due: "gelecek hafta"},
			{Task: "İş D", Owner: "Muhasebe departmanı sorumlusu ve tüm alt ekibi birlikte", Due: ""}, // >30 rune
		},
	}
	out := SanitizeSummary(in)

	if out.ActionItems[0].Owner != "" {
		t.Errorf("jenerik sahip boşaltılmalıydı, gelen: %q", out.ActionItems[0].Owner)
	}
	if out.ActionItems[1].Owner != "" {
		t.Errorf("'görevli kişi' boşaltılmalıydı, gelen: %q", out.ActionItems[1].Owner)
	}
	if out.ActionItems[2].Owner != "Ali Yılmaz" {
		t.Errorf("gerçek isim korunmalıydı, gelen: %q", out.ActionItems[2].Owner)
	}
	if out.ActionItems[2].Due != "gelecek hafta" {
		t.Errorf("gerçek tarih korunmalıydı, gelen: %q", out.ActionItems[2].Due)
	}
	if out.ActionItems[3].Owner != "" {
		t.Errorf("çok uzun sahip (cümle) boşaltılmalıydı, gelen: %q", out.ActionItems[3].Owner)
	}
}

func TestSanitizeSummary_BlanksGenericDue(t *testing.T) {
	in := domain.SummaryContent{
		ActionItems: []domain.ActionItem{
			{Task: "İş A", Due: "belirlenecek"},
			{Task: "İş B", Due: "15 Mart"},
		},
	}
	out := SanitizeSummary(in)
	if out.ActionItems[0].Due != "" {
		t.Errorf("belirsiz tarih boşaltılmalıydı, gelen: %q", out.ActionItems[0].Due)
	}
	if out.ActionItems[1].Due != "15 Mart" {
		t.Errorf("gerçek tarih korunmalıydı, gelen: %q", out.ActionItems[1].Due)
	}
}

func TestSanitizeSummary_ReplacesYaratmak(t *testing.T) {
	in := domain.SummaryContent{
		Headline:  "Manevi ortam yaratma planı",
		KeyPoints: []string{"Yeni fırsatlar yaratmak"},
		Decisions: []string{"Bir havuz yaratılması kararlaştırıldı"}, // pasif
		ActionItems: []domain.ActionItem{
			{Task: "Sistem yaratmak"},
		},
	}
	out := SanitizeSummary(in)

	cases := map[string]string{
		"headline":  out.Headline,
		"key_point": out.KeyPoints[0],
		"decision":  out.Decisions[0],
		"action":    out.ActionItems[0].Task,
	}
	for name, v := range cases {
		if strings.Contains(strings.ToLower(v), "yarat") {
			t.Errorf("%s hâlâ 'yarat' içeriyor: %q", name, v)
		}
	}
	if out.Headline != "Manevi ortam oluşturma planı" {
		t.Errorf("headline beklenmedik: %q", out.Headline)
	}
	if out.Decisions[0] != "Bir havuz oluşturulması kararlaştırıldı" {
		t.Errorf("pasif çekim yanlış: %q", out.Decisions[0])
	}
}

func TestNewCorrections_And_Apply(t *testing.T) {
	r := NewCorrections("iyitim=>iyilik; resale-i nur => Risale-i Nur")
	if r == nil {
		t.Fatal("replacer nil olmamalıydı")
	}
	in := domain.SummaryContent{
		Headline:  "Kamu iyitim kampanyası",
		KeyPoints: []string{"resale-i nur dersleri"},
		ActionItems: []domain.ActionItem{
			{Task: "iyitim projesi", Owner: ""},
		},
	}
	out := ApplyCorrectionsToSummary(r, in)
	if out.Headline != "Kamu iyilik kampanyası" {
		t.Errorf("headline düzeltilmedi: %q", out.Headline)
	}
	if out.KeyPoints[0] != "Risale-i Nur dersleri" {
		t.Errorf("key_point düzeltilmedi: %q", out.KeyPoints[0])
	}
	if out.ActionItems[0].Task != "iyilik projesi" {
		t.Errorf("task düzeltilmedi: %q", out.ActionItems[0].Task)
	}
}

func TestNewCorrections_EmptyAndNilSafe(t *testing.T) {
	if r := NewCorrections(""); r != nil {
		t.Error("boş spec için nil beklenirdi")
	}
	if r := NewCorrections("  ;  bozuk-satır  "); r != nil {
		t.Error("geçersiz spec için nil beklenirdi")
	}
	// nil replacer güvenli olmalı
	in := domain.SummaryContent{Headline: "değişmez"}
	out := ApplyCorrectionsToSummary(nil, in)
	if out.Headline != "değişmez" {
		t.Errorf("nil replacer içeriği değiştirmemeliydi: %q", out.Headline)
	}
}

func TestStripJSONFences(t *testing.T) {
	cases := []struct{ in, want string }{
		{"```json\n{\"a\":1}\n```", `{"a":1}`},
		{"```\n{\"a\":1}\n```", `{"a":1}`},
		{`{"a":1}`, `{"a":1}`},
		{"Şöyle bir cevap: {\"a\":1} umarım olur", `{"a":1}`},
		{"  {\"nested\":{\"b\":2}}  ", `{"nested":{"b":2}}`},
	}
	for _, c := range cases {
		if got := stripJSONFences(c.in); got != c.want {
			t.Errorf("stripJSONFences(%q) = %q, beklenen %q", c.in, got, c.want)
		}
	}
}

func TestRenderText(t *testing.T) {
	c := domain.SummaryContent{
		Headline:    "Kısa özet",
		KeyPoints:   []string{"Madde bir"},
		Decisions:   []string{"Karar bir"},
		ActionItems: []domain.ActionItem{{Task: "İş bir", Owner: "Ali", Due: "yarın"}},
	}
	out := RenderText("Haftalık Toplantı", c)
	for _, must := range []string{"Haftalık Toplantı", "Kısa özet", "Madde bir", "Karar bir", "İş bir", "Ali"} {
		if !strings.Contains(out, must) {
			t.Errorf("render çıktısı %q içermeli:\n%s", must, out)
		}
	}
}

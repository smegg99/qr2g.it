// core/verifier/shares_test.go

package verifier

import (
	"reflect"
	"testing"

	"smegg.me/qr2g.it/core/models"
)

func TestSharesKeepTheLargestSix(t *testing.T) {
	got := Shares(map[string]int64{"Go": 700, "Vue": 200, "CSS": 40, "Shell": 20, "Makefile": 20, "HTML": 10, "Dockerfile": 10, "Empty": 0})
	want := []models.LanguageShare{{Name: "Go", Share: 0.7}, {Name: "Vue", Share: 0.2}, {Name: "CSS", Share: 0.04}, {Name: "Makefile", Share: 0.02}, {Name: "Shell", Share: 0.02}, {Name: "Dockerfile", Share: 0.01}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("shares = %v", got)
	}
	if len(Shares(nil)) != 0 {
		t.Fatal("no languages gave shares")
	}
}

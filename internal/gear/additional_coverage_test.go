package gear

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
)

func TestGearName_CustomFallback(t *testing.T) {
	if got := gearName(gearRaw{CustomMakeModel: " Custom "}); got != "Custom" {
		t.Fatalf("gearName=%q", got)
	}
}

func TestGearAPIs_ValidationAndRequestErrors(t *testing.T) {
	bad := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "bad", http.StatusBadRequest)
	})
	if _, err := List(context.Background(), bad, 1); err == nil {
		t.Fatal("expected list error")
	}
	if _, err := GetStats(context.Background(), bad, "u1"); err == nil {
		t.Fatal("expected stats error")
	}
	if _, err := Get(context.Background(), bad, 1, "u1"); err == nil {
		t.Fatal("expected get resolve error")
	}
	gears := WithStats(context.Background(), bad, []Gear{{UUID: "u1"}})
	if gears[0].TotalMeters != nil || gears[0].Activities != nil {
		t.Fatalf("failed stats should be skipped: %#v", gears[0])
	}
	if _, err := Create(context.Background(), bad, 1, CreateOptions{Name: "x"}); err == nil {
		t.Fatal("expected create error")
	}
	if _, err := SetStatus(context.Background(), bad, 1, " ", "active"); err == nil {
		t.Fatal("expected blank status uuid error")
	}
	if _, err := SetStatus(context.Background(), bad, 1, "u1", "active"); err == nil {
		t.Fatal("expected status list error")
	}
	if err := Link(context.Background(), bad, " ", 1); err == nil {
		t.Fatal("expected blank link uuid error")
	}
	if err := Unlink(context.Background(), bad, " ", 1); err == nil {
		t.Fatal("expected blank unlink uuid error")
	}
	if _, err := ForActivity(context.Background(), bad, 1); err == nil {
		t.Fatal("expected for-activity error")
	}
	if _, err := Resolve(context.Background(), bad, 1, "x"); err == nil {
		t.Fatal("expected resolve list error")
	}
	if err := SetDefault(context.Background(), bad, " ", 1, true); err == nil {
		t.Fatal("expected default blank uuid error")
	}
}

func TestSetStatus_ActiveAndPutError(t *testing.T) {
	var putBody map[string]any
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			_, _ = w.Write([]byte(`[{"uuid":"u1","dateEnd":"old"}]`))
			return
		}
		if err := json.NewDecoder(r.Body).Decode(&putBody); err != nil {
			t.Fatal(err)
		}
		http.Error(w, "bad", http.StatusBadRequest)
	})
	_, err := SetStatus(context.Background(), c, 1, "u1", "active")
	if err == nil {
		t.Fatal("expected put error")
	}
	if value, ok := putBody["dateEnd"]; !ok || value != nil {
		t.Fatalf("active status should clear dateEnd: %#v", putBody)
	}
}

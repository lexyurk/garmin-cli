package activities

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/lexyurk/garmin-cli/internal/auth"
)

type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) { return 0, errors.New("write failed") }

func TestList_EdgeCasesAndRequestErrors(t *testing.T) {
	c := manageTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "bad request", http.StatusBadRequest)
	})
	for _, tc := range []struct {
		name          string
		after, before string
	}{
		{"bad after", "not-a-date", ""},
		{"bad before", "", "not-a-date"},
		{"reversed", "2026-02-17", "2026-02-16"},
		{"request error", "", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := List(context.Background(), c, 2, tc.after, tc.before, ""); err == nil {
				t.Fatal("expected error")
			}
		})
	}

	empty := manageTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, `[]`)
	})
	got, err := List(context.Background(), empty, 2, "", "", "")
	if err != nil || len(got) != 0 {
		t.Fatalf("List empty = %#v, %v", got, err)
	}
}

func TestList_StopsAtOldestDateAndFiltersPage(t *testing.T) {
	calls := 0
	c := manageTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		_, _ = io.WriteString(w, `[
			{"activityId":1,"startTimeLocal":"2026-02-18 10:00:00","activityType":{"typeKey":"cycling"}},
			{"activityId":2,"startTimeLocal":"2026-02-14 10:00:00","activityType":{"typeKey":"running"}}
		]`)
	})
	got, err := List(context.Background(), c, 2, "2026-02-15", "2026-02-20", "running")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 || calls != 1 {
		t.Fatalf("got=%#v calls=%d", got, calls)
	}
}

func TestListByGearAndLatest_EdgeCases(t *testing.T) {
	bad := manageTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "bad request", http.StatusBadRequest)
	})
	if _, err := ListByGear(context.Background(), bad, " ", 1); err == nil {
		t.Fatal("expected blank gear error")
	}
	if _, err := ListByGear(context.Background(), bad, "uuid", 0); err == nil {
		t.Fatal("expected request error after defaulting limit")
	}
	if _, err := Latest(context.Background(), bad); err == nil {
		t.Fatal("expected latest request error")
	}

	empty := manageTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, `[]`)
	})
	if _, err := Latest(context.Background(), empty); err == nil || !strings.Contains(err.Error(), "no activities") {
		t.Fatalf("unexpected latest error: %v", err)
	}

	one := manageTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, `[{"activityId":9,"activityName":"Latest"}]`)
	})
	latest, err := Latest(context.Background(), one)
	if err != nil || latest.ID != 9 {
		t.Fatalf("latest=%#v err=%v", latest, err)
	}
}

func TestActivityAPIs_PropagateErrorsAndRawWeather(t *testing.T) {
	bad := manageTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "bad request", http.StatusBadRequest)
	})
	if _, err := GetRaw(context.Background(), bad, 1); err == nil {
		t.Fatal("expected GetRaw error")
	}
	if _, err := GetActivityTypes(context.Background(), bad); err == nil {
		t.Fatal("expected GetActivityTypes error")
	}
	if _, err := GetWeatherRaw(context.Background(), bad, 1); err == nil {
		t.Fatal("expected GetWeatherRaw error")
	}
	if _, err := GetWeather(context.Background(), bad, 1); err == nil {
		t.Fatal("expected GetWeather error")
	}

	rawClient := manageTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, `{"temp":12,"custom":"kept"}`)
	})
	raw, err := GetWeatherRaw(context.Background(), rawClient, 42)
	if err != nil || raw["custom"] != "kept" {
		t.Fatalf("raw=%#v err=%v", raw, err)
	}
}

func TestManageAPIs_AllUpdateFieldsAndErrors(t *testing.T) {
	typesClient := manageTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, `[{"typeId":1,"typeKey":"running","parentTypeId":17}]`)
	})
	types, err := GetActivityTypes(context.Background(), typesClient)
	if err != nil || len(types) != 1 {
		t.Fatalf("types=%#v err=%v", types, err)
	}

	var body string
	ok := manageTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		data, _ := io.ReadAll(r.Body)
		body = string(data)
		w.WriteHeader(http.StatusNoContent)
	})
	description := "desc"
	typ := ActivityType{TypeID: 1, TypeKey: "running", ParentTypeID: 17}
	if err := Update(context.Background(), ok, 7, UpdateOptions{Description: &description, Type: &typ}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(body, `"description":"desc"`) || !strings.Contains(body, `"activityTypeDTO"`) {
		t.Fatalf("unexpected update body: %s", body)
	}

	bad := manageTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "bad request", http.StatusBadRequest)
	})
	name := "x"
	if err := Update(context.Background(), bad, 1, UpdateOptions{Name: &name}); err == nil {
		t.Fatal("expected update error")
	}
}

func TestExport_AllTypesAuthAndWriterErrors(t *testing.T) {
	for _, typ := range []ExportType{ExportTCX, ExportOriginal} {
		t.Run(string(typ), func(t *testing.T) {
			c := manageTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				if !strings.Contains(r.URL.Path, "/"+string(typ)+"/") {
					t.Fatalf("path=%s", r.URL.Path)
				}
				_, _ = io.WriteString(w, "data")
			})
			var out strings.Builder
			if err := Export(context.Background(), c, 1, typ, &out); err != nil || out.String() != "data" {
				t.Fatalf("out=%q err=%v", out.String(), err)
			}
		})
	}

	unauthorized := manageTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "expired", http.StatusUnauthorized)
	})
	err := Export(context.Background(), unauthorized, 1, ExportGPX, io.Discard)
	if !errors.Is(err, auth.ErrNotAuthenticated) {
		t.Fatalf("expected auth error, got %v", err)
	}

	writable := manageTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, "data")
	})
	if err := Export(context.Background(), writable, 1, ExportGPX, failingWriter{}); err == nil {
		t.Fatal("expected writer error")
	}
}

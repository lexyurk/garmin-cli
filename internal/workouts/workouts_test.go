package workouts

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/lexyurk/garmin-cli/internal/auth"
	"github.com/lexyurk/garmin-cli/internal/client"
)

func testClient(t *testing.T, handler http.HandlerFunc) *client.Client {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	sess := &auth.Session{
		OAuth1: auth.OAuth1Token{OAuthToken: "o1", OAuthTokenSecret: "o1s"},
		OAuth2: auth.OAuth2Token{TokenType: "bearer", AccessToken: "ok", ExpiresAt: time.Now().Add(time.Hour).Unix()},
	}
	return client.NewWithSession("ignored", "default", sess, client.Options{HTTPClient: srv.Client(), BaseURL: srv.URL})
}

func TestList_ParsesWorkouts(t *testing.T) {
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/workout-service/workouts" {
			t.Fatalf("path: %s", r.URL.Path)
		}
		if got := r.URL.Query().Get("limit"); got != "5" {
			t.Fatalf("limit: %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[
		  {"workoutId":111,"workoutName":"Intervals","sportType":{"sportTypeId":1,"sportTypeKey":"running"},"estimatedDurationInSecs":3600,"estimatedDistanceInMeters":10000}
		]`))
	})

	out, err := List(context.Background(), c, 5)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(out) != 1 || out[0].WorkoutID != 111 || out[0].Sport != "running" {
		t.Fatalf("unexpected: %#v", out)
	}
	if out[0].DurationSecs != 3600 {
		t.Fatalf("duration: %v", out[0].DurationSecs)
	}
}

func TestSummarizeAndCountSteps(t *testing.T) {
	raw := map[string]any{}
	body := `{
	  "workoutName": "Track day",
	  "description": "speed",
	  "sportType": {"sportTypeId":1,"sportTypeKey":"running"},
	  "estimatedDurationInSecs": 2400,
	  "workoutSegments": [
	    {"workoutSteps": [
	      {"stepType":{"stepTypeKey":"warmup"}},
	      {"numberOfIterations": 4, "workoutSteps": [
	        {"stepType":{"stepTypeKey":"interval"}},
	        {"stepType":{"stepTypeKey":"recovery"}}
	      ]},
	      {"stepType":{"stepTypeKey":"cooldown"}}
	    ]}
	  ]
	}`
	if err := json.Unmarshal([]byte(body), &raw); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	s := SummarizeRaw(111, raw)
	if s.Name != "Track day" || s.Sport != "running" || s.DurationSecs != 2400 {
		t.Fatalf("summary: %#v", s)
	}
	// warmup + 4*(interval+recovery) + cooldown = 1 + 8 + 1 = 10
	if got := CountSteps(raw); got != 10 {
		t.Fatalf("CountSteps: %d", got)
	}
}

func TestDelete_SendsDelete(t *testing.T) {
	var method, path string
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		method, path = r.Method, r.URL.Path
		w.WriteHeader(http.StatusNoContent)
	})
	if err := Delete(context.Background(), c, 111); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if method != http.MethodDelete || path != "/workout-service/workout/111" {
		t.Fatalf("unexpected request: %s %s", method, path)
	}
}

func TestUpdate_FetchesAndPutsName(t *testing.T) {
	var putBody map[string]any
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.Method {
		case http.MethodGet:
			_, _ = w.Write([]byte(`{"workoutName":"old","description":"d","sportType":{"sportTypeKey":"running"}}`))
		case http.MethodPut:
			_ = json.NewDecoder(r.Body).Decode(&putBody)
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Fatalf("unexpected method %s", r.Method)
		}
	})

	name := "new name"
	s, err := Update(context.Background(), c, 111, UpdateOptions{Name: &name})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if putBody["workoutName"] != "new name" {
		t.Fatalf("workoutName not set in PUT body: %#v", putBody)
	}
	// description preserved from the fetched object
	if putBody["description"] != "d" {
		t.Fatalf("description not preserved: %#v", putBody)
	}
	if s.Name != "new name" {
		t.Fatalf("summary name: %q", s.Name)
	}
}

func TestUpdate_NothingToUpdate(t *testing.T) {
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("should not call API")
	})
	if _, err := Update(context.Background(), c, 111, UpdateOptions{}); err == nil {
		t.Fatalf("expected error")
	}
}

func TestSchedule_PostsDate(t *testing.T) {
	var path string
	var body map[string]any
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		_ = json.NewDecoder(r.Body).Decode(&body)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"workoutScheduleId":9001}`))
	})

	res, err := Schedule(context.Background(), c, 111, "2026-06-01")
	if err != nil {
		t.Fatalf("Schedule: %v", err)
	}
	if path != "/workout-service/schedule/111" {
		t.Fatalf("path: %s", path)
	}
	if body["date"] != "2026-06-01" {
		t.Fatalf("date not in body: %#v", body)
	}
	if res.WorkoutScheduleID != 9001 || res.Date != "2026-06-01" {
		t.Fatalf("unexpected result: %#v", res)
	}
}

func TestSchedule_RejectsBadDate(t *testing.T) {
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("should not call API")
	})
	_, err := Schedule(context.Background(), c, 111, "06/01/2026")
	if err == nil || err.Error() != `invalid date "06/01/2026" (expected YYYY-MM-DD)` {
		t.Fatalf("error: %v", err)
	}
}

func TestExportFIT_StreamsBody(t *testing.T) {
	var path string
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		w.Header().Set("Content-Type", "application/octet-stream")
		_, _ = w.Write([]byte("FITDATA"))
	})

	var buf bytes.Buffer
	if err := ExportFIT(context.Background(), c, 111, &buf); err != nil {
		t.Fatalf("ExportFIT: %v", err)
	}
	if path != "/workout-service/workout/FIT/111" {
		t.Fatalf("path: %s", path)
	}
	if buf.String() != "FITDATA" {
		t.Fatalf("body: %q", buf.String())
	}
}

type scheduleCall struct {
	method string
	path   string
	body   string
}

func scheduleClient(t *testing.T, getStatus int, getBody string, putStatus int) (*client.Client, *[]scheduleCall) {
	t.Helper()
	var calls []scheduleCall
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		calls = append(calls, scheduleCall{method: r.Method, path: r.URL.Path, body: string(b)})
		switch r.Method {
		case http.MethodGet:
			if getStatus != http.StatusOK {
				http.Error(w, "get failed", getStatus)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(getBody))
		case http.MethodPut:
			if putStatus != http.StatusNoContent && putStatus != http.StatusOK {
				http.Error(w, "put failed", putStatus)
				return
			}
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Errorf("unexpected method %s %s", r.Method, r.URL.Path)
			http.Error(w, "method", http.StatusMethodNotAllowed)
		}
	})
	return c, &calls
}

func assertNoWrite(t *testing.T, calls []scheduleCall) {
	t.Helper()
	for _, call := range calls {
		if call.method != http.MethodGet {
			t.Fatalf("unexpected %s %s body %s", call.method, call.path, call.body)
		}
	}
}

func decodeScheduleBody(t *testing.T, body string) map[string]any {
	t.Helper()
	dec := json.NewDecoder(strings.NewReader(body))
	dec.UseNumber()
	var payload any
	if err := dec.Decode(&payload); err != nil {
		t.Fatalf("decode put body: %v\n%s", err, body)
	}
	obj, ok := payload.(map[string]any)
	if !ok {
		t.Fatalf("put body is not an object: %s", body)
	}
	return obj
}

func TestMove_RejectsBadDate(t *testing.T) {
	calls := 0
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
	})
	_, err := Move(context.Background(), c, 9001, "06/01/2026")
	if err == nil || err.Error() != `invalid date "06/01/2026" (expected YYYY-MM-DD)` {
		t.Fatalf("error: %v", err)
	}
	if calls != 0 {
		t.Fatalf("http calls: %d", calls)
	}
}

func TestMove_ReschedulesOneOccurrence(t *testing.T) {
	getBody := `{
	  "workoutScheduleId": 9001,
	  "calendarDate": "2026-10-20",
	  "date": "2026-10-01",
	  "ownerId": 7,
	  "note": "A & B",
	  "workout": {
	    "workoutId": 111,
	    "workoutName": "Intervals",
	    "description": "speed",
	    "sportType": {"sportTypeKey": "running"}
	  }
	}`
	c, calls := scheduleClient(t, http.StatusOK, getBody, http.StatusNoContent)
	res, err := Move(context.Background(), c, 9001, "2026-10-23")
	if err != nil {
		t.Fatalf("Move: %v", err)
	}
	if res.ScheduleID != 9001 || res.WorkoutID != 111 || res.PreviousDate != "2026-10-20" || res.Date != "2026-10-23" || res.Status != "moved" {
		t.Fatalf("result: %#v", res)
	}
	raw, err := json.Marshal(res)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	for _, want := range []string{`"schedule_id":9001`, `"workout_id":111`, `"previous_date":"2026-10-20"`, `"date":"2026-10-23"`, `"status":"moved"`} {
		if !strings.Contains(string(raw), want) {
			t.Fatalf("result json %s missing %s", raw, want)
		}
	}
	got := *calls
	if len(got) != 2 || got[0].method != http.MethodGet || got[1].method != http.MethodPut {
		t.Fatalf("calls: %#v", got)
	}
	for _, call := range got {
		if call.path != "/workout-service/schedule/9001" {
			t.Fatalf("path: %s", call.path)
		}
	}
	put := got[1].body
	if !strings.Contains(put, "A & B") || strings.Contains(put, `\u0026`) {
		t.Fatalf("note was escaped: %s", put)
	}
	if strings.Contains(put, "workoutName") || strings.Contains(put, "Intervals") || strings.Contains(put, "sportType") {
		t.Fatalf("workout was not shrunk: %s", put)
	}
	obj := decodeScheduleBody(t, put)
	if obj["calendarDate"] != "2026-10-23" || obj["date"] != "2026-10-23" || obj["note"] != "A & B" {
		t.Fatalf("put fields: %#v", obj)
	}
	if obj["ownerId"] != json.Number("7") || obj["workoutScheduleId"] != json.Number("9001") {
		t.Fatalf("siblings: %#v", obj)
	}
	workout, ok := obj["workout"].(map[string]any)
	if !ok || len(workout) != 1 || workout["workoutId"] != json.Number("111") {
		t.Fatalf("workout: %#v", obj["workout"])
	}
}

func TestMove_PreservesLargeWorkoutID(t *testing.T) {
	c, calls := scheduleClient(t, http.StatusOK, `{"calendarDate":"2026-10-20","date":"2026-10-20","workout":{"workoutId":9007199254740993,"workoutName":"big"}}`, http.StatusNoContent)
	res, err := Move(context.Background(), c, 9001, "2026-10-23")
	if err != nil {
		t.Fatalf("Move: %v", err)
	}
	if res.WorkoutID != 9007199254740993 {
		t.Fatalf("workout id: %d", res.WorkoutID)
	}
	put := (*calls)[1].body
	if !strings.Contains(put, "9007199254740993") || strings.Contains(put, "e+") || strings.Contains(put, "E+") {
		t.Fatalf("put body: %s", put)
	}
	obj := decodeScheduleBody(t, put)
	workout, ok := obj["workout"].(map[string]any)
	if !ok || len(workout) != 1 || workout["workoutId"] != json.Number("9007199254740993") {
		t.Fatalf("workout: %#v", obj["workout"])
	}
	if obj["date"] != "2026-10-23" {
		t.Fatalf("date: %#v", obj["date"])
	}
}

func TestMove_MissingDateKeyIsNotAdded(t *testing.T) {
	c, calls := scheduleClient(t, http.StatusOK, `{"calendarDate":"2026-10-20","workout":{"workoutId":111,"extra":true}}`, http.StatusNoContent)
	res, err := Move(context.Background(), c, 9001, "2026-10-23")
	if err != nil {
		t.Fatalf("Move: %v", err)
	}
	if res.PreviousDate != "2026-10-20" {
		t.Fatalf("previous: %#v", res)
	}
	obj := decodeScheduleBody(t, (*calls)[1].body)
	if _, ok := obj["date"]; ok {
		t.Fatalf("date key added: %#v", obj)
	}
	if obj["calendarDate"] != "2026-10-23" {
		t.Fatalf("calendarDate: %#v", obj["calendarDate"])
	}
}

func TestMove_PreviousDateFallsBackToDate(t *testing.T) {
	c, calls := scheduleClient(t, http.StatusOK, `{"date":"2026-09-01","workout":{"workoutId":5,"extra":true}}`, http.StatusNoContent)
	res, err := Move(context.Background(), c, 9001, "2026-10-23")
	if err != nil {
		t.Fatalf("Move: %v", err)
	}
	if res.PreviousDate != "2026-09-01" || res.WorkoutID != 5 {
		t.Fatalf("result: %#v", res)
	}
	obj := decodeScheduleBody(t, (*calls)[1].body)
	if obj["date"] != "2026-10-23" || obj["calendarDate"] != "2026-10-23" {
		t.Fatalf("dates: %#v", obj)
	}
}

func TestMove_OmitsPreviousDateWithoutStringDate(t *testing.T) {
	c, calls := scheduleClient(t, http.StatusOK, `{"calendarDate":12,"workout":{"workoutId":5}}`, http.StatusNoContent)
	res, err := Move(context.Background(), c, 9001, "2026-10-23")
	if err != nil {
		t.Fatalf("Move: %v", err)
	}
	if res.PreviousDate != "" || res.WorkoutID != 5 {
		t.Fatalf("result: %#v", res)
	}
	raw, err := json.Marshal(res)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if strings.Contains(string(raw), "previous_date") {
		t.Fatalf("json included previous_date: %s", raw)
	}
	obj := decodeScheduleBody(t, (*calls)[1].body)
	if obj["calendarDate"] != "2026-10-23" {
		t.Fatalf("calendarDate: %#v", obj["calendarDate"])
	}
}

func TestMove_AbsentWorkoutSetsNull(t *testing.T) {
	cases := []struct {
		name string
		body string
	}{
		{"absent", `{"calendarDate":"2026-10-20","ownerId":4,"note":"A & B"}`},
		{"null", `{"calendarDate":"2026-10-20","workout":null,"ownerId":4,"note":"A & B"}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c, calls := scheduleClient(t, http.StatusOK, tc.body, http.StatusNoContent)
			res, err := Move(context.Background(), c, 9001, "2026-10-23")
			if err != nil {
				t.Fatalf("Move: %v", err)
			}
			if res.WorkoutID != 0 || res.PreviousDate != "2026-10-20" || res.Status != "moved" || res.Date != "2026-10-23" {
				t.Fatalf("result: %#v", res)
			}
			rawResult, err := json.Marshal(res)
			if err != nil {
				t.Fatalf("marshal: %v", err)
			}
			if strings.Contains(string(rawResult), "workout_id") {
				t.Fatalf("json included workout_id: %s", rawResult)
			}
			put := (*calls)[1].body
			if !strings.Contains(put, `"workout":null`) || strings.Contains(put, `\u0026`) || !strings.Contains(put, "A & B") {
				t.Fatalf("put body: %s", put)
			}
			obj := decodeScheduleBody(t, put)
			if obj["workout"] != nil || obj["calendarDate"] != "2026-10-23" || obj["note"] != "A & B" {
				t.Fatalf("put object: %#v", obj)
			}
			if _, ok := obj["date"]; ok {
				t.Fatalf("date key added: %#v", obj)
			}
			if obj["ownerId"] != json.Number("4") {
				t.Fatalf("ownerId: %#v", obj["ownerId"])
			}
		})
	}
}

func TestMove_MissingWorkoutIDDoesNotPut(t *testing.T) {
	cases := []struct {
		name string
		body string
	}{
		{"empty object", `{"calendarDate":"2026-10-20","workout":{}}`},
		{"missing workoutId", `{"calendarDate":"2026-10-20","workout":{"workoutName":"Intervals"}}`},
		{"string workoutId", `{"calendarDate":"2026-10-20","workout":{"workoutId":"111"}}`},
		{"float workoutId", `{"calendarDate":"2026-10-20","workout":{"workoutId":1.5}}`},
		{"null workoutId", `{"calendarDate":"2026-10-20","workout":{"workoutId":null}}`},
		{"overflow workoutId", `{"calendarDate":"2026-10-20","workout":{"workoutId":9223372036854775808}}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c, calls := scheduleClient(t, http.StatusOK, tc.body, http.StatusNoContent)
			_, err := Move(context.Background(), c, 9001, "2026-10-23")
			if err == nil || !strings.Contains(err.Error(), "workoutId") || !strings.Contains(err.Error(), "schedule 9001:") {
				t.Fatalf("error: %v", err)
			}
			assertNoWrite(t, *calls)
		})
	}
}

func TestMove_WorkoutNotObjectDoesNotPut(t *testing.T) {
	c, calls := scheduleClient(t, http.StatusOK, `{"calendarDate":"2026-10-20","workout":["nope"]}`, http.StatusNoContent)
	_, err := Move(context.Background(), c, 9001, "2026-10-23")
	if err == nil || !strings.Contains(err.Error(), "expected a JSON object") || !strings.Contains(err.Error(), "schedule 9001:") {
		t.Fatalf("error: %v", err)
	}
	assertNoWrite(t, *calls)
}

func TestMove_NonObjectBodyDoesNotPut(t *testing.T) {
	c, calls := scheduleClient(t, http.StatusOK, `[1,2]`, http.StatusNoContent)
	_, err := Move(context.Background(), c, 9001, "2026-10-23")
	if err == nil || !strings.Contains(err.Error(), "expected a JSON object") || !strings.Contains(err.Error(), "schedule 9001:") {
		t.Fatalf("error: %v", err)
	}
	assertNoWrite(t, *calls)
}

func TestMove_GetFailureDoesNotPut(t *testing.T) {
	c, calls := scheduleClient(t, http.StatusBadRequest, `{"workout":{"workoutId":111}}`, http.StatusNoContent)
	_, err := Move(context.Background(), c, 9001, "2026-10-23")
	if err == nil || !strings.Contains(err.Error(), "400") {
		t.Fatalf("error: %v", err)
	}
	assertNoWrite(t, *calls)
	if len(*calls) == 0 {
		t.Fatal("expected a GET")
	}
}

func TestMove_PutFailure(t *testing.T) {
	c, calls := scheduleClient(t, http.StatusOK, `{"calendarDate":"2026-10-20","workout":{"workoutId":111}}`, http.StatusBadRequest)
	_, err := Move(context.Background(), c, 9001, "2026-10-23")
	if err == nil || !strings.Contains(err.Error(), "400") {
		t.Fatalf("error: %v", err)
	}
	puts := 0
	for _, call := range *calls {
		if call.method == http.MethodPut {
			puts++
			if call.path != "/workout-service/schedule/9001" || !strings.Contains(call.body, `"calendarDate":"2026-10-23"`) {
				t.Fatalf("put: %#v", call)
			}
		}
	}
	if puts != 1 {
		t.Fatalf("puts: %d", puts)
	}
}

func TestMove_PutAuthFailure(t *testing.T) {
	c, calls := scheduleClient(t, http.StatusOK, `{"calendarDate":"2026-10-20","workout":{"workoutId":111}}`, http.StatusUnauthorized)
	_, err := Move(context.Background(), c, 9001, "2026-10-23")
	if !errors.Is(err, auth.ErrNotAuthenticated) {
		t.Fatalf("error: %v", err)
	}
	puts := 0
	for _, call := range *calls {
		if call.method == http.MethodPut {
			puts++
		}
	}
	if puts != 1 {
		t.Fatalf("puts: %d", puts)
	}
}

func TestUnschedule_SendsDelete(t *testing.T) {
	var method, path string
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		method, path = r.Method, r.URL.Path
		w.WriteHeader(http.StatusNoContent)
	})
	if err := Unschedule(context.Background(), c, 9001); err != nil {
		t.Fatalf("Unschedule: %v", err)
	}
	if method != http.MethodDelete || path != "/workout-service/schedule/9001" {
		t.Fatalf("unexpected request: %s %s", method, path)
	}
}

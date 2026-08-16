// Package courses manages Garmin Connect navigation courses.
package courses

import (
	"context"
	"fmt"
	"io"
	"math"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/lexyurk/garmin-cli/internal/auth"
	"github.com/lexyurk/garmin-cli/internal/client"
)

const earthRadiusMeters = 6371000.0

var activityTypeIDs = map[string]int{
	"running": 1, "cycling": 2, "hiking": 3, "gravel_cycling": 4,
	"mountain_biking": 5, "trail_running": 6, "walking": 9, "road_biking": 10,
}

type GeoPoint struct {
	Latitude  float64  `json:"latitude"`
	Longitude float64  `json:"longitude"`
	Elevation *float64 `json:"elevation"`
	Distance  float64  `json:"distance"`
	Timestamp any      `json:"timestamp"`
}

type CoursePoint struct {
	CoursePointID    any     `json:"coursePointId"`
	Name             string  `json:"name"`
	CoursePK         any     `json:"coursePk"`
	CoursePointType  string  `json:"coursePointType"`
	Lon              float64 `json:"lon"`
	Lat              float64 `json:"lat"`
	Distance         float64 `json:"distance"`
	Elevation        float64 `json:"elevation"`
	DerivedElevation any     `json:"derivedElevation"`
	Timestamp        int64   `json:"timestamp"`
	CreatedDate      any     `json:"createdDate"`
	ModifiedDate     any     `json:"modifiedDate"`
	UUID             any     `json:"uuid"`
	Note             any     `json:"note"`
	CutoffDuration   any     `json:"cutoffDuration"`
	RestDuration     any     `json:"restDuration"`
}

type Course struct {
	CourseID            int64         `json:"courseId"`
	CourseName          string        `json:"courseName"`
	Description         string        `json:"description"`
	DistanceMeter       float64       `json:"distanceMeter"`
	DistanceInMeters    float64       `json:"distanceInMeters"`
	ElevationGainMeter  float64       `json:"elevationGainMeter"`
	ElevationGainMeters float64       `json:"elevationGainInMeters"`
	ElevationLossMeter  float64       `json:"elevationLossMeter"`
	ElevationLossMeters float64       `json:"elevationLossInMeters"`
	ActivityTypePK      int           `json:"activityTypePk"`
	ActivityType        ActivityType  `json:"activityType"`
	Created             string        `json:"createdDateFormatted"`
	GeoPoints           []GeoPoint    `json:"geoPoints"`
	CoursePoints        []CoursePoint `json:"coursePoints"`
	CourseLines         []CourseLine  `json:"courseLines"`
}

type ActivityType struct {
	TypeKey string `json:"typeKey"`
}

type CourseLine struct {
	CourseID                 any        `json:"courseId"`
	SortOrder                int        `json:"sortOrder"`
	NumberOfPoints           int        `json:"numberOfPoints"`
	DistanceInMeters         float64    `json:"distanceInMeters"`
	Bearing                  float64    `json:"bearing"`
	Points                   []GeoPoint `json:"points"`
	CoordinateSystem         string     `json:"coordinateSystem"`
	OriginalCoordinateSystem string     `json:"originalCoordinateSystem"`
}

type Summary struct {
	ID             int64   `json:"id"`
	Name           string  `json:"name"`
	DistanceMeters float64 `json:"distance_meters"`
	ElevationGain  float64 `json:"elevation_gain_meters"`
	ElevationLoss  float64 `json:"elevation_loss_meters"`
	Activity       string  `json:"activity"`
	Created        string  `json:"created,omitempty"`
	RoutePoints    int     `json:"route_points,omitempty"`
	CoursePoints   int     `json:"course_points,omitempty"`
	URL            string  `json:"url"`
}

func summarize(c Course) Summary {
	distance := c.DistanceMeter
	if distance == 0 {
		distance = c.DistanceInMeters
	}
	gain := c.ElevationGainMeter
	if gain == 0 {
		gain = c.ElevationGainMeters
	}
	loss := c.ElevationLossMeter
	if loss == 0 {
		loss = c.ElevationLossMeters
	}
	activity := c.ActivityType.TypeKey
	if activity == "" {
		activity = ActivityTypeName(c.ActivityTypePK)
	}
	points := len(c.GeoPoints)
	if points == 0 && len(c.CourseLines) > 0 {
		points = c.CourseLines[0].NumberOfPoints
	}
	return Summary{ID: c.CourseID, Name: c.CourseName, DistanceMeters: distance, ElevationGain: gain,
		ElevationLoss: loss, Activity: activity, Created: c.Created, RoutePoints: points,
		CoursePoints: len(c.CoursePoints), URL: CourseURL(c.CourseID)}
}

func List(ctx context.Context, c *client.Client) ([]Summary, error) {
	var raw []Course
	if err := c.GetJSON(ctx, "/course-service/course", nil, &raw); err != nil {
		return nil, err
	}
	out := make([]Summary, 0, len(raw))
	for _, item := range raw {
		out = append(out, summarize(item))
	}
	return out, nil
}

func Get(ctx context.Context, c *client.Client, id int64) (Summary, error) {
	if id <= 0 {
		return Summary{}, fmt.Errorf("course id must be > 0")
	}
	var raw Course
	if err := c.GetJSON(ctx, fmt.Sprintf("/course-service/course/%d", id), nil, &raw); err != nil {
		return Summary{}, err
	}
	if raw.CourseID != id {
		return Summary{}, fmt.Errorf("course verification failed: requested %d, got %d", id, raw.CourseID)
	}
	return summarize(raw), nil
}

func Delete(ctx context.Context, c *client.Client, id int64) error {
	if id <= 0 {
		return fmt.Errorf("course id must be > 0")
	}
	return c.Delete(ctx, fmt.Sprintf("/course-service/course/%d", id), nil)
}

func Export(ctx context.Context, c *client.Client, id int64, w io.Writer) error {
	if id <= 0 {
		return fmt.Errorf("course id must be > 0")
	}
	resp, err := c.DoRaw(ctx, http.MethodGet, fmt.Sprintf("/course-service/course/gpx/%d", id), nil, nil, "", "application/gpx+xml,application/xml,*/*")
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 16<<10))
		if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
			return fmt.Errorf("%w: %s: %s", auth.ErrNotAuthenticated, resp.Status, strings.TrimSpace(string(body)))
		}
		return fmt.Errorf("garmin connectapi error: %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}
	_, err = io.Copy(w, resp.Body)
	return err
}

type PointSpec struct {
	Type, Name string
	Distance   float64
}

func ParsePointSpec(s string) (PointSpec, error) {
	parts := strings.SplitN(s, "|", 3)
	if len(parts) != 3 || strings.TrimSpace(parts[0]) == "" || strings.TrimSpace(parts[2]) == "" {
		return PointSpec{}, fmt.Errorf("invalid course point %q (expected TYPE|DISTANCE|NAME)", s)
	}
	d, err := ParseDistance(parts[1])
	if err != nil {
		return PointSpec{}, fmt.Errorf("invalid course point %q: %w", s, err)
	}
	return PointSpec{Type: strings.ToUpper(strings.TrimSpace(parts[0])), Distance: d, Name: strings.TrimSpace(parts[2])}, nil
}

func ParseDistance(s string) (float64, error) {
	s = strings.ToLower(strings.TrimSpace(s))
	mult := 1.0
	switch {
	case strings.HasSuffix(s, "km"):
		mult, s = 1000, strings.TrimSpace(strings.TrimSuffix(s, "km"))
	case strings.HasSuffix(s, "m"):
		s = strings.TrimSpace(strings.TrimSuffix(s, "m"))
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil || math.IsNaN(v) || math.IsInf(v, 0) || v < 0 {
		return 0, fmt.Errorf("distance must be a non-negative number with optional m/km suffix")
	}
	return v * mult, nil
}

func ActivityTypeID(name string) (int, error) {
	id, ok := activityTypeIDs[strings.ToLower(strings.TrimSpace(name))]
	if !ok {
		return 0, fmt.Errorf("unknown activity type %q (supported: cycling, gravel_cycling, hiking, mountain_biking, road_biking, running, trail_running, walking)", name)
	}
	return id, nil
}

func ActivityTypeName(id int) string {
	for name, candidate := range activityTypeIDs {
		if candidate == id {
			return name
		}
	}
	if id == 0 {
		return ""
	}
	return strconv.Itoa(id)
}

type ImportOptions struct {
	Filename, Name, ActivityType, Description string
	Points                                    []PointSpec
}

func Import(ctx context.Context, c *client.Client, gpx []byte, opts ImportOptions) (Summary, error) {
	if len(gpx) == 0 {
		return Summary{}, fmt.Errorf("GPX input is empty")
	}
	id, err := ActivityTypeID(opts.ActivityType)
	if err != nil {
		return Summary{}, err
	}
	var parsed Course
	if err := c.PostMultipartFile(ctx, "/course-service/course/import", "file", opts.Filename, "application/gpx+xml", gpx, &parsed); err != nil {
		return Summary{}, err
	}
	name := strings.TrimSpace(opts.Name)
	if name == "" {
		name = strings.TrimSpace(parsed.CourseName)
	}
	if name == "" {
		name = strings.TrimSuffix(filepath.Base(opts.Filename), filepath.Ext(opts.Filename))
	}
	if name == "" {
		return Summary{}, fmt.Errorf("course name is empty; pass --name")
	}
	payload, err := BuildPayload(parsed, name, id, opts.Description, opts.Points)
	if err != nil {
		return Summary{}, err
	}
	var created Course
	if err := c.PostJSON(ctx, "/course-service/course", nil, payload, &created); err != nil {
		return Summary{}, err
	}
	if created.CourseID <= 0 {
		return Summary{}, fmt.Errorf("course create response did not include a valid id")
	}
	verified, err := Get(ctx, c, created.CourseID)
	if err != nil {
		return Summary{}, fmt.Errorf("course %d was created but verification failed: %w", created.CourseID, err)
	}
	if verified.Name != name {
		return Summary{}, fmt.Errorf("course %d verification failed: expected name %q, got %q", created.CourseID, name, verified.Name)
	}
	return verified, nil
}

func CourseURL(id int64) string {
	return fmt.Sprintf("https://connect.garmin.com/modern/course/%d", id)
}

func haversine(a, b GeoPoint) float64 {
	lat1, lon1 := a.Latitude*math.Pi/180, a.Longitude*math.Pi/180
	lat2, lon2 := b.Latitude*math.Pi/180, b.Longitude*math.Pi/180
	dlat, dlon := lat2-lat1, lon2-lon1
	x := math.Sin(dlat/2)*math.Sin(dlat/2) + math.Cos(lat1)*math.Cos(lat2)*math.Sin(dlon/2)*math.Sin(dlon/2)
	return 2 * earthRadiusMeters * math.Asin(math.Sqrt(x))
}

func bearing(a, b GeoPoint) float64 {
	lat1, lat2 := a.Latitude*math.Pi/180, b.Latitude*math.Pi/180
	dlon := (b.Longitude - a.Longitude) * math.Pi / 180
	x := math.Sin(dlon) * math.Cos(lat2)
	y := math.Cos(lat1)*math.Sin(lat2) - math.Sin(lat1)*math.Cos(lat2)*math.Cos(dlon)
	v := math.Atan2(x, y) * 180 / math.Pi
	return math.Mod(v+360, 360)
}

func elevation(p GeoPoint) float64 {
	if p.Elevation == nil {
		return 0
	}
	return *p.Elevation
}

func BuildPayload(parsed Course, name string, activityTypeID int, description string, specs []PointSpec) (map[string]any, error) {
	points := append([]GeoPoint(nil), parsed.GeoPoints...)
	if len(points) < 2 {
		return nil, fmt.Errorf("parsed course has fewer than 2 geo points; GPX is empty or invalid")
	}
	total := 0.0
	minLat, maxLat, minLon, maxLon := points[0].Latitude, points[0].Latitude, points[0].Longitude, points[0].Longitude
	for i := range points {
		if i > 0 {
			total += haversine(points[i-1], points[i])
		}
		points[i].Distance = total
		if points[i].Elevation == nil {
			zero := 0.0
			points[i].Elevation = &zero
		}
		minLat, maxLat = math.Min(minLat, points[i].Latitude), math.Max(maxLat, points[i].Latitude)
		minLon, maxLon = math.Min(minLon, points[i].Longitude), math.Max(maxLon, points[i].Longitude)
	}
	cp := make([]CoursePoint, 0, len(specs))
	for _, spec := range specs {
		if math.IsNaN(spec.Distance) || math.IsInf(spec.Distance, 0) {
			return nil, fmt.Errorf("course point %q distance %v m is not finite (route length %.2f m)", spec.Name, spec.Distance, total)
		}
		if spec.Distance < 0 || spec.Distance > total {
			return nil, fmt.Errorf("course point %q distance %.2f m is outside route range [0, %.2f] m", spec.Name, spec.Distance, total)
		}
		nearest := points[0]
		best := math.Abs(spec.Distance - nearest.Distance)
		for _, p := range points[1:] {
			if d := math.Abs(spec.Distance - p.Distance); d < best {
				nearest, best = p, d
			}
		}
		cp = append(cp, CoursePoint{Name: spec.Name, CoursePointType: spec.Type, Lon: nearest.Longitude, Lat: nearest.Latitude,
			Distance: nearest.Distance, Elevation: elevation(nearest)})
	}
	start := map[string]any{"latitude": points[0].Latitude, "longitude": points[0].Longitude, "elevation": elevation(points[0]), "distance": nil, "timestamp": nil}
	bbox := map[string]any{
		"center":    map[string]float64{"latitude": (minLat + maxLat) / 2, "longitude": (minLon + maxLon) / 2},
		"lowerLeft": map[string]float64{"latitude": minLat, "longitude": minLon}, "upperRight": map[string]float64{"latitude": maxLat, "longitude": maxLon},
		"lowerLeftLatIsSet": true, "lowerLeftLongIsSet": true, "upperRightLatIsSet": true, "upperRightLongIsSet": true,
	}
	line := CourseLine{SortOrder: 1, NumberOfPoints: len(points), DistanceInMeters: total, Bearing: bearing(points[0], points[len(points)-1]),
		Points: points, CoordinateSystem: "WGS84", OriginalCoordinateSystem: "WGS84"}
	return map[string]any{
		"courseName": name, "description": nilIfEmpty(description), "openStreetMap": false, "matchedToSegments": false,
		"userProfilePk": nil, "userGroupPk": nil, "rulePK": 2, "geoRoutePk": nil, "sourceTypeId": 3, "sourcePk": nil,
		"distanceMeter": total, "elevationGainMeter": 0.0, "elevationLossMeter": 0.0, "startPoint": start,
		"coursePoints": cp, "boundingBox": bbox, "hasShareableEvent": false, "hasTurnDetectionDisabled": false,
		"activityTypePk": activityTypeID, "virtualPartnerId": nil, "includeLaps": false, "elapsedSeconds": nil,
		"speedMeterPerSecond": nil, "courseLines": []CourseLine{line}, "coordinateSystem": "WGS84",
		"targetCoordinateSystem": "WGS84", "originalCoordinateSystem": "WGS84", "consumer": nil, "elevationSource": 3,
		"hasPaceBand": false, "hasPowerGuide": false, "favorite": false, "startNote": nil, "finishNote": nil,
		"cutoffDuration": nil, "geoPoints": points,
	}, nil
}

func nilIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}

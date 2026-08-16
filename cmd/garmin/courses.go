package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	garmincourses "github.com/lexyurk/garmin-cli/internal/courses"
	"github.com/lexyurk/garmin-cli/internal/output"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var courseIsTerminal = term.IsTerminal

func NewCoursesCmd(opts *globalOptions) *cobra.Command {
	cmd := &cobra.Command{Use: "courses", Short: "Navigation course management"}
	cmd.AddCommand(newCoursesListCmd(opts), newCoursesGetCmd(opts), newCoursesImportCmd(opts), newCoursesExportCmd(opts), newCoursesDeleteCmd(opts))
	return cmd
}

func newCoursesListCmd(opts *globalOptions) *cobra.Command {
	return &cobra.Command{
		Use: "list", Short: "List saved courses",
		RunE: func(cmd *cobra.Command, _ []string) error {
			c, err := newAuthedClient(cmd, opts)
			if err != nil {
				return err
			}
			items, err := garmincourses.List(cmd.Context(), c)
			if err != nil {
				return handleAuthedErrorTo(cmd.ErrOrStderr(), opts, err)
			}
			if opts.Format == "json" {
				return output.JSONTo(cmd.OutOrStdout(), items)
			}
			rows := make([][]string, 0, len(items))
			for _, item := range items {
				rows = append(rows, courseRow(item))
			}
			return renderTableTo(cmd.OutOrStdout(), opts.Format, []string{"id", "name", "activity", "dist_km", "gain_m", "loss_m"}, rows)
		},
	}
}

func newCoursesGetCmd(opts *globalOptions) *cobra.Command {
	return &cobra.Command{
		Use: "get [course-id]", Short: "Get course details", Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := parseCourseID(args[0])
			if err != nil {
				return err
			}
			c, err := newAuthedClient(cmd, opts)
			if err != nil {
				return err
			}
			item, err := garmincourses.Get(cmd.Context(), c, id)
			if err != nil {
				return handleAuthedErrorTo(cmd.ErrOrStderr(), opts, err)
			}
			return renderCourse(cmd, opts, "Course", item)
		},
	}
}

func newCoursesImportCmd(opts *globalOptions) *cobra.Command {
	var name, activityType, description string
	var rawPoints []string
	var replaceID int64
	var force bool
	cmd := &cobra.Command{
		Use: "import [file.gpx|-]", Short: "Import a GPX file as a course", Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if replaceID < 0 {
				return fmt.Errorf("--replace must be a positive course id")
			}
			input, filename, err := readGPX(cmd, args[0])
			if err != nil {
				return err
			}
			specs := make([]garmincourses.PointSpec, 0, len(rawPoints))
			for _, raw := range rawPoints {
				spec, err := garmincourses.ParsePointSpec(raw)
				if err != nil {
					return err
				}
				specs = append(specs, spec)
			}
			c, err := newAuthedClient(cmd, opts)
			if err != nil {
				return err
			}
			created, err := garmincourses.Import(cmd.Context(), c, input, garmincourses.ImportOptions{
				Filename: filename, Name: name, ActivityType: activityType, Description: description, Points: specs,
			})
			if err != nil {
				return handleAuthedErrorTo(cmd.ErrOrStderr(), opts, err)
			}
			if replaceID > 0 {
				if created.ID == replaceID {
					return fmt.Errorf("refusing to replace course %d with the same id", replaceID)
				}
				prompt := fmt.Sprintf("Course %d was created and verified. Delete old course %d?", created.ID, replaceID)
				if !confirmDestructive(cmd, prompt, force) {
					return fmt.Errorf("replacement not completed: new course %d is verified and kept; old course %d was not deleted (pass --force non-interactively)", created.ID, replaceID)
				}
				if err := garmincourses.Delete(cmd.Context(), c, replaceID); err != nil {
					return handleAuthedErrorTo(cmd.ErrOrStderr(), opts, fmt.Errorf("new course %d is verified, but deleting old course %d failed: %w", created.ID, replaceID, err))
				}
			}
			return renderCourse(cmd, opts, "Course imported", created)
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Course name (defaults to the GPX/import name)")
	cmd.Flags().StringVar(&activityType, "activity-type", "running", "Activity type")
	cmd.Flags().StringVar(&description, "description", "", "Course description")
	cmd.Flags().StringArrayVar(&rawPoints, "point", nil, "Course point TYPE|DISTANCE|NAME; distance accepts m or km (repeatable)")
	cmd.Flags().Int64Var(&replaceID, "replace", 0, "Delete this old course only after the new course verifies")
	cmd.Flags().BoolVar(&force, "force", false, "Skip replacement delete confirmation")
	return cmd
}

func newCoursesExportCmd(opts *globalOptions) *cobra.Command {
	var outPath string
	var force bool
	cmd := &cobra.Command{
		Use: "export [course-id]", Short: "Export a course as GPX", Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := parseCourseID(args[0])
			if err != nil {
				return err
			}
			w := cmd.OutOrStdout()
			if strings.TrimSpace(outPath) == "" {
				if f, ok := w.(*os.File); ok && courseIsTerminal(int(f.Fd())) {
					return fmt.Errorf("refusing to write GPX to terminal; use --out or redirect stdout")
				}
			} else if !force {
				if _, err := os.Stat(outPath); err == nil {
					return fmt.Errorf("output file already exists: %s (pass --force to overwrite)", outPath)
				} else if !os.IsNotExist(err) {
					return err
				}
			}
			c, err := newAuthedClient(cmd, opts)
			if err != nil {
				return err
			}
			var data strings.Builder
			if err := garmincourses.Export(cmd.Context(), c, id, &data); err != nil {
				return handleAuthedErrorTo(cmd.ErrOrStderr(), opts, err)
			}
			if strings.TrimSpace(outPath) == "" {
				_, err = io.WriteString(w, data.String())
				return err
			}
			if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
				return err
			}
			flags := os.O_CREATE | os.O_WRONLY | os.O_TRUNC
			if !force {
				flags = os.O_CREATE | os.O_WRONLY | os.O_EXCL
			}
			f, err := os.OpenFile(outPath, flags, 0o644)
			if err != nil {
				return err
			}
			if _, err = io.WriteString(f, data.String()); err != nil {
				_ = f.Close()
				return err
			}
			if err = f.Close(); err != nil {
				return err
			}
			if !opts.Quiet {
				_, _ = fmt.Fprintln(cmd.ErrOrStderr(), "downloaded")
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&outPath, "out", "", "Write to file instead of stdout")
	cmd.Flags().BoolVar(&force, "force", false, "Overwrite output file if it exists")
	return cmd
}

func newCoursesDeleteCmd(opts *globalOptions) *cobra.Command {
	var force bool
	cmd := &cobra.Command{
		Use: "delete [course-id]", Short: "Delete a course (irreversible)", Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := parseCourseID(args[0])
			if err != nil {
				return err
			}
			if !confirmDestructive(cmd, fmt.Sprintf("Delete course %d? This cannot be undone.", id), force) {
				return fmt.Errorf("aborted: pass --force to delete non-interactively")
			}
			c, err := newAuthedClient(cmd, opts)
			if err != nil {
				return err
			}
			if err := garmincourses.Delete(cmd.Context(), c, id); err != nil {
				return handleAuthedErrorTo(cmd.ErrOrStderr(), opts, err)
			}
			if opts.Format == "json" {
				return output.JSONTo(cmd.OutOrStdout(), courseDeleteJSON{ID: id, Status: "deleted"})
			}
			return renderKVTo(cmd.OutOrStdout(), opts.Format, "Course deleted", map[string]string{"id": strconv.FormatInt(id, 10), "status": "deleted"})
		},
	}
	cmd.Flags().BoolVar(&force, "force", false, "Skip confirmation prompt")
	return cmd
}

type courseDeleteJSON struct {
	ID     int64  `json:"id"`
	Status string `json:"status"`
}

func parseCourseID(raw string) (int64, error) {
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id <= 0 {
		return 0, fmt.Errorf("invalid course id %q", raw)
	}
	return id, nil
}

func readGPX(cmd *cobra.Command, path string) ([]byte, string, error) {
	if path == "-" {
		data, err := io.ReadAll(cmd.InOrStdin())
		return data, "stdin.gpx", err
	}
	if strings.ToLower(filepath.Ext(path)) != ".gpx" {
		return nil, "", fmt.Errorf("course import requires a .gpx file or - for stdin")
	}
	data, err := os.ReadFile(path)
	return data, filepath.Base(path), err
}

func courseRow(item garmincourses.Summary) []string {
	return []string{strconv.FormatInt(item.ID, 10), item.Name, orDash(item.Activity), formatDistanceKM(item.DistanceMeters),
		fmt.Sprintf("%.0f", item.ElevationGain), fmt.Sprintf("%.0f", item.ElevationLoss)}
}

func renderCourse(cmd *cobra.Command, opts *globalOptions, title string, item garmincourses.Summary) error {
	if opts.Format == "json" {
		return output.JSONTo(cmd.OutOrStdout(), item)
	}
	return renderKVTo(cmd.OutOrStdout(), opts.Format, title, map[string]string{
		"id": strconv.FormatInt(item.ID, 10), "name": item.Name, "activity": orDash(item.Activity),
		"distance": formatDistanceKM(item.DistanceMeters) + " km", "elevation_gain": fmt.Sprintf("%.0f m", item.ElevationGain),
		"elevation_loss": fmt.Sprintf("%.0f m", item.ElevationLoss), "route_points": strconv.Itoa(item.RoutePoints),
		"course_points": strconv.Itoa(item.CoursePoints), "url": item.URL,
	})
}

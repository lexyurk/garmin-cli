package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/lexyurk/garmin-cli/internal/auth"
	"github.com/lexyurk/garmin-cli/internal/client"
	"github.com/lexyurk/garmin-cli/internal/config"
	"github.com/lexyurk/garmin-cli/internal/output"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

func NewAuthCmd(opts *globalOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auth",
		Short: "Authentication commands",
	}

	cmd.AddCommand(
		newAuthLoginCmd(opts),
		newAuthStatusCmd(opts),
		newAuthLogoutCmd(opts),
	)

	return cmd
}

func newAuthLoginCmd(opts *globalOptions) *cobra.Command {
	var email string
	var password string
	var passwordStdin bool
	var mfaCode string

	cmd := &cobra.Command{
		Use:   "login",
		Short: "Login to Garmin Connect",
		RunE: func(cmd *cobra.Command, args []string) error {
			if email == "" {
				email = os.Getenv("GARMIN_EMAIL")
			}
			if passwordStdin {
				if strings.TrimSpace(password) != "" {
					return fmt.Errorf("use either --password or --password-stdin (not both)")
				}
				b, err := io.ReadAll(cmd.InOrStdin())
				if err != nil {
					return err
				}
				password = strings.TrimSpace(string(b))
			} else if password == "" {
				password = os.Getenv("GARMIN_PASSWORD")
			}
			if !passwordStdin && password == "" {
				// Try to prompt interactively (avoids leaking password via process args).
				if tty, err := os.Open("/dev/tty"); err == nil {
					defer tty.Close()
					fmt.Fprint(os.Stderr, "Password: ")
					b, err := term.ReadPassword(int(tty.Fd()))
					fmt.Fprintln(os.Stderr)
					if err != nil {
						return err
					}
					password = strings.TrimSpace(string(b))
				}
			}
			if email == "" || password == "" {
				return fmt.Errorf("missing credentials: provide --email/--password, --password-stdin, or set GARMIN_EMAIL/GARMIN_PASSWORD")
			}

			cfgDir, err := config.ResolveConfigDir(opts.ConfigDir)
			if err != nil {
				return err
			}

			ctx := cmd.Context()
			prompt := func() (string, error) {
				if mfaCode != "" {
					return mfaCode, nil
				}
				// Try to read from /dev/tty so `--password-stdin` can coexist with interactive MFA.
				if tty, err := os.Open("/dev/tty"); err == nil {
					defer tty.Close()
					return readLine("MFA code: ", tty)
				}
				return readLine("MFA code: ", cmd.InOrStdin())
			}

			session, err := auth.Login(ctx, cfgDir, email, password, prompt)
			if err != nil {
				return err
			}
			if err := auth.SaveSession(cfgDir, opts.Profile, session); err != nil {
				return err
			}

			if opts.Format == "json" {
				return output.JSON(map[string]any{
					"authenticated": true,
					"profile":       opts.Profile,
					"expires_at":    session.OAuth2.ExpiresAt,
				})
			}

			exp := time.Unix(session.OAuth2.ExpiresAt, 0).Format(time.RFC3339)
			return renderKV(opts.Format, "Authenticated", map[string]string{
				"profile":    orDefault(opts.Profile, "default"),
				"expires_at": exp,
			})
		},
	}

	cmd.Flags().StringVar(&email, "email", "", "Garmin Connect email")
	cmd.Flags().StringVar(&password, "password", "", "Garmin Connect password")
	cmd.Flags().BoolVar(&passwordStdin, "password-stdin", false, "Read Garmin Connect password from stdin")
	cmd.Flags().StringVar(&mfaCode, "mfa-code", "", "MFA code (optional; if not provided, you will be prompted)")

	return cmd
}

func newAuthStatusCmd(opts *globalOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Check authentication status",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfgDir, err := config.ResolveConfigDir(opts.ConfigDir)
			if err != nil {
				return err
			}

			// Lightweight verified call (also refreshes OAuth2 if needed).
			ctx := cmd.Context()
			c, err := client.New(cfgDir, opts.Profile, client.Options{})
			if err != nil {
				if errors.Is(err, auth.ErrNotAuthenticated) {
					if opts.Format == "json" {
						return output.JSON(map[string]any{
							"authenticated": false,
							"profile":       opts.Profile,
						})
					}
					return renderKV(opts.Format, "Authentication", map[string]string{
						"status":  "not authenticated",
						"message": "Run `garmin auth login`",
						"profile": orDefault(opts.Profile, "default"),
					})
				}
				if opts.Format == "json" {
					return output.JSON(map[string]any{
						"authenticated": false,
						"error":         err.Error(),
						"profile":       opts.Profile,
					})
				}
				_ = renderKV(opts.Format, "Authentication", map[string]string{
					"status":  "not authenticated",
					"message": "Run `garmin auth login`",
					"profile": orDefault(opts.Profile, "default"),
				})
				return err
			}

			var profile map[string]any
			err = c.GetJSON(ctx, "/userprofile-service/socialProfile", nil, &profile)
			if err != nil {
				return err
			}

			displayName := ""
			if v, ok := profile["displayName"].(string); ok {
				displayName = v
			}

			if opts.Format == "json" {
				// Keep it intentionally small; don't emit tokens.
				sess, _ := auth.LoadSession(cfgDir, opts.Profile)
				return output.JSON(map[string]any{
					"authenticated": true,
					"profile":       opts.Profile,
					"display_name":  displayName,
					"expires_at":    sess.OAuth2.ExpiresAt,
				})
			}

			sess, _ := auth.LoadSession(cfgDir, opts.Profile)
			exp := time.Unix(sess.OAuth2.ExpiresAt, 0).Format(time.RFC3339)
			return renderKV(opts.Format, "Authentication", map[string]string{
				"status":       "authenticated",
				"profile":      orDefault(opts.Profile, "default"),
				"display_name": displayName,
				"expires_at":   exp,
			})
		},
	}
	return cmd
}

func newAuthLogoutCmd(opts *globalOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "logout",
		Short: "Clear stored tokens",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfgDir, err := config.ResolveConfigDir(opts.ConfigDir)
			if err != nil {
				return err
			}
			if err := auth.Logout(cfgDir, opts.Profile); err != nil {
				return err
			}
			if opts.Format == "json" {
				return output.JSON(map[string]any{
					"ok":      true,
					"profile": opts.Profile,
				})
			}
			return renderKV(opts.Format, "Logged out", map[string]string{
				"profile": orDefault(opts.Profile, "default"),
			})
		},
	}
	return cmd
}

func orDefault(s, d string) string {
	if strings.TrimSpace(s) == "" {
		return d
	}
	return s
}

func readLine(prompt string, r io.Reader) (string, error) {
	fmt.Fprint(os.Stderr, prompt)
	line, err := bufio.NewReader(r).ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return "", err
	}
	return strings.TrimSpace(line), nil
}



package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var version = "dev"

var rootCmd = &cobra.Command{
	Use:   "garmin-cli",
	Short: "Garmin Connect from your terminal",
	Long:  "Fast, ergonomic Garmin Connect CLI. Pipe it, script it, automate it.",
}

func init() {
	// Global flags
	rootCmd.PersistentFlags().StringP("format", "f", "json", "Output format: json, table, human")
	rootCmd.PersistentFlags().BoolP("verbose", "v", false, "Verbose output")
	rootCmd.PersistentFlags().BoolP("quiet", "q", false, "Suppress non-essential output")
	rootCmd.PersistentFlags().StringP("config", "c", "", "Path to config file")
	rootCmd.PersistentFlags().StringP("profile", "p", "", "Named profile to use")

	// Register subcommands
	rootCmd.AddCommand(authCmd)
	rootCmd.AddCommand(healthCmd)
	rootCmd.AddCommand(activitiesCmd)
	rootCmd.AddCommand(trainingCmd)
	rootCmd.AddCommand(versionCmd)
}

// --- auth ---

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Authentication commands",
}

var authLoginCmd = &cobra.Command{
	Use:   "login",
	Short: "Login to Garmin Connect",
	RunE: func(cmd *cobra.Command, args []string) error {
		// TODO: implement Garmin SSO authentication
		fmt.Println("TODO: garmin-cli auth login")
		return nil
	},
}

var authStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Check authentication status",
	RunE: func(cmd *cobra.Command, args []string) error {
		// TODO: check if tokens are valid
		fmt.Println("TODO: garmin-cli auth status")
		return nil
	},
}

var authLogoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Clear stored tokens",
	RunE: func(cmd *cobra.Command, args []string) error {
		// TODO: remove stored tokens
		fmt.Println("TODO: garmin-cli auth logout")
		return nil
	},
}

func init() {
	authLoginCmd.Flags().String("email", "", "Garmin Connect email")
	authLoginCmd.Flags().String("password", "", "Garmin Connect password")
	authCmd.AddCommand(authLoginCmd, authStatusCmd, authLogoutCmd)
}

// --- health ---

var healthCmd = &cobra.Command{
	Use:   "health",
	Short: "Health data (sleep, HR, steps, stress, body battery)",
}

var healthSleepCmd = &cobra.Command{
	Use:   "sleep",
	Short: "Sleep data",
	RunE: func(cmd *cobra.Command, args []string) error {
		// TODO: fetch sleep data from Garmin Connect API
		fmt.Println("TODO: garmin-cli health sleep")
		return nil
	},
}

var healthHeartRateCmd = &cobra.Command{
	Use:   "heart-rate",
	Short: "Heart rate data",
	RunE: func(cmd *cobra.Command, args []string) error {
		// TODO: fetch heart rate data
		fmt.Println("TODO: garmin-cli health heart-rate")
		return nil
	},
}

var healthStepsCmd = &cobra.Command{
	Use:   "steps",
	Short: "Step count",
	RunE: func(cmd *cobra.Command, args []string) error {
		// TODO: fetch step data
		fmt.Println("TODO: garmin-cli health steps")
		return nil
	},
}

var healthStressCmd = &cobra.Command{
	Use:   "stress",
	Short: "Stress levels",
	RunE: func(cmd *cobra.Command, args []string) error {
		// TODO: fetch stress data
		fmt.Println("TODO: garmin-cli health stress")
		return nil
	},
}

var healthBodyBatteryCmd = &cobra.Command{
	Use:   "body-battery",
	Short: "Body battery",
	RunE: func(cmd *cobra.Command, args []string) error {
		// TODO: fetch body battery data
		fmt.Println("TODO: garmin-cli health body-battery")
		return nil
	},
}

func init() {
	for _, cmd := range []*cobra.Command{healthSleepCmd, healthHeartRateCmd, healthStepsCmd, healthStressCmd, healthBodyBatteryCmd} {
		cmd.Flags().String("date", "", "Date (YYYY-MM-DD, default: today)")
	}
	healthCmd.AddCommand(healthSleepCmd, healthHeartRateCmd, healthStepsCmd, healthStressCmd, healthBodyBatteryCmd)
}

// --- activities ---

var activitiesCmd = &cobra.Command{
	Use:   "activities",
	Short: "Activity management",
}

var activitiesListCmd = &cobra.Command{
	Use:   "list",
	Short: "List activities",
	RunE: func(cmd *cobra.Command, args []string) error {
		// TODO: list activities from Garmin Connect
		fmt.Println("TODO: garmin-cli activities list")
		return nil
	},
}

var activitiesGetCmd = &cobra.Command{
	Use:   "get [activity-id]",
	Short: "Get activity details",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		// TODO: get activity details
		fmt.Printf("TODO: garmin-cli activities get %s\n", args[0])
		return nil
	},
}

var activitiesSplitsCmd = &cobra.Command{
	Use:   "splits [activity-id]",
	Short: "Get activity splits/laps",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		// TODO: get activity splits
		fmt.Printf("TODO: garmin-cli activities splits %s\n", args[0])
		return nil
	},
}

func init() {
	activitiesListCmd.Flags().Int("limit", 20, "Number of activities to return")
	activitiesListCmd.Flags().String("after", "", "Activities after date (YYYY-MM-DD)")
	activitiesListCmd.Flags().String("before", "", "Activities before date (YYYY-MM-DD)")
	activitiesListCmd.Flags().String("type", "", "Activity type filter (running, cycling, etc.)")
	activitiesGetCmd.Flags().Bool("details", false, "Include extended details")
	activitiesCmd.AddCommand(activitiesListCmd, activitiesGetCmd, activitiesSplitsCmd)
}

// --- training ---

var trainingCmd = &cobra.Command{
	Use:   "training",
	Short: "Training metrics (status, VO2max, HRV, readiness)",
}

var trainingStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Training status (Productive, Peaking, etc.)",
	RunE: func(cmd *cobra.Command, args []string) error {
		// TODO: fetch training status
		fmt.Println("TODO: garmin-cli training status")
		return nil
	},
}

var trainingReadinessCmd = &cobra.Command{
	Use:   "readiness",
	Short: "Training readiness score (0-100)",
	RunE: func(cmd *cobra.Command, args []string) error {
		// TODO: fetch training readiness
		fmt.Println("TODO: garmin-cli training readiness")
		return nil
	},
}

var trainingVo2maxCmd = &cobra.Command{
	Use:   "vo2max",
	Short: "VO2 max estimates",
	RunE: func(cmd *cobra.Command, args []string) error {
		// TODO: fetch VO2 max
		fmt.Println("TODO: garmin-cli training vo2max")
		return nil
	},
}

var trainingHrvCmd = &cobra.Command{
	Use:   "hrv",
	Short: "Heart rate variability",
	RunE: func(cmd *cobra.Command, args []string) error {
		// TODO: fetch HRV data
		fmt.Println("TODO: garmin-cli training hrv")
		return nil
	},
}

func init() {
	for _, cmd := range []*cobra.Command{trainingStatusCmd, trainingReadinessCmd, trainingVo2maxCmd, trainingHrvCmd} {
		cmd.Flags().String("date", "", "Date (YYYY-MM-DD, default: today)")
	}
	trainingCmd.AddCommand(trainingStatusCmd, trainingReadinessCmd, trainingVo2maxCmd, trainingHrvCmd)
}

// --- version ---

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("garmin-cli %s\n", version)
	},
}

// --- main ---

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

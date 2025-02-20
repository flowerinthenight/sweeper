package main

import (
	goflag "flag"
	"log"
	"os"

	"github.com/flowerinthenight/sweeper/gdrive"
	ll "github.com/flowerinthenight/sweeper/log"
	"github.com/flowerinthenight/sweeper/params"
	"github.com/spf13/cobra"
	flag "github.com/spf13/pflag"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"

	rootCmd = &cobra.Command{
		Use:   "sweeper",
		Short: "Not yet sure what this is",
		Long: `Not yet sure what this is.

[version=` + version + `, commit=` + commit + `]`,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			goflag.Parse() // for cobra + glog flags
		},
		Run: func(cmd *cobra.Command, args []string) {
			ll.Fail("invalid cmd, please run -h")
		},
	}
)

func init() {
	rootCmd.Flags().SortFlags = false
	rootCmd.PersistentFlags().SortFlags = false
	rootCmd.PersistentFlags().StringVar(&params.RunEnv, "env", params.RunEnv, "runtime environment (dev, qa, next, prod); some cmds get this info from input creds")
	rootCmd.PersistentFlags().StringVar(&params.Region, "region", os.Getenv("AWS_REGION"), "aws region")
	rootCmd.PersistentFlags().StringVar(&params.Key, "key", os.Getenv("AWS_ACCESS_KEY_ID"), "aws key")
	rootCmd.PersistentFlags().StringVar(&params.Secret, "secret", os.Getenv("AWS_SECRET_ACCESS_KEY"), "secret")
	rootCmd.PersistentFlags().StringVar(&params.RoleArn, "rolearn", os.Getenv("ROLE_ARN"), "role arn to assume using --key and --secret")
	rootCmd.PersistentFlags().StringVar(&params.RoleArnAlm, "rolearnalm", os.Getenv("ROLE_ARN_ALM"), "role arn to assume ALM")
	rootCmd.AddCommand(
		gdrive.ForexRatesCmd(),
		TestCmd(),
	)

	// For cobra + glog flags.
	flag.CommandLine.AddGoFlagSet(goflag.CommandLine)
}

func main() {
	log.SetOutput(os.Stdout)
	cobra.EnableCommandSorting = false
	rootCmd.Execute()
}

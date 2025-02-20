package gdrive

import (
	"os"

	ll "github.com/flowerinthenight/sweeper/log"
	"github.com/spf13/cobra"
)

func ForexRatesCmd() *cobra.Command {
	var (
		fixerKey string
	)

	type rates struct {
		Success   bool               `json:"success"`
		Timestamp int64              `json:"timestamp"`
		Base      string             `json:"base"`
		Date      string             `json:"date"`
		Rates     map[string]float64 `json:"rates"`
	}

	cmd := &cobra.Command{
		Use:   "forexrates [yyyy-mm-dd]",
		Short: "Update our forex rate records",
		Long:  `Update our forex rate records, with option to specify date.`,
		Run: func(cmd *cobra.Command, args []string) {
			if os.Getenv("GOOGLE_APPLICATION_CREDENTIALS") == "" {
				ll.Fail("GOOGLE_APPLICATION_CREDENTIALS is not set")
			}

		},
		SilenceUsage: true,
	}

	cmd.Flags().SortFlags = false
	cmd.Flags().StringVar(&fixerKey, "apikey", os.Getenv("FIXERIO_APIKEY"), "fixer.io API key, defaults to $FIXERIO_APIKEY")
	return cmd
}

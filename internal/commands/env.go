package commands

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func NewEnvCheckCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "env",
		Short: "Show environment variables",
		RunE:  runEnvChecks(),
	}
	return cmd
}

func runEnvChecks() func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		fmt.Printf("\nDetected Environment Variables:\n\n")

		fmt.Printf("%30v", "Base URI: ")
		fmt.Println(os.Getenv("CX_BASE_URI"))
		fmt.Printf("%30v", "Base Auth URI: ")
		fmt.Println(os.Getenv("CX_BASE_AUTH_URI"))
		fmt.Printf("%30v", "AST Tenant: ")
		fmt.Println(os.Getenv("CX_TENANT"))
		fmt.Printf("%30v", "HTTP Proxy: ")
		fmt.Println(os.Getenv("CX_HTTP_PROXY"))
		fmt.Printf("%30v", "Client ID: ")
		fmt.Println(os.Getenv("CX_CLIENT_ID"))
		fmt.Printf("%30v", "Client Secret: ")
		fmt.Println(os.Getenv("CX_BASE_URI"))
		fmt.Printf("%30v", "Client Secret: ")
		fmt.Println(os.Getenv("CX_CLIENT_SECRET"))
		fmt.Printf("%30v", "APIKEY: ")
		fmt.Println(os.Getenv("CX_APIKEY"))
		fmt.Printf("%30v", "Branch: ")
		fmt.Println(os.Getenv("CX_BRANCH"))
		return nil
	}
}

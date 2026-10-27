// SPDX-License-Identifier: MIT

package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

var (
	brokerURL string
	clientID  string
	username  string
	password  string
	tlsCert   string
	tlsKey    string
	tlsCA     string
	jsonOut   bool
)

var rootCmd = &cobra.Command{
	Use:   "sparkplugbctl",
	Short: "Sparkplug B 3.0 command-line tool",
	Long: `sparkplugbctl is a CLI for interacting with Sparkplug B 3.0 MQTT
infrastructure. It can listen for messages, simulate edge nodes and host
applications, and encode/decode Sparkplug B payloads.

Every global flag can also be set by an environment variable: SPARKPLUGBCTL_
and the flag name in upper case with "_" for "-", e.g. SPARKPLUGBCTL_BROKER.
A flag on the command line takes precedence.`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		return applyEnv(cmd.Root().PersistentFlags())
	},
}

// envPrefix prefixes the environment variable of each global flag.
const envPrefix = "SPARKPLUGBCTL_"

// applyEnv sets every flag of fs not given on the command line from its
// environment variable, when that is set.
func applyEnv(fs *pflag.FlagSet) error {
	var err error
	fs.VisitAll(func(f *pflag.Flag) {
		if err != nil || f.Changed {
			return
		}
		name := envPrefix + strings.ToUpper(strings.ReplaceAll(f.Name, "-", "_"))
		if v, ok := os.LookupEnv(name); ok {
			if e := fs.Set(f.Name, v); e != nil {
				err = fmt.Errorf("%s: %w", name, e)
			}
		}
	})
	return err
}

func init() {
	rootCmd.PersistentFlags().StringVar(&brokerURL, "broker", "tcp://localhost:1883", "MQTT broker URL")
	rootCmd.PersistentFlags().StringVar(&clientID, "client-id", "", "MQTT client ID (auto-generated if empty)")
	rootCmd.PersistentFlags().StringVar(&username, "username", "", "MQTT username")
	rootCmd.PersistentFlags().StringVar(&password, "password", "", "MQTT password")
	rootCmd.PersistentFlags().StringVar(&tlsCert, "tls-cert", "", "TLS client certificate file")
	rootCmd.PersistentFlags().StringVar(&tlsKey, "tls-key", "", "TLS client key file")
	rootCmd.PersistentFlags().StringVar(&tlsCA, "tls-ca", "", "TLS CA certificate file")
	rootCmd.PersistentFlags().BoolVar(&jsonOut, "json", false, "output as JSON")
}

func Execute() error {
	if err := rootCmd.Execute(); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		return err
	}
	return nil
}

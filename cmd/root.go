// SPDX-License-Identifier: MIT

package cmd

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

var (
	brokerURL    string
	clientID     string
	username     string
	password     string
	passwordFile string
	tlsCert      string
	tlsKey       string
	tlsCA        string
	jsonOut      bool
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
		commandEntered = true
		pf := cmd.Root().PersistentFlags()
		if err := applyEnv(pf); err != nil {
			return withExit(ExitUsage, err)
		}
		if password != "" && passwordFile != "" {
			return withExit(ExitUsage, errors.New("--password and --password-file (or SPARKPLUGBCTL_PASSWORD and SPARKPLUGBCTL_PASSWORD_FILE) are both set: give one"))
		}
		return applyFormatEnv(pf.Changed("json"))
	},
}

// envPrefix prefixes the environment variable of each global flag.
const envPrefix = "SPARKPLUGBCTL_"

// envAlternatives pairs the flags that give the same value: one given on
// the command line keeps the variable of the other from applying.
var envAlternatives = map[string]string{"password": "password-file", "password-file": "password"}

// applyEnv sets every flag of fs not given on the command line from its
// environment variable, when that is set.
func applyEnv(fs *pflag.FlagSet) error {
	onCommandLine := map[string]bool{}
	fs.VisitAll(func(f *pflag.Flag) { onCommandLine[f.Name] = f.Changed })
	var err error
	fs.VisitAll(func(f *pflag.Flag) {
		if err != nil || f.Changed || onCommandLine[envAlternatives[f.Name]] {
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
	rootCmd.PersistentFlags().StringVar(&clientID, "client-id", "", "MQTT client ID (default: sparkplugbctl-<command>-<8 random hex digits>; host and edge derive it from their IDs)")
	rootCmd.PersistentFlags().StringVar(&username, "username", "", "MQTT username")
	rootCmd.PersistentFlags().StringVar(&password, "password", "", "MQTT password (deprecated: use --password-file or SPARKPLUGBCTL_PASSWORD)")
	rootCmd.PersistentFlags().StringVar(&passwordFile, "password-file", "", "file holding the MQTT password (one trailing line end is ignored)")
	rootCmd.PersistentFlags().StringVar(&tlsCert, "tls-cert", "", "TLS client certificate file")
	rootCmd.PersistentFlags().StringVar(&tlsKey, "tls-key", "", "TLS client key file")
	rootCmd.PersistentFlags().StringVar(&tlsCA, "tls-ca", "", "TLS CA certificate file")
	rootCmd.PersistentFlags().BoolVar(&jsonOut, "json", false, "output as JSON")
	completeFlags(rootCmd, cobra.NoFileCompletions, "broker", "client-id", "username", "password")
	completeFlags(rootCmd, fileExt("pem", "crt", "cer"), "tls-cert", "tls-ca")
	completeFlags(rootCmd, fileExt("pem", "key"), "tls-key")
	completeFlags(rootCmd, anyFile, "password-file")
}

func Execute() error {
	commandEntered = false
	if err := rootCmd.Execute(); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		return err
	}
	return nil
}

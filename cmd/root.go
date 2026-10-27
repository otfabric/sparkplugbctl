// SPDX-License-Identifier: MIT

package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
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
applications, and encode/decode Sparkplug B payloads.`,
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

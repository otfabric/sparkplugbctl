// SPDX-License-Identifier: MIT

package cmd

import (
	"context"
	"fmt"
	"os/signal"
	"strconv"
	"strings"
	"syscall"

	"github.com/otfabric/go-sparkplugb/mqtt"
	"github.com/otfabric/go-sparkplugb/payload"
	pb "github.com/otfabric/go-sparkplugb/proto"
	"github.com/spf13/cobra"
)

var (
	pubTopic   string
	pubMetrics []string
	pubSeq     uint64
)

var publishCmd = &cobra.Command{
	ValidArgsFunction: cobra.NoFileCompletions,
	Use:               "publish",
	Short:             "Publish a Sparkplug B payload",
	Long: `Publish creates a Sparkplug B payload from the given metrics and
publishes it to the specified topic.

Metric format: name:type=value
Examples:
  --metric "temperature:Double=22.5"
  --metric "status:Boolean=true"
  --metric "label:String=hello"`,
	RunE: runPublish,
}

func init() {
	publishCmd.Flags().StringVar(&pubTopic, "topic", "", "MQTT topic (required)")
	publishCmd.Flags().StringSliceVar(&pubMetrics, "metric", nil, "metric in name:type=value format (repeatable)")
	publishCmd.Flags().Uint64Var(&pubSeq, "seq", 0, "sequence number (0 to 255)")
	completeFlags(publishCmd, cobra.NoFileCompletions, "topic", "seq")
	completeFlags(publishCmd, completeMetric, "metric")
	_ = publishCmd.MarkFlagRequired("topic")
	rootCmd.AddCommand(publishCmd)
}

// maxSeq is the largest payload sequence number: Sparkplug 3.0.0 counts
// from 0 to 255 and starts again at 0.
const maxSeq = 255

func runPublish(cmd *cobra.Command, args []string) error {
	// A topic filter is no topic name to publish to (MQTT 3.1.1 section
	// 4.7.1): a broker delivers such a message to nobody, or ends the
	// connection.
	if strings.ContainsAny(pubTopic, "+#") {
		return withExit(ExitInvalidInput, fmt.Errorf("--topic %q: a topic to publish to has no wildcard (+ or #)", pubTopic))
	}
	if pubSeq > maxSeq {
		return withExit(ExitInvalidInput, fmt.Errorf("--seq %d: a Sparkplug sequence number is 0 to %d", pubSeq, maxSeq))
	}
	metrics, err := parseMetrics(pubMetrics)
	if err != nil {
		return withExit(ExitInvalidInput, err)
	}

	p := payload.NewPayload(pubSeq, metrics...)
	data, err := encodePayload(p)
	if err != nil {
		return withExit(ExitInvalidInput, fmt.Errorf("encode: %w", err))
	}

	opts, err := connOpts(cmd.ErrOrStderr())
	if err != nil {
		return err
	}
	transport, err := newTransport(opts)
	if err != nil {
		return withExit(ExitInvalidInput, err)
	}

	ctx, cancel := signal.NotifyContext(cmd.Context(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	cid := clientID
	if cid == "" {
		cid = randomClientID("publish")
	}

	err = transport.Connect(ctx, mqtt.ConnectOpts{
		BrokerURL:    brokerURL,
		ClientID:     cid,
		CleanSession: true,
		Username:     opts.Username,
		Password:     opts.Password,
	})
	if err != nil {
		return withExit(ExitTransport, fmt.Errorf("connect: %w", err))
	}
	defer func() { _ = transport.Disconnect(context.Background()) }()

	if err := transport.Publish(ctx, pubTopic, 0, false, data); err != nil {
		return withExit(ExitTransport, fmt.Errorf("publish: %w", err))
	}

	_, err = fmt.Fprintf(cmd.OutOrStdout(), "Published %d metric(s) to %s\n", len(metrics), pubTopic)
	return outputError(err)
}

func parseMetrics(specs []string) ([]*pb.Payload_Metric, error) {
	var metrics []*pb.Payload_Metric
	for _, spec := range specs {
		m, err := parseMetricSpec(spec)
		if err != nil {
			return nil, fmt.Errorf("metric %q: %w", spec, err)
		}
		metrics = append(metrics, m)
	}
	return metrics, nil
}

func parseMetricSpec(spec string) (*pb.Payload_Metric, error) {
	colonIdx := strings.Index(spec, ":")
	if colonIdx < 0 {
		return nil, fmt.Errorf("expected name:type=value")
	}
	name := spec[:colonIdx]
	rest := spec[colonIdx+1:]

	eqIdx := strings.Index(rest, "=")
	if eqIdx < 0 {
		return nil, fmt.Errorf("expected name:type=value")
	}
	typeName := rest[:eqIdx]
	valStr := rest[eqIdx+1:]

	dt, err := parseDataType(typeName)
	if err != nil {
		return nil, err
	}

	val, err := parseValue(dt, valStr)
	if err != nil {
		return nil, fmt.Errorf("parse value: %w", err)
	}

	return payload.NewMetric(name, dt, val)
}

func parseDataType(name string) (pb.DataType, error) {
	switch strings.ToLower(name) {
	case "int8":
		return pb.DataType_Int8, nil
	case "int16":
		return pb.DataType_Int16, nil
	case "int32":
		return pb.DataType_Int32, nil
	case "int64":
		return pb.DataType_Int64, nil
	case "uint8":
		return pb.DataType_UInt8, nil
	case "uint16":
		return pb.DataType_UInt16, nil
	case "uint32":
		return pb.DataType_UInt32, nil
	case "uint64":
		return pb.DataType_UInt64, nil
	case "float":
		return pb.DataType_Float, nil
	case "double":
		return pb.DataType_Double, nil
	case "boolean", "bool":
		return pb.DataType_Boolean, nil
	case "string":
		return pb.DataType_String, nil
	case "text":
		return pb.DataType_Text, nil
	case "uuid":
		return pb.DataType_UUID, nil
	case "bytes":
		return pb.DataType_Bytes, nil
	default:
		return 0, fmt.Errorf("unknown type %q", name)
	}
}

func parseValue(dt pb.DataType, s string) (any, error) {
	switch dt {
	case pb.DataType_Int8:
		v, err := strconv.ParseInt(s, 10, 8)
		return int8(v), err
	case pb.DataType_Int16:
		v, err := strconv.ParseInt(s, 10, 16)
		return int16(v), err
	case pb.DataType_Int32:
		v, err := strconv.ParseInt(s, 10, 32)
		return int32(v), err
	case pb.DataType_Int64:
		v, err := strconv.ParseInt(s, 10, 64)
		return v, err
	case pb.DataType_UInt8:
		v, err := strconv.ParseUint(s, 10, 8)
		return uint8(v), err
	case pb.DataType_UInt16:
		v, err := strconv.ParseUint(s, 10, 16)
		return uint16(v), err
	case pb.DataType_UInt32:
		v, err := strconv.ParseUint(s, 10, 32)
		return uint32(v), err
	case pb.DataType_UInt64:
		v, err := strconv.ParseUint(s, 10, 64)
		return v, err
	case pb.DataType_Float:
		v, err := strconv.ParseFloat(s, 32)
		return float32(v), err
	case pb.DataType_Double:
		v, err := strconv.ParseFloat(s, 64)
		return v, err
	case pb.DataType_Boolean:
		v, err := strconv.ParseBool(s)
		return v, err
	case pb.DataType_String, pb.DataType_Text, pb.DataType_UUID:
		return s, nil
	case pb.DataType_Bytes:
		return []byte(s), nil
	default:
		return nil, fmt.Errorf("unsupported type for parsing")
	}
}

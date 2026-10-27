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
	"github.com/otfabric/sparkplugbctl/internal/broker"
	"github.com/spf13/cobra"
)

var (
	pubTopic   string
	pubMetrics []string
	pubSeq     uint64
)

var publishCmd = &cobra.Command{
	Use:   "publish",
	Short: "Publish a Sparkplug B payload",
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
	publishCmd.Flags().Uint64Var(&pubSeq, "seq", 0, "sequence number")
	_ = publishCmd.MarkFlagRequired("topic")
	rootCmd.AddCommand(publishCmd)
}

func runPublish(cmd *cobra.Command, args []string) error {
	metrics, err := parseMetrics(pubMetrics)
	if err != nil {
		return err
	}

	p := payload.NewPayload(pubSeq, metrics...)
	data, err := payload.Encode(p)
	if err != nil {
		return fmt.Errorf("encode: %w", err)
	}

	transport, err := broker.NewTransport(brokerOpts())
	if err != nil {
		return err
	}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	cid := clientID
	if cid == "" {
		cid = "sparkplugbctl-publish"
	}

	err = transport.Connect(ctx, mqtt.ConnectOpts{
		BrokerURL:    brokerURL,
		ClientID:     cid,
		CleanSession: true,
		Username:     username,
		Password:     password,
	})
	if err != nil {
		return fmt.Errorf("connect: %w", err)
	}
	defer transport.Disconnect(context.Background())

	if err := transport.Publish(ctx, pubTopic, 0, false, data); err != nil {
		return fmt.Errorf("publish: %w", err)
	}

	fmt.Printf("Published %d metric(s) to %s\n", len(metrics), pubTopic)
	return nil
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

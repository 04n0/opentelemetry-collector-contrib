// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package awscloudwatchlogsexporter // import "github.com/open-telemetry/opentelemetry-collector-contrib/exporter/awscloudwatchlogsexporter"

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"
	"github.com/google/uuid"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	exp "go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/exporter/exporterhelper"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.uber.org/zap"

	"github.com/open-telemetry/opentelemetry-collector-contrib/exporter/awscloudwatchlogsexporter/internal/metadata"
	"github.com/open-telemetry/opentelemetry-collector-contrib/internal/aws/awsutil"
	"github.com/open-telemetry/opentelemetry-collector-contrib/internal/aws/cwlogs"
)

type cwlExporter struct {
	Config           *Config
	logger           *zap.Logger
	collectorID      string
	svcStructuredLog *cwlogs.Client
	pusherFactory    cwlogs.MultiStreamPusherFactory
}

type awsMetadata struct {
	LogGroupName  string `json:"logGroupName,omitempty"`
	LogStreamName string `json:"logStreamName,omitempty"`
}

type emfMetadata struct {
	AWSMetadata   *awsMetadata `json:"_aws,omitempty"`
	LogGroupName  string       `json:"log_group_name,omitempty"`
	LogStreamName string       `json:"log_stream_name,omitempty"`
}

func newCwLogsPusher(ctx context.Context, expConfig *Config, params exp.Settings) (*cwlExporter, error) {
	if expConfig == nil {
		return nil, errors.New("awscloudwatchlogs exporter config is nil")
	}

	expConfig.logger = params.Logger

	awsConfig, err := awsutil.GetAWSConfig(ctx, params.Logger, &expConfig.AWSSessionSettings)
	if err != nil {
		return nil, err
	}

	// create CWLogs client with aws session config
	svcStructuredLog := cwlogs.NewClient(params.Logger, awsConfig, params.BuildInfo, expConfig.LogGroupName, expConfig.LogRetention, expConfig.Tags, metadata.Type.String())
	collectorIdentifier, err := uuid.NewRandom()
	if err != nil {
		return nil, err
	}

	logStreamManager := cwlogs.NewLogStreamManager(*svcStructuredLog)
	multiStreamPusherFactory := cwlogs.NewMultiStreamPusherFactory(logStreamManager, *svcStructuredLog, params.Logger)

	logsExporter := &cwlExporter{
		svcStructuredLog: svcStructuredLog,
		Config:           expConfig,
		logger:           params.Logger,
		collectorID:      collectorIdentifier.String(),
		pusherFactory:    multiStreamPusherFactory,
	}
	return logsExporter, nil
}

func newCwLogsExporter(ctx context.Context, config component.Config, params exp.Settings) (exp.Logs, error) {
	expConfig := config.(*Config)
	logsPusher, err := newCwLogsPusher(ctx, expConfig, params)
	if err != nil {
		return nil, err
	}
	return exporterhelper.NewLogs(
		ctx,
		params,
		config,
		logsPusher.consumeLogs,
		exporterhelper.WithQueue(expConfig.QueueSettings),
		exporterhelper.WithRetry(expConfig.BackOffConfig),
		exporterhelper.WithCapabilities(consumer.Capabilities{MutatesData: false}),
		exporterhelper.WithShutdown(logsPusher.shutdown),
	)
}

func (e *cwlExporter) consumeLogs(ctx context.Context, ld plog.Logs) error {
	pusher := e.pusherFactory.CreateMultiStreamPusher()
	var errs error

	err := pushLogsToCWLogs(ctx, e.logger, ld, e.Config, pusher)
	if err != nil {
		errs = errors.Join(errs, fmt.Errorf("error pushing logs: %w", err))
	}

	err = pusher.ForceFlush(ctx)
	if err != nil {
		errs = errors.Join(errs, fmt.Errorf("error flushing logs: %w", err))
	}

	return errs
}

func (*cwlExporter) shutdown(context.Context) error {
	return nil
}

func pushLogsToCWLogs(ctx context.Context, logger *zap.Logger, ld plog.Logs, config *Config, pusher cwlogs.Pusher) error {
	n := ld.ResourceLogs().Len()

	if n == 0 {
		return nil
	}

	var errs error

	rls := ld.ResourceLogs()
	for i := 0; i < rls.Len(); i++ {
		rl := rls.At(i)
		resourceAttrs := attrsValue(rl.Resource().Attributes())

		sls := rl.ScopeLogs()
		for j := 0; j < sls.Len(); j++ {
			sl := sls.At(j)
			scope := sl.Scope()
			logs := sl.LogRecords()
			for k := 0; k < logs.Len(); k++ {
				log := logs.At(k)
				event, err := logToCWLog(resourceAttrs, scope, log, config)
				if err != nil {
					logger.Debug("Failed to convert to CloudWatch Log", zap.Error(err))
				} else {
					err := pusher.AddLogEntry(ctx, event)
					if err != nil {
						errs = errors.Join(errs, err)
					}
				}
			}
		}
	}

	return errs
}

type scopeCwLogBody struct {
	Name       string         `json:"name,omitempty"`
	Version    string         `json:"version,omitempty"`
	Attributes map[string]any `json:"attributes,omitempty"`
}

type cwLogBody struct {
	Body                   any             `json:"body,omitempty"`
	SeverityNumber         int32           `json:"severity_number,omitempty"`
	SeverityText           string          `json:"severity_text,omitempty"`
	DroppedAttributesCount uint32          `json:"dropped_attributes_count,omitempty"`
	Flags                  uint32          `json:"flags,omitempty"`
	TraceID                string          `json:"trace_id,omitempty"`
	SpanID                 string          `json:"span_id,omitempty"`
	Attributes             map[string]any  `json:"attributes,omitempty"`
	Scope                  *scopeCwLogBody `json:"scope,omitempty"`
	Resource               map[string]any  `json:"resource,omitempty"`
}

func logToCWLog(resourceAttrs map[string]any, scope pcommon.InstrumentationScope, log plog.LogRecord, config *Config) (*cwlogs.Event, error) {
	// TODO(jbd): Benchmark and improve the allocations.
	// Evaluate go.elastic.co/fastjson as a replacement for encoding/json.
	// Replace loggroup and logstream with resource attribute
	logGroupName, logStreamName, _ := getLogInfo(resourceAttrs, config)

	var bodyJSON []byte
	var err error
	if config.RawLog {
		// Check if this is an emf log
		var metadata emfMetadata
		bodyString := log.Body().AsString()
		err = json.Unmarshal([]byte(bodyString), &metadata)
		// v1 emf json
		if err == nil && metadata.AWSMetadata != nil && metadata.AWSMetadata.LogGroupName != "" {
			logGroupName = metadata.AWSMetadata.LogGroupName
			if metadata.AWSMetadata.LogStreamName != "" {
				logStreamName = metadata.AWSMetadata.LogStreamName
			}
		} else /* v0 emf json */ if err == nil && metadata.LogGroupName != "" {
			logGroupName = metadata.LogGroupName
			if metadata.LogStreamName != "" {
				logStreamName = metadata.LogStreamName
			}
		}
		bodyJSON = []byte(bodyString)
	} else {
		body := cwLogBody{
			Body:                   log.Body().AsRaw(),
			SeverityNumber:         int32(log.SeverityNumber()),
			SeverityText:           log.SeverityText(),
			DroppedAttributesCount: log.DroppedAttributesCount(),
			Flags:                  uint32(log.Flags()),
		}
		if traceID := log.TraceID(); !traceID.IsEmpty() {
			body.TraceID = hex.EncodeToString(traceID[:])
		}
		if spanID := log.SpanID(); !spanID.IsEmpty() {
			body.SpanID = hex.EncodeToString(spanID[:])
		}
		body.Attributes = attrsValue(log.Attributes())
		body.Resource = resourceAttrs

		// scope should have a name at least
		if scope.Name() != "" {
			scopeBody := &scopeCwLogBody{
				Name:       scope.Name(),
				Version:    scope.Version(),
				Attributes: attrsValue(scope.Attributes()),
			}
			body.Scope = scopeBody
		}

		bodyJSON, err = json.Marshal(body)
		if err != nil {
			return &cwlogs.Event{}, err
		}
	}

	return &cwlogs.Event{
		InputLogEvent: types.InputLogEvent{
			Timestamp: aws.Int64(int64(log.Timestamp()) / int64(time.Millisecond)), // in milliseconds
			Message:   aws.String(string(bodyJSON)),
		},
		StreamKey: cwlogs.StreamKey{
			LogGroupName:  logGroupName,
			LogStreamName: logStreamName,
		},
		GeneratedTime: time.Now(),
	}, nil
}

func attrsValue(attrs pcommon.Map) map[string]any {
	if attrs.Len() == 0 {
		return nil
	}
	out := make(map[string]any, attrs.Len())
	for k, v := range attrs.All() {
		out[k] = v.AsRaw()
	}
	return out
}

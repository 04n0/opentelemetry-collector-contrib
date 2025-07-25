// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package coralogixprocessor // import "github.com/open-telemetry/opentelemetry-collector-contrib/processor/coralogixprocessor"

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.opentelemetry.io/collector/processor"
	"go.opentelemetry.io/collector/processor/processorhelper"
	"go.uber.org/zap"

	"github.com/open-telemetry/opentelemetry-collector-contrib/processor/coralogixprocessor/internal/transactions"
)

type coralogixProcessor struct {
	config *Config
	component.StartFunc
	component.ShutdownFunc
	logger *zap.Logger
}

func newCoralogixProcessor(ctx context.Context, set processor.Settings, cfg *Config, nextConsumer consumer.Traces) (processor.Traces, error) {
	sp := &coralogixProcessor{
		config: cfg,
		logger: set.Logger.With(zap.String("component", "coralogixprocessor")),
	}

	return processorhelper.NewTraces(ctx,
		set,
		cfg,
		nextConsumer,
		sp.processTraces,
		processorhelper.WithCapabilities(consumer.Capabilities{MutatesData: true}))
}

func (sp *coralogixProcessor) processTraces(_ context.Context, td ptrace.Traces) (ptrace.Traces, error) {
	//nolint:staticcheck // QF1008: Keeping embedded field for clarity
	if sp.config.TransactionsConfig.Enabled {
		tracesWithTransactions, err := transactions.ApplyTransactionsAttributes(
			td,
			sp.logger.With(zap.String("feature", "transactions")),
		)
		if err != nil {
			return tracesWithTransactions, err
		}
	}

	return td, nil
}

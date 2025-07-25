// Copyright The OpenTelemetry Authors
// Copyright (c) 2019 The Jaeger Authors.
// Copyright (c) 2018 Uber Technologies, Inc.
// SPDX-License-Identifier: Apache-2.0

package zipkin // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/translator/zipkin/zipkinthriftconverter"

import (
	"context"

	"github.com/apache/thrift/lib/go/thrift"
	"github.com/jaegertracing/jaeger-idl/thrift-gen/zipkincore"
)

// SerializeThrift is only used in tests.
func SerializeThrift(ctx context.Context, spans []*zipkincore.Span) ([]byte, error) {
	t := thrift.NewTMemoryBuffer()
	p := thrift.NewTBinaryProtocolConf(t, &thrift.TConfiguration{})
	if err := p.WriteListBegin(ctx, thrift.STRUCT, len(spans)); err != nil {
		return nil, err
	}

	for _, s := range spans {
		if err := s.Write(ctx, p); err != nil {
			return nil, err
		}
	}
	if err := p.WriteListEnd(ctx); err != nil {
		return nil, err
	}
	return t.Bytes(), nil
}

// DeserializeThrift decodes Thrift bytes to a list of spans.
func DeserializeThrift(ctx context.Context, b []byte) ([]*zipkincore.Span, error) {
	buffer := thrift.NewTMemoryBuffer()
	buffer.Write(b)

	transport := thrift.NewTBinaryProtocolConf(buffer, &thrift.TConfiguration{})
	_, size, err := transport.ReadListBegin(ctx) // Ignore the returned element type
	if err != nil {
		return nil, err
	}

	// We don't depend on the size returned by ReadListBegin to preallocate the array because it
	// sometimes returns a nil error on bad input and provides an unreasonably large int for size
	var spans []*zipkincore.Span
	for i := 0; i < size; i++ {
		zs := &zipkincore.Span{}
		err = zs.Read(ctx, transport)
		if err != nil {
			return nil, err
		}
		spans = append(spans, zs)
	}

	return spans, nil
}

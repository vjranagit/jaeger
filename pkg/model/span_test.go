package model

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSpan(t *testing.T) {
	span := NewSpan()

	assert.NotNil(t, span)
	assert.NotZero(t, span.StartTime)
	assert.NotNil(t, span.Tags)
	assert.NotNil(t, span.Logs)
	assert.Equal(t, 0, len(span.Tags))
	assert.Equal(t, 0, len(span.Logs))
}

func TestTraceIDIsValid(t *testing.T) {
	tests := []struct {
		name  string
		tid   TraceID
		valid bool
	}{
		{
			name:  "zero trace ID is invalid",
			tid:   TraceID{High: 0, Low: 0},
			valid: false,
		},
		{
			name:  "low-only trace ID is valid",
			tid:   TraceID{High: 0, Low: 123},
			valid: true,
		},
		{
			name:  "high-only trace ID is valid",
			tid:   TraceID{High: 456, Low: 0},
			valid: true,
		},
		{
			name:  "full trace ID is valid",
			tid:   TraceID{High: 789, Low: 123},
			valid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.valid, tt.tid.IsValid())
		})
	}
}

func TestSpanIDIsValid(t *testing.T) {
	assert.False(t, SpanID(0).IsValid())
	assert.True(t, SpanID(123).IsValid())
}

func TestSpanJSONMarshaling(t *testing.T) {
	now := time.Now().Truncate(time.Second) // Truncate for comparison

	span := &Span{
		TraceID:       TraceID{High: 1, Low: 2},
		SpanID:        SpanID(123),
		ParentSpanID:  SpanID(456),
		OperationName: "test-operation",
		StartTime:     now,
		Duration:      100 * time.Millisecond,
		Tags: []KeyValue{
			{Key: "service", VType: StringType, VStr: "test-service"},
			{Key: "http.status_code", VType: Int64Type, VInt64: 200},
		},
		Process: &Process{
			ServiceName: "test-service",
		},
	}

	// Marshal to JSON
	data, err := json.Marshal(span)
	require.NoError(t, err)

	// Unmarshal back
	var decoded Span
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	// Verify
	assert.Equal(t, span.TraceID, decoded.TraceID)
	assert.Equal(t, span.SpanID, decoded.SpanID)
	assert.Equal(t, span.OperationName, decoded.OperationName)
	assert.Equal(t, len(span.Tags), len(decoded.Tags))
}

func TestNewTrace(t *testing.T) {
	traceID := TraceID{High: 1, Low: 2}
	trace := NewTrace(traceID)

	assert.NotNil(t, trace)
	assert.Equal(t, traceID, trace.TraceID)
	assert.NotNil(t, trace.Spans)
	assert.NotNil(t, trace.Processes)
	assert.Equal(t, 0, len(trace.Spans))
}

func TestKeyValueTypes(t *testing.T) {
	tests := []struct {
		name string
		kv   KeyValue
	}{
		{
			name: "string type",
			kv:   KeyValue{Key: "key", VType: StringType, VStr: "value"},
		},
		{
			name: "int64 type",
			kv:   KeyValue{Key: "count", VType: Int64Type, VInt64: 42},
		},
		{
			name: "float64 type",
			kv:   KeyValue{Key: "ratio", VType: Float64Type, VFloat64: 3.14},
		},
		{
			name: "bool type",
			kv:   KeyValue{Key: "enabled", VType: BoolType, VBool: true},
		},
		{
			name: "binary type",
			kv:   KeyValue{Key: "data", VType: BinaryType, VBinary: []byte{1, 2, 3}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.kv)
			require.NoError(t, err)

			var decoded KeyValue
			err = json.Unmarshal(data, &decoded)
			require.NoError(t, err)

			assert.Equal(t, tt.kv.Key, decoded.Key)
			assert.Equal(t, tt.kv.VType, decoded.VType)
		})
	}
}

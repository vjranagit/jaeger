package model

import (
	"encoding/json"
	"fmt"
	"time"
)

// Span represents a single unit of work in a distributed trace.
// Unlike OTLP protobuf, we use native Go types for simplicity.
type Span struct {
	TraceID       TraceID           `json:"traceId"`
	SpanID        SpanID            `json:"spanId"`
	ParentSpanID  SpanID            `json:"parentSpanId,omitempty"`
	OperationName string            `json:"operationName"`
	References    []Reference       `json:"references,omitempty"`
	Flags         uint32            `json:"flags"`
	StartTime     time.Time         `json:"startTime"`
	Duration      time.Duration     `json:"duration"`
	Tags          []KeyValue        `json:"tags,omitempty"`
	Logs          []Log             `json:"logs,omitempty"`
	Process       *Process          `json:"process,omitempty"`
	ProcessID     string            `json:"processId,omitempty"`
	Warnings      []string          `json:"warnings,omitempty"`
}

// TraceID is a unique identifier for a trace (128-bit)
type TraceID struct {
	High uint64 `json:"high"`
	Low  uint64 `json:"low"`
}

// SpanID is a unique identifier for a span (64-bit)
type SpanID uint64

// Reference represents a relationship between spans
type Reference struct {
	RefType RefType `json:"refType"`
	TraceID TraceID `json:"traceId"`
	SpanID  SpanID  `json:"spanId"`
}

// RefType indicates the type of reference between spans
type RefType string

const (
	// ChildOf indicates a parent-child relationship
	ChildOf RefType = "CHILD_OF"
	// FollowsFrom indicates a sequential relationship
	FollowsFrom RefType = "FOLLOWS_FROM"
)

// KeyValue represents a key-value pair (tag or attribute)
type KeyValue struct {
	Key      string      `json:"key"`
	VType    ValueType   `json:"vType"`
	VStr     string      `json:"vStr,omitempty"`
	VInt64   int64       `json:"vInt64,omitempty"`
	VFloat64 float64     `json:"vFloat64,omitempty"`
	VBool    bool        `json:"vBool,omitempty"`
	VBinary  []byte      `json:"vBinary,omitempty"`
}

// ValueType indicates the type of a KeyValue
type ValueType string

const (
	StringType  ValueType = "string"
	BoolType    ValueType = "bool"
	Int64Type   ValueType = "int64"
	Float64Type ValueType = "float64"
	BinaryType  ValueType = "binary"
)

// Log represents a structured log event within a span
type Log struct {
	Timestamp time.Time  `json:"timestamp"`
	Fields    []KeyValue `json:"fields"`
}

// Process describes the process/service that generated spans
type Process struct {
	ServiceName string     `json:"serviceName"`
	Tags        []KeyValue `json:"tags,omitempty"`
}

// Trace is a collection of spans that share a trace ID
type Trace struct {
	TraceID   TraceID   `json:"traceId"`
	Spans     []*Span   `json:"spans"`
	Processes []*Process `json:"processes,omitempty"`
	Warnings  []string  `json:"warnings,omitempty"`
}

// NewSpan creates a new span with default values
func NewSpan() *Span {
	return &Span{
		StartTime: time.Now(),
		Tags:      make([]KeyValue, 0),
		Logs:      make([]Log, 0),
	}
}

// NewTrace creates a new trace
func NewTrace(traceID TraceID) *Trace {
	return &Trace{
		TraceID:   traceID,
		Spans:     make([]*Span, 0),
		Processes: make([]*Process, 0),
	}
}

// MarshalJSON implements custom JSON marshaling for TraceID
func (t TraceID) MarshalJSON() ([]byte, error) {
	return json.Marshal(t.String())
}

// String converts TraceID to hex string
func (t TraceID) String() string {
	if t.High == 0 {
		return string(SpanID(t.Low).String())
	}
	return string(SpanID(t.High).String()) + string(SpanID(t.Low).String())
}

// String converts SpanID to hex string
func (s SpanID) String() string {
	return fmt.Sprintf("%016x", uint64(s))
}

// IsValid checks if TraceID is valid (non-zero)
func (t TraceID) IsValid() bool {
	return t.High != 0 || t.Low != 0
}

// IsValid checks if SpanID is valid (non-zero)
func (s SpanID) IsValid() bool {
	return s != 0
}

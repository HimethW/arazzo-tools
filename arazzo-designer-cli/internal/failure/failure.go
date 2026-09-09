// Package failure defines the runner's failure vocabulary: a short, stable label naming WHY a step
// failed, carried alongside the human-readable message rather than replacing it.
//
// The message is for a person; the class is for a program. Without a class, a client that wants to
// behave differently per failure has only one option - matching substrings of prose we write and
// occasionally reword - which breaks silently the moment a message changes.
//
// # Adding, renaming or removing a class
//
// This file is the ONE place the vocabulary is defined. A class is attached where the failure is
// CREATED (see Errorf/Wrap), never re-derived later from the message text, so renaming a message can
// never change a class. Add a constant here, add it to All, and tag it at its source.
package failure

import (
	"errors"
	"fmt"
)

// Class names one kind of failure. An empty Class means "unclassified": a failure outside this
// vocabulary, which a client should treat exactly as it treated every failure before classes
// existed.
type Class string

const (
	// AdapterUnsupported - the AsyncAPI document declares a protocol the runner has no adapter for
	// (kafka, amqp, ...). Retrying cannot help; the document or the runner has to change.
	AdapterUnsupported Class = "adapter_unsupported"

	// ConnectFailed - the broker could not be reached: DNS, refused, TLS, or a connect that timed
	// out. Usually worth retrying, since it is commonly transient.
	ConnectFailed Class = "connect_failed"

	// ReceiveTimeout - the subscription was live but no matching message arrived before the step's
	// timeout. Worth retrying only with a longer timeout, or once the sender is known to have sent.
	ReceiveTimeout Class = "receive_timeout"

	// CorrelationUnresolved - the step's `correlationId` expression produced no value, so there is
	// no id to match on. The runner refuses to silently degrade to an unfiltered receive. Retrying
	// cannot help; the expression or the data it reads has to change.
	CorrelationUnresolved Class = "correlation_unresolved"

	// SerializeFailed - the message could not be encoded or decoded: no serializer for the content
	// type, or a payload the serializer rejected. Retrying cannot help.
	SerializeFailed Class = "serialize_failed"
)

// All returns every defined class, so tests and documentation can enumerate the vocabulary instead
// of repeating it.
func All() []Class {
	return []Class{
		AdapterUnsupported,
		ConnectFailed,
		ReceiveTimeout,
		CorrelationUnresolved,
		SerializeFailed,
	}
}

// classified is an error carrying its class. It keeps the original error's message verbatim, so
// tagging a failure never changes what a person reads.
type classified struct {
	class Class
	err   error
}

func (c *classified) Error() string { return c.err.Error() }
func (c *classified) Unwrap() error { return c.err }

// Wrap tags an existing error with a class. The message is unchanged.
func Wrap(class Class, err error) error {
	if err == nil {
		return nil
	}
	return &classified{class: class, err: err}
}

// Errorf builds a classified error, the one-line form of Wrap(class, fmt.Errorf(...)).
func Errorf(class Class, format string, a ...interface{}) error {
	return &classified{class: class, err: fmt.Errorf(format, a...)}
}

// ClassOf reports the class an error was tagged with, following wrapped errors. It returns the empty
// Class for an untagged error, which is the honest answer: this failure is outside the vocabulary.
func ClassOf(err error) Class {
	var c *classified
	if errors.As(err, &c) {
		return c.class
	}
	return ""
}

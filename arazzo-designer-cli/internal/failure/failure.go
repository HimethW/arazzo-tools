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

// The vocabulary. Names are deliberately free of any one protocol or nesting level - a class covers
// the same SITUATION wherever it occurs, so a step-level and a workflow-level instance of "this
// points at something that does not exist" share one name and none of these has to be renamed when
// the next caller needs it. Renaming is the one change that breaks clients silently, since a client
// matching the old string simply stops matching and reports nothing.
const (
	// TargetUnresolved - the step or workflow points at something that does not exist: a channel
	// absent from the AsyncAPI document, an operation no source declares, a workflow that is not in
	// the document. The reference is well-formed; the thing it names is missing. Retrying cannot
	// help - the document or the spec it references has to change.
	TargetUnresolved Class = "target_unresolved"

	// DocumentInvalid - the document is malformed rather than merely pointing somewhere wrong: an
	// action that is neither send nor receive, a channelPath step with no action to give it a
	// direction, a workflow with no steps. Retrying cannot help.
	//
	// Told apart from TargetUnresolved by WHERE the fix is: there, in the referenced spec; here, in
	// this document.
	DocumentInvalid Class = "document_invalid"

	// AdapterUnsupported - the declared protocol has no adapter (kafka, amqp, ...), or no adapter is
	// configured for it. Retrying cannot help; the document or the runner has to change.
	AdapterUnsupported Class = "adapter_unsupported"

	// ConnectFailed - the link to the remote endpoint did not work: DNS, refused, TLS, a connect that
	// timed out, or a publish/subscribe/write that failed on an already-open connection. Covers a
	// message broker and an HTTP server alike. The distinction between "never connected" and
	// "connected, then the operation failed" is left to the message, because the situation and the
	// advice are the same either way. Usually worth retrying, since it is commonly transient.
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

	// CriteriaUnmet - everything worked and the answer was wrong: the request was sent, or the
	// message arrived, and the step's `successCriteria` did not hold. This is the ordinary
	// assertion failure, and the one a CI run sees most - which is exactly why it must be
	// distinguishable from an infrastructure failure. Retrying reproduces it.
	CriteriaUnmet Class = "criteria_unmet"

	// DependencyUnmet - the step never ran, because a prerequisite it declared in `dependsOn` had
	// not completed successfully. Distinct from every class above in a way a report should show:
	// nothing about THIS step failed, and its own correctness is still unknown. The failure to chase
	// is the prerequisite's.
	DependencyUnmet Class = "dependency_unmet"

	// UnsupportedFeature - the document asks for something the spec allows and this runner has not
	// implemented (today: cross-document step `dependsOn`). Not DocumentInvalid: the document is
	// correct, and "fixing" it would be the wrong advice. Kept apart from AdapterUnsupported so a
	// report can separate "no adapter for this protocol" from "this feature does not exist yet" -
	// two different roadmap questions. Retrying cannot help.
	UnsupportedFeature Class = "unsupported_feature"
)

// All returns every defined class, so tests and documentation can enumerate the vocabulary instead
// of repeating it.
func All() []Class {
	return []Class{
		TargetUnresolved,
		DocumentInvalid,
		AdapterUnsupported,
		ConnectFailed,
		ReceiveTimeout,
		CorrelationUnresolved,
		SerializeFailed,
		CriteriaUnmet,
		DependencyUnmet,
		UnsupportedFeature,
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

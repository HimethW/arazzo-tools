package executor

import (
	"errors"
	"testing"

	"github.com/wso2/arazzo-designer-cli/internal/failure"
	"github.com/wso2/arazzo-designer-cli/internal/models"
	"github.com/wso2/arazzo-designer-cli/internal/telemetry"
)

// classSourceDescs declares the sources each class needs to be provoked from a REAL failure rather
// than a hand-built error: a kafka broker with no adapter, and a normal in-memory bus.
func classSourceDescs() map[string]interface{} {
	return map[string]interface{}{
		"orderBus": map[string]interface{}{
			"asyncapi": "3.0.0",
			"channels": map[string]interface{}{
				"orders": map[string]interface{}{"address": "orders/new"},
			},
		},
		"exotic": map[string]interface{}{
			"asyncapi": "3.0.0",
			"servers": map[string]interface{}{
				"cluster": map[string]interface{}{"host": "kafka.example.com:9092", "protocol": "kafka"},
			},
			"channels": map[string]interface{}{
				"events": map[string]interface{}{"address": "events"},
			},
		},
		"typed": map[string]interface{}{
			"asyncapi": "3.0.0",
			"channels": map[string]interface{}{
				"weird": map[string]interface{}{
					"address": "weird",
					"messages": map[string]interface{}{
						"m": map[string]interface{}{"contentType": "application/x-nonesuch"},
					},
				},
			},
		},
	}
}

func classExecutor() *StepExecutor {
	return NewStepExecutor(map[string]interface{}{}, classSourceDescs(), &models.RuntimeParams{}, &telemetry.NoopSink{})
}

// Every class must arrive on the step result from a real failure. These run the actual code paths -
// no error is constructed by the test - so a class that stopped being attached anywhere along the
// way fails here.
func TestFailureClassesReachTheStepResult(t *testing.T) {
	cases := []struct {
		name  string
		step  map[string]interface{}
		want  failure.Class
		setup func(se *StepExecutor, state *models.ExecutionState)
	}{
		{
			name: "kafka has no adapter",
			step: map[string]interface{}{
				"stepId": "emit", "channelPath": "exotic#/channels/events", "action": "send",
				"requestBody": map[string]interface{}{"payload": map[string]interface{}{"a": 1}},
			},
			want: failure.AdapterUnsupported,
		},
		{
			name: "no serializer for the declared content type",
			step: map[string]interface{}{
				"stepId": "emit", "channelPath": "typed#/channels/weird", "action": "send",
				"requestBody": map[string]interface{}{"payload": map[string]interface{}{"a": 1}},
			},
			want: failure.SerializeFailed,
		},
		{
			name: "nothing arrives before the timeout",
			step: map[string]interface{}{
				"stepId": "wait", "channelPath": "orderBus#/channels/orders", "action": "receive",
				"timeout": 50,
			},
			want: failure.ReceiveTimeout,
		},
		{
			name: "the correlationId expression resolves to nothing",
			step: map[string]interface{}{
				"stepId": "wait", "channelPath": "orderBus#/channels/orders", "action": "receive",
				"correlationId": "$inputs.missingToken", "timeout": 50,
			},
			want: failure.CorrelationUnresolved,
		},
		{
			name: "the channel named does not exist in the document",
			step: map[string]interface{}{
				"stepId": "emit", "channelPath": "orderBus#/channels/nosuchchannel", "action": "send",
			},
			want: failure.TargetUnresolved,
		},
		{
			name: "a channelPath step with no action has no direction",
			step: map[string]interface{}{
				"stepId": "emit", "channelPath": "orderBus#/channels/orders",
			},
			want: failure.DocumentInvalid,
		},
		{
			name: "an action that is neither send nor receive",
			step: map[string]interface{}{
				"stepId": "emit", "channelPath": "orderBus#/channels/orders", "action": "sideways",
			},
			want: failure.DocumentInvalid,
		},
		{
			name: "the send worked and successCriteria did not hold",
			step: map[string]interface{}{
				"stepId": "emit", "channelPath": "orderBus#/channels/orders", "action": "send",
				"requestBody": map[string]interface{}{"payload": map[string]interface{}{"n": 1}},
				// Nothing about the send failed; the assertion on it did.
				"successCriteria": []interface{}{
					map[string]interface{}{"condition": "$statusCode == 999"},
				},
			},
			want: failure.CriteriaUnmet,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			se := classExecutor()
			state := models.NewExecutionState("wf", nil, nil, nil)
			if tc.setup != nil {
				tc.setup(se, state)
			}
			r := se.ExecuteStep(tc.step, nil, state)
			if r.Success {
				t.Fatalf("step unexpectedly succeeded; this case must fail to carry a class")
			}
			if r.ErrorClass != string(tc.want) {
				t.Errorf("ErrorClass = %q, want %q (message was: %s)", r.ErrorClass, tc.want, r.Error)
			}
		})
	}
}

// connect_failed needs a broker that refuses, which port 1 does instantly and locally - no network
// and no waiting. It runs through ExecuteStep rather than the adapter alone so the class is checked
// where a caller would actually read it.
func TestConnectFailureReachesTheStepResult(t *testing.T) {
	se := NewStepExecutor(map[string]interface{}{}, map[string]interface{}{
		"deadBus": map[string]interface{}{
			"asyncapi": "3.0.0",
			"servers": map[string]interface{}{
				// Nothing listens on port 1, so the dial is refused rather than timing out.
				"gateway": map[string]interface{}{"host": "127.0.0.1:1", "protocol": "ws"},
			},
			"channels": map[string]interface{}{
				"orders": map[string]interface{}{"address": "orders/new"},
			},
		},
	}, &models.RuntimeParams{}, &telemetry.NoopSink{})

	r := se.ExecuteStep(map[string]interface{}{
		"stepId": "emit", "channelPath": "deadBus#/channels/orders", "action": "send",
		"requestBody": map[string]interface{}{"payload": map[string]interface{}{"a": 1}},
	}, nil, models.NewExecutionState("wf", nil, nil, nil))

	if r.Success {
		t.Fatal("sending to an unreachable broker must fail")
	}
	if r.ErrorClass != string(failure.ConnectFailed) {
		t.Errorf("ErrorClass = %q, want %q (message was: %s)", r.ErrorClass, failure.ConnectFailed, r.Error)
	}
}

// Every runtime failure path is labelled now, so an EMPTY class no longer means "outside the
// vocabulary" - it means a path was added without one. The field must still be able to say "I don't
// know", though, or a client could never tell a real classification from a default. This pins that
// capability at the vocabulary level, where nothing can accidentally erode it.
func TestAnUntaggedErrorHasNoClass(t *testing.T) {
	if got := failure.ClassOf(errors.New("something nobody classified")); got != "" {
		t.Errorf("ClassOf(untagged) = %q, want empty", got)
	}
	if got := failure.ClassOf(nil); got != "" {
		t.Errorf("ClassOf(nil) = %q, want empty", got)
	}
}

// The vocabulary must not grow duplicates or blanks - a repeated value would make two situations
// indistinguishable, which is the one thing a class exists to prevent.
func TestTheVocabularyIsWellFormed(t *testing.T) {
	seen := map[failure.Class]bool{}
	for _, c := range failure.All() {
		if c == "" {
			t.Error("All() contains an empty class")
		}
		if seen[c] {
			t.Errorf("class %q is listed twice", c)
		}
		seen[c] = true
	}
	if len(seen) != len(failure.All()) {
		t.Error("All() has duplicates")
	}
}

// The class must survive a message being reworded, which is the whole reason it is attached at the
// source. This pins the property directly: the class comes from the error's TAG, not its text.
func TestClassIsIndependentOfTheMessage(t *testing.T) {
	original := failure.Errorf(failure.ConnectFailed, "some very specific wording")
	reworded := failure.Errorf(failure.ConnectFailed, "completely different wording")

	if failure.ClassOf(original) != failure.ConnectFailed || failure.ClassOf(reworded) != failure.ConnectFailed {
		t.Error("the class must not depend on the message text")
	}
	if failure.ClassOf(nil) != "" {
		t.Error("a nil error has no class")
	}
}

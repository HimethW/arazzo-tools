// adapter_select.go chooses WHICH transport adapter an async step runs on (Phase 11). Arazzo has no
// broker field — the AsyncAPI document does: its `servers` section declares `protocol` (mqtt, ws, …)
// and `host`. This maps that declaration to an Adapter: ws/wss → WSAdapter, mqtt/mqtts → MQTTAdapter,
// no servers → the default in-memory adapter (tests/local runs). Adapters are cached per
// protocol+host so every step against the same broker shares one connection.
//
// TODO(phase-future): Kafka adapter — implement KafkaAdapter (map channel -> topic, consumer groups,
// auth/TLS) together with REAL Avro/Protobuf codecs + schema-registry config replacing the Phase-10
// schemaRequiredSerializer stubs in serializer.go (Kafka is where those formats are actually used).
package executor

import (
	"fmt"
	"sort"
	"strings"

	"github.com/wso2/arazzo-designer-cli/internal/failure"
)

// The transports an AsyncAPI document can select. These name an adapter WITHOUT building one, so a
// caller can report what a step would run on (the CLI/MCP workflow details) without opening a
// connection.
const (
	TransportInMemory  = "in-memory"
	TransportMQTT      = "mqtt"
	TransportWebSocket = "websocket"
)

// TransportForProtocol maps an AsyncAPI `servers.protocol` to the transport that carries it, or the
// error a step would fail with. An empty protocol means no servers are declared, which is in-memory.
//
// This is the ONE protocol table: adapterFor switches on its result rather than repeating the cases,
// so a describer and a run can never disagree about what a document selects.
func TransportForProtocol(protocol string) (string, error) {
	switch normalizeProtocol(protocol) {
	case "":
		return TransportInMemory, nil
	case "ws", "wss":
		return TransportWebSocket, nil
	case "mqtt", "mqtts", "secure-mqtt":
		return TransportMQTT, nil
	case "kafka", "kafka-secure":
		// TODO(phase-future): Kafka adapter + real Avro/Protobuf serializers (see file comment).
		return "", failure.Errorf(failure.AdapterUnsupported, "the %q protocol is not yet supported: a Kafka adapter (with Avro/Protobuf schema support) is a planned future phase - supported protocols: ws, wss, mqtt, mqtts (and in-memory when no servers are declared)", protocol)
	default:
		return "", failure.Errorf(failure.AdapterUnsupported, "unsupported AsyncAPI server protocol %q - supported: ws, wss, mqtt, mqtts (and in-memory when no servers are declared)", protocol)
	}
}

// normalizeProtocol is the single reading of an AsyncAPI `servers.protocol`: trimmed and lowercased,
// with "" meaning no server was declared. Every protocol decision goes through it so a describer and
// a run cannot disagree over whitespace or case.
func normalizeProtocol(protocol string) string {
	return strings.ToLower(strings.TrimSpace(protocol))
}

// TransportForSource reports the transport an AsyncAPI source description selects, reading the same
// first-server rule adapterFor uses. Exported so the CLI can report a step's transport without
// constructing (and therefore connecting) an adapter.
func TransportForSource(spec map[string]interface{}) (string, error) {
	protocol, _ := firstServer(spec)
	return TransportForProtocol(protocol)
}

// adapterFor picks the transport adapter for a resolved async target from its source's AsyncAPI
// `servers` declaration. With no servers section the default adapter (in-memory) is used, keeping
// Phase 9/10 documents and tests working unchanged. Adapters are cached per protocol+host so every
// step against the same broker shares one connection.
func (se *StepExecutor) adapterFor(info *AsyncInfo) (Adapter, error) {
	protocol, host := firstServer(toMap(se.SourceDescriptions[info.Source]))
	transport, err := TransportForProtocol(protocol)
	if err != nil {
		return nil, err
	}
	if transport == TransportInMemory {
		return se.AsyncAdapter, nil
	}

	key := normalizeProtocol(protocol) + "://" + host
	if a, ok := se.asyncAdapters[key]; ok {
		return a, nil
	}

	var adapter Adapter
	switch transport {
	case TransportWebSocket:
		// normalizeProtocol already reduced this to exactly "ws" or "wss" - the only two protocols
		// TransportForProtocol maps to a websocket - so it doubles as the URL scheme.
		adapter = NewWSAdapter(normalizeProtocol(protocol) + "://" + host)
	case TransportMQTT:
		adapter = NewMQTTAdapter(protocol, host)
	default:
		// Unreachable today: TransportForProtocol returns in-memory, one of the two above, or an
		// error. It exists so that adding a transport there without wiring it here fails loudly
		// instead of caching a nil adapter under this key.
		return nil, fmt.Errorf("no adapter is wired for transport %q", transport)
	}

	if se.asyncAdapters == nil {
		se.asyncAdapters = map[string]Adapter{}
	}
	se.asyncAdapters[key] = adapter
	return adapter, nil
}

// firstServer returns the protocol and host of the AsyncAPI document's first server (by sorted server
// name, so the choice is deterministic — Go map iteration is not). Empty strings mean "no usable
// servers declared".
func firstServer(spec map[string]interface{}) (protocol, host string) {
	servers := toMap(spec["servers"])
	if len(servers) == 0 {
		return "", ""
	}
	names := make([]string, 0, len(servers))
	for name := range servers {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		server := toMap(servers[name])
		p, _ := server["protocol"].(string)
		h, _ := server["host"].(string)
		if strings.TrimSpace(p) != "" && strings.TrimSpace(h) != "" {
			return strings.TrimSpace(p), strings.TrimSpace(h)
		}
	}
	return "", ""
}

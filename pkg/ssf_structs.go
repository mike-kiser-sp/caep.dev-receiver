package pkg

import (
	"github.com/MicahParks/keyfunc/v3"
	"github.com/golang-jwt/jwt"
	event "github.com/mike-kiser-sp/receiver/pkg/ssf_events"
)

// Represents the interface for the SSF receiver with user facing
// methods
type SsfReceiver interface {
	ConfigureCallback(callback func(events []event.SsfEvent), pollInterval int) error

	// Polls the configured receiver a returns a list of the available SSF
	// Events
	PollEvents() ([]event.SsfEvent, error)

	// Cleans up the Receiver's resources and deletes it from the transmitter
	DeleteReceiver()

	// Get stream status from the transmitter
	GetStreamStatus() (StreamStatus, error)

	// Enable the stream
	EnableStream() (StreamStatus, error)

	// Pause the stream
	PauseStream() (StreamStatus, error)

	// Disable the stream
	DisableStream() (StreamStatus, error)

	// Disable the stream
	PrintStream()
}

// The struct that contains all the necessary fields and methods for the
// SSF Receiver's implementation
type SsfReceiverImplementation struct {
	// transmitterUrl stores the base url of the transmitter the
	// receiver will make request to
	transmitterUrl string

	// transmitterTypeRfc determines whether this receiver is push or pull
	transmitterTypeRfc string

	// transmitterPollUrl defines the url that the receiver
	// should hit to receive SSF Events
	transmitterPollUrl string

	// receiverPushUrl defines the url that the receiver
	// should hit to receive SSF Events
	receiverPushUrl string

	// TransmitterStreamUrl defines the URL that the receiver will use
	// to update/get the stream status
	transmitterStatusUrl string

	// eventsRequested contains a list of the SSF Event URI's requested
	// by the receiver
	eventsRequested []string

	// authorizationToken defines the Auth Token used to authorize the
	// receiver with the transmitter
	authorizationToken string

	// pollCallback defines the method the receiver will call to pass
	// events into when the poll interval is triggered
	pollCallback func(events []event.SsfEvent)

	// pollInterval defines the interval, in seconds, between every
	// poll request the receiver will make to the transmitter. After
	// each poll request, the available SSF events will be passed in
	// a function call to pollCallback
	pollInterval int

	// configurationUrl defines the transmitter's configuration url
	configurationUrl string

	// streamId defines the Id of the stream that corresponds to the
	// transmitter
	streamId string

	// terminate is used to stop the push interval routine
	terminate chan bool

	// JWKS from the transmitter
	transmitterJwks keyfunc.Keyfunc
}

// Struct used to read a Transmitter's configuration
type TransmitterConfig struct {
	Issuer                   string                   `json:"issuer"`
	JwksUri                  string                   `json:"jwks_uri,omitempty"`
	DeliveryMethodsSupported []string                 `json:"delivery_methods_supported,omitempty"`
	ConfigurationEndpoint    string                   `json:"configuration_endpoint,omitempty"`
	StatusEndpoint           string                   `json:"status_endpoint,omitempty"`
	AddSubjectEndpoint       string                   `json:"add_subject_endpoint,omitempty"`
	RemoveSubjectEndpoint    string                   `json:"remove_subject_endpoint,omitempty"`
	SpecVersion              string                   `json:"spec_version,omitempty"`
	AuthorizationSchemes     []map[string]interface{} `json:"authorization_schemes,omitempty"`
}

// Struct used to make a Create Stream request for the receiver
type CreateStreamReq struct {
	Delivery        SsfDelivery `json:"delivery"`
	EventsRequested []string    `json:"events_requested"`
	Description     string      `json:"description,omitempty"`
}

// Struct that defines the deliver method for the Create Stream Request
type SsfDelivery struct {
	DeliveryMethod string `json:"method"`
	EndpointUrl    string `json:"endpoint_url,omitempty"`
}

// Struct to make a request to poll SSF Events to the
// configured transmitter
type PollTransmitterRequest struct {
	Acknowledgements  []string `json:"ack"`
	MaxEvents         int      `json:"maxEvents,omitempty"`
	ReturnImmediately bool     `json:"returnImmediately"`
}

// Struct to make subject changes to a stream
type StreamSubjectRequest struct {
	StreamID string `json:"stream_id,omitempty"`
	Subject  SubId  `json:"subject,omitempty"`
	Verified bool   `json:"verified,omitempty"`
}

// Struct to make a request to update the stream status
type UpdateStreamRequest struct {
	StreamId string `json:"stream_id"`
	Status   string `json:"status"`
	Reason   string `json:"reason"`
}

type StreamStatus int

const (
	StreamEnabled StreamStatus = iota + 1
	StreamPaused
	StreamDisabled
)

var StatusEnumMap = map[string]StreamStatus{
	"enabled":  StreamEnabled,
	"paused":   StreamPaused,
	"disabled": StreamDisabled,
}

var EnumToStringStatusMap = map[StreamStatus]string{
	StreamEnabled:  "enabled",
	StreamPaused:   "paused",
	StreamDisabled: "disabled",
}

type StreamConfig struct {
	StreamId        string      `json:"stream_id"`
	Issuer          string      `json:"iss"`
	Audience        string      `json:"aud"`
	EventsSupported []string    `json:"events_supported"`
	EventsRequested []string    `json:"events_requested"`
	EventsDelivered []string    `json:"events_delivered"`
	Delivery        SsfDelivery `json:"delivery"`
	Description     string      `json:"description,omitempty"`
}

type SubId struct {
	Format string `json:"format"`
	Email  string `json:"email"`
}

type SETSessionRevoked struct {
	SubID  SubId `json:"sub_id"`
	Events struct {
		Event struct {
			EventTimestamp int64  `json:"event_timestamp"`
			Reason         string `json:"reason,omitempty"`
			ReasonAdmin    string `json:"reason_admin,omitempty"`
			ReasonUser     string `json:"reason_user,omitempty"`
		} `json:"https://schemas.openid.net/secevent/caep/event-type/session-revoked"`
	} `json:"events"`

	jwt.StandardClaims
}

type SETCredentialChange struct {
	SubID  SubId `json:"sub_id"`
	Events struct {
		Event struct {
			EventTimestamp int64  `json:"event_timestamp"`
			Reason         string `json:"reason,omitempty"`
			CredentialType string `json:"credential_type"`
			ChangeType     string `json:"change_type"`
			ReasonAdmin    string `json:"reason_admin,omitempty"`
			ReasonUser     string `json:"reason_user,omitempty"`
		} `json:"https://schemas.openid.net/secevent/caep/event-type/credential-change"`
	} `json:"events"`

	jwt.StandardClaims
}

type EventSet struct {
}

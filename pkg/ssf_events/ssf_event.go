package ssf_events

import (
	"errors"
	"fmt"
	"log"
	"time"
)

type EventType int

const (
	SessionRevoked EventType = iota
	CredentialChange
	DeviceCompliance
	AssuranceLevelChange
	TokenClaimsChange
	VerificationEventType
	StreamUpdatedEventType
)

type SubjectFormat int

const (
	Account SubjectFormat = iota
	Email
	IssuerAndSubject
	Opaque
	PhoneNumber
	DecentralizedIdentifier
	UniqueResourceIdentifier
	Aliases
	ComplexSubject
)

const AccountSubjectFormat = "account"
const EmailSubjectFormat = "email"
const IssuerAndSubjectFormat = "iss_sub"
const OpaqueSubjectFormat = "opaque"
const PhoneNumberSubjectFormat = "phone_number"
const DecentralizedIdentifierSubjectFormat = "did"
const UniqueResourceIdentifierSubjectFormat = "uri"
const AliasesSubjectFormat = "aliases"

// Represents the interface that all SSF Events should implement
//
// See the SessionRevokedEvent (./events/session_revoked_event.go)
// for an example
type SsfEvent interface {
	// Returns the Event URI for the given event
	GetEventUri() string

	// Returns the format of the event's subject
	GetSubjectFormat() SubjectFormat

	// Returns the subject of the event
	GetSubject() map[string]interface{}

	// Returns the Unix timestamp of the event
	GetTimestamp() int64

	// Return the type of event
	GetType() EventType
}

var EventUri = map[EventType]string{
	SessionRevoked:         "https://schemas.openid.net/secevent/caep/event-type/session-revoked",
	CredentialChange:       "https://schemas.openid.net/secevent/caep/event-type/credential-change",
	DeviceCompliance:       "https://schemas.openid.net/secevent/caep/event-type/device-compliance-change",
	AssuranceLevelChange:   "https://schemas.openid.net/secevent/caep/event-type/assurance-level-change",
	TokenClaimsChange:      "https://schemas.openid.net/secevent/caep/event-type/token-claims-change",
	VerificationEventType:  "https://schemas.openid.net/secevent/caep/event-type/verification-event",
	StreamUpdatedEventType: "https://schemas.openid.net/secevent/caep/event-type/stream-updated",
}

var EventEnum = map[string]EventType{
	"https://schemas.openid.net/secevent/caep/event-type/session-revoked":          SessionRevoked,
	"https://schemas.openid.net/secevent/caep/event-type/credential-change":        CredentialChange,
	"https://schemas.openid.net/secevent/caep/event-type/device-compliance-change": DeviceCompliance,
	"https://schemas.openid.net/secevent/caep/event-type/assurance-level-change":   AssuranceLevelChange,
	"https://schemas.openid.net/secevent/caep/event-type/token-claims-change":      TokenClaimsChange,
	"https://schemas.openid.net/secevent/caep/event-type/verification-event":       VerificationEventType,
	"https://schemas.openid.net/secevent/caep/event-type/stream-updated":           StreamUpdatedEventType,
}

// Takes an event subject from the JSON of an SSF Event, and converts it into the matching struct for that event
func EventStructFromEvent(eventUri string, eventSubject interface{}, eventDetails interface{}, claimsJson map[string]interface{}) (SsfEvent, error) {
	eventEnum := EventEnum[eventUri]
	eventAttributes, ok := eventDetails.(map[string]interface{})
	subIdAttributes, ok := eventSubject.(map[string]interface{})

	log.Println("instide construct event")
	// Special Event Types
	if eventEnum == VerificationEventType {
		state, ok := eventAttributes["state"].(string)
		if !ok {
			return nil, errors.New("unable to parse state")
		}

		event := VerificationEvent{
			Json:  claimsJson,
			State: state,
		}
		return &event, nil

	} else if eventEnum == StreamUpdatedEventType {
		status, ok := eventAttributes["status"].(string)
		if !ok {
			return nil, errors.New("unable to parse state")
		}

		reason, _ := eventAttributes["reason"].(string)

		event := StreamUpdatedEvent{
			Json:   claimsJson,
			Status: status,
			Reason: reason,
		}
		return &event, nil
	}
	log.Println("after event switch ")
	log.Println("subjAttrs: ", eventAttributes)
	// caep dev treats this as a string, others should see it as an int
	//	timestamp, err := strconv.ParseInt(eventAttributes["event_timestamp"].(float64), 10, 64)
	timestamp := time.Now().Unix()
	if eventAttributes["event_timestamp"] != nil {
		timestamp = int64(eventAttributes["event_timestamp"].(float64))
	}

	if !ok {
		return nil, errors.New("unable to parse event subject")
	}
	log.Println("before get subj format")
	log.Println(subIdAttributes["sub_id"])
	format, err := GetSubjectFormat(subIdAttributes["sub_id"].(map[string]interface{}))
	if err != nil {
		return nil, err
	}
	log.Println("after get subj format")

	// Add more Ssf Events as desired
	switch eventEnum {
	case CredentialChange:
		credentialType, ok := eventAttributes["credentialType"].(float64)
		if !ok {
			return nil, errors.New("unable to parse credential type")
		}

		changeType, ok := eventAttributes["changeType"].(float64)
		if !ok {
			return nil, errors.New("unable to parse credential type")
		}

		event := CredentialChangeEvent{
			Json:           claimsJson,
			Format:         format,
			Subject:        subIdAttributes["sub_id"].(map[string]interface{}),
			EventTimestamp: timestamp,
			CredentialType: CredentialTypeEnumMap[uint64(credentialType)],
			ChangeType:     ChangeTypeEnumMap[uint64(changeType)],
		}
		return &event, nil

	case SessionRevoked:
		event := SessionRevokedEvent{
			Json:           claimsJson,
			Format:         format,
			Subject:        subIdAttributes["sub_id"].(map[string]interface{}),
			EventTimestamp: timestamp,
		}
		return &event, nil

	case DeviceCompliance:
		previousStatus, ok := eventAttributes["previousStatus"].(string)
		if !ok {
			return nil, errors.New("unable to parse previous status")
		}

		currentStatus, ok := eventAttributes["currentStatus"].(string)
		if !ok {
			return nil, errors.New("unable to parse current status")
		}

		event := DeviceComplianceEvent{
			Json:           claimsJson,
			Format:         format,
			Subject:        subIdAttributes["sub_id"].(map[string]interface{}),
			EventTimestamp: timestamp,
			PreviousStatus: previousStatus,
			CurrentStatus:  currentStatus,
		}
		return &event, nil

	case AssuranceLevelChange:
		previousLevel, _ := eventAttributes["previousLevel"].(string)
		changeDirection, _ := eventAttributes["changeDirection"].(string)

		currentLevel, ok := eventAttributes["currentLevel"].(string)
		if !ok {
			return nil, errors.New("unable to parse current level")
		}

		namespace, ok := eventAttributes["namespace"].(string)
		if !ok {
			return nil, errors.New("unable to parse namespace")
		}

		event := AssuranceLevelChangeEvent{
			Json:            claimsJson,
			Format:          format,
			Subject:         subIdAttributes["sub_id"].(map[string]interface{}),
			EventTimestamp:  timestamp,
			Namespace:       namespace,
			PreviousLevel:   &previousLevel,
			CurrentLevel:    currentLevel,
			ChangeDirection: &changeDirection,
		}
		return &event, nil

	case TokenClaimsChange:
		claims, ok := eventAttributes["claims"].(map[string]interface{})
		if !ok {
			return nil, errors.New("unable to parse claims")
		}

		event := TokenClaimsChangeEvent{
			Json:           claimsJson,
			Format:         format,
			Subject:        subIdAttributes["sub_id"].(map[string]interface{}),
			EventTimestamp: timestamp,
			Claims:         claims,
		}
		return &event, nil

	default:
		return nil, errors.New("no matching events")
	}
}

func GetSubjectFormat(subject map[string]interface{}) (SubjectFormat, error) {
	format, formatFound := subject["format"]
	formatString := fmt.Sprintf("%v", format)
	if !formatFound {
		return ComplexSubject, nil
	}

	switch formatString {
	case AccountSubjectFormat:
		return Account, nil
	case EmailSubjectFormat:
		return Email, nil
	case IssuerAndSubjectFormat:
		return IssuerAndSubject, nil
	case OpaqueSubjectFormat:
		return Opaque, nil
	case PhoneNumberSubjectFormat:
		return PhoneNumber, nil
	case DecentralizedIdentifierSubjectFormat:
		return DecentralizedIdentifier, nil
	case UniqueResourceIdentifierSubjectFormat:
		return UniqueResourceIdentifier, nil
	case AliasesSubjectFormat:
		return Aliases, nil
	default:
		return -1, errors.New("unable to determine subject format")
	}
}

// Converts a list of Ssf Events to a list of their corresponding Event URI's
func EventTypeArrayToEventUriArray(events []EventType) []string {
	var eventUriArr []string
	for i := 0; i < len(events); i++ {
		eventUriArr = append(eventUriArr, EventUri[events[i]])
	}
	return eventUriArr
}

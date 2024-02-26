package pkg

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/httplog/v2"
	"github.com/go-chi/jwtauth/v5"
	"github.com/go-chi/render"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"io"
	"log"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	keyfunc "github.com/MicahParks/keyfunc/v3"
	"github.com/golang-jwt/jwt/v5"
	events "github.com/mike-kiser-sp/receiver/pkg/ssf_events"
)

const TransmitterConfigMetadataPath = "/.well-known/ssf-configuration"
const TransmitterPollRFC = "urn:ietf:rfc:8936"
const TransmitterPushRFC = "urn:ietf:rfc:8935"

const protocol = "http"
const hostName = "localhost"
const addr = ":9425"
const specVersion = "1_0-ID2"

const pushEventsUrl = "/ssf/push"

var tokenAuth *jwtauth.JWTAuth

var mainReceiver SsfReceiverImplementation

// Initializes the SSF Receiver based on the specified configuration.
//
// Returns an error if any process of configuring the receiver, registering
// it with the transmitter, or setting up the poll interval failed
func ConfigureSsfReceiver(cfg ReceiverConfig, streamId string) (SsfReceiver, error) {
	if cfg.TransmitterUrl == "" || len(cfg.EventsRequested) == 0 || cfg.AuthorizationToken == "" {
		return nil, errors.New("Receiver Config - missing required field")
	}

	transmitterUrl, err := url.Parse(cfg.TransmitterUrl)
	if err != nil {
		return nil, err
	}

	baseUrl := transmitterUrl.Host
	trailingPath := transmitterUrl.Path

	log.Println(baseUrl)
	log.Println(trailingPath)
	log.Println(TransmitterConfigMetadataPath)

	transmitterConfigEndpoint := ""
	if strings.Contains(baseUrl, "localhost") {
		transmitterConfigEndpoint = "http://" + baseUrl
	} else {
		transmitterConfigEndpoint = "https://" + baseUrl
	}
	if trailingPath != "/" {
		transmitterConfigEndpoint += trailingPath + TransmitterConfigMetadataPath
	} else {
		transmitterConfigEndpoint += TransmitterConfigMetadataPath
	}
	log.Println("url:", transmitterConfigEndpoint)
	transmitterCfg, err := makeTransmitterConfigRequest(transmitterConfigEndpoint)
	if err != nil {
		return nil, err
	}

	if transmitterCfg.ConfigurationEndpoint == "" {
		return nil, errors.New("Given transmitter doesn't specify the configuration endpoint")
	}

	key, err := keyfunc.NewDefault([]string{transmitterCfg.JwksUri})
	if err != nil {
		log.Fatalf("Failed to create a keyfunc.Keyfunc from the server's URL.\nError: %s", err)
	}

	var receiver SsfReceiverImplementation

	if cfg.TransmitterTypeRfc == TransmitterPollRFC {
		var pollUrl = ""

		if streamId != "" {
			streamId, pollUrl, err = getStreamConfig(transmitterCfg.ConfigurationEndpoint, cfg, streamId)
			if err != nil {
				return nil, err
			}
		} else {
			streamId, pollUrl, err = makeCreateStreamRequest(transmitterCfg.ConfigurationEndpoint, cfg)
			if err != nil {
				return nil, err
			}
		}
		receiver = SsfReceiverImplementation{
			transmitterUrl:       cfg.TransmitterUrl,
			transmitterPollUrl:   pollUrl,
			eventsRequested:      events.EventTypeArrayToEventUriArray(cfg.EventsRequested),
			authorizationToken:   cfg.AuthorizationToken,
			transmitterStatusUrl: transmitterCfg.StatusEndpoint,
			pollInterval:         300,
			streamId:             streamId,
			configurationUrl:     transmitterCfg.ConfigurationEndpoint,
			transmitterJwks:      key,
		}
		if cfg.PollInterval != 0 {
			receiver.pollInterval = cfg.PollInterval
		}

		if cfg.PollCallback != nil {
			receiver.pollCallback = cfg.PollCallback
			receiver.InitPollInterval()
		}
		mainReceiver = receiver

	} else {

		var pushUrl = ""
		if streamId != "" {
			streamId, pushUrl, err = getStreamConfig(transmitterCfg.ConfigurationEndpoint, cfg, streamId)
			if err != nil {
				return nil, err
			}
		} else {

			// make a streamID (receiver side id)
			Id, _ := uuid.NewRandom()
			IdString := Id.String()
			streamId := strings.Replace(IdString, "-", "", -1)

			// make a receiver url also
			cfg.TransmitterPushUrl = cfg.TransmitterPushUrl + "/" + streamId

			streamId, pushUrl, err = makeCreateStreamRequest(transmitterCfg.ConfigurationEndpoint, cfg)
			if err != nil {
				return nil, err
			}
		}
		receiver = SsfReceiverImplementation{
			transmitterUrl:       cfg.TransmitterUrl,
			receiverPushUrl:      pushUrl,
			eventsRequested:      events.EventTypeArrayToEventUriArray(cfg.EventsRequested),
			authorizationToken:   cfg.AuthorizationToken,
			transmitterStatusUrl: transmitterCfg.StatusEndpoint,
			pollInterval:         300,
			streamId:             streamId,
			configurationUrl:     transmitterCfg.ConfigurationEndpoint,
			transmitterJwks:      key,
		}

		if cfg.PollCallback != nil {
			receiver.pollCallback = cfg.PollCallback
		}

		// need to start listening on the url
		log.Println(receiver)
		fmt.Printf("Starting server on %v\n", addr)
		mainReceiver = receiver
		http.ListenAndServe(addr, router())

	}

	return &receiver, nil
}

// Makes the Transmitter Configuration Metadata request to determine
// the transmitter's configuration url for creating a stream
func makeTransmitterConfigRequest(url string) (*TransmitterConfig, error) {
	client := &http.Client{}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	response, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}

	log.Println("before unmarshall")
	log.Println(string(body))

	var configMetadata TransmitterConfig
	err = json.Unmarshal(body, &configMetadata)
	if err != nil {
		return nil, err
	}

	return &configMetadata, nil
}

func getStreamConfig(url string, cfg ReceiverConfig, streamId string) (string, string, error) {
	client := &http.Client{}

	//add stream id to end of request
	getStreamUrl := url + "?stream_id=" + streamId

	req, err := http.NewRequest("GET", getStreamUrl, nil)
	if err != nil {
		return "", "", err
	}

	req.Header.Set("Authorization", "Bearer "+cfg.AuthorizationToken)
	req.Header.Set("Content-Type", "application/json")

	response, err := client.Do(req)
	if err != nil {
		log.Println("failed create of req to get config")
		return "", "", err
	}

	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	type Stream struct {
		StreamId string      `json:"stream_id"`
		Iss      string      `json:"iss"`
		Aud      string      `json:"aud"`
		Delivery SsfDelivery `json:"delivery"`
	}

	println("response:", string(body))

	var stream Stream
	err = json.Unmarshal(body, &stream)
	if err != nil {
		return "", "", err
	}

	println("streamid: ", stream.StreamId)
	println("iss: ", stream.Iss)
	println("aud: ", stream.Aud)
	println("method: ", stream.Delivery.DeliveryMethod)
	println(" endpoint_url:", stream.Delivery.EndpointUrl)

	return stream.StreamId, stream.Delivery.EndpointUrl, nil
}

// Makes the Create Stream Request to the transmitter
func makeCreateStreamRequest(url string, cfg ReceiverConfig) (string, string, error) {
	client := &http.Client{}

	var delivery SsfDelivery

	if cfg.TransmitterTypeRfc == TransmitterPollRFC {
		delivery = SsfDelivery{DeliveryMethod: cfg.TransmitterTypeRfc}
	} else {
		delivery = SsfDelivery{DeliveryMethod: cfg.TransmitterTypeRfc, EndpointUrl: cfg.TransmitterPushUrl}
	}

	createStreamRequest := CreateStreamReq{
		Delivery:        delivery,
		EventsRequested: events.EventTypeArrayToEventUriArray(cfg.EventsRequested),
	}
	log.Println("input to create stream: ", createStreamRequest)

	requestBody, err := json.Marshal(createStreamRequest)
	if err != nil {
		return "", "", err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(requestBody))
	if err != nil {
		return "", "", err
	}

	req.Header.Set("Authorization", "Bearer "+cfg.AuthorizationToken)
	req.Header.Set("Content-Type", "application/json")

	response, err := client.Do(req)
	if err != nil {
		log.Println("failed create of req")
		return "", "", err
	}

	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)

	println("response:", string(body))

	var stream StreamConfig
	err = json.Unmarshal(body, &stream)
	if err != nil {
		return "", "", err
	}

	println("streamid: ", stream.StreamId)
	println("iss: ", stream.Issuer)
	println("aud: ", stream.Audience)
	println("method: ", stream.Delivery.DeliveryMethod)
	println(" endpoint_url:", stream.Delivery.EndpointUrl)

	return stream.StreamId, stream.Delivery.EndpointUrl, nil
}

// Initializes the poll interval for the receiver that will intermittently
// send SSF Events to the specified callback function
func (receiver *SsfReceiverImplementation) InitPollInterval() {
	// Create a channel to listen for quit signals
	receiver.terminate = make(chan bool)

	// Start a Goroutine to run the request on a schedule
	go func() {
		for {
			select {
			case <-receiver.terminate:
				return
			default:
				println("Polling for Events")
				events, err := receiver.PollEvents()
				if err == nil {
					receiver.pollCallback(events)
				} else {
					// TODO: What to do on error?
					panic(err)
				}
				time.Sleep(time.Duration(receiver.pollInterval) * time.Second)
			}
		}
	}()
}

// TODO: Not Yet Implemented
func (receiver *SsfReceiverImplementation) ConfigureCallback(callback func(events []events.SsfEvent), pollInterval int) error {
	return nil
}

// Polls the transmitter for all available SSF Events, returning them as a list
// for use
func (receiver *SsfReceiverImplementation) PollEvents() ([]events.SsfEvent, error) {
	client := &http.Client{}
	pollRequest := PollTransmitterRequest{Acknowledgements: []string{}, MaxEvents: 10, ReturnImmediately: true}
	requestBody, err := json.Marshal(pollRequest)
	if err != nil {
		return []events.SsfEvent{}, err
	}

	log.Println("***************\n\n", string(requestBody), "\n*********\n\n")

	req, err := http.NewRequest("POST", receiver.transmitterPollUrl, bytes.NewBuffer(requestBody))
	if err != nil {
		return []events.SsfEvent{}, err
	}

	req.Header.Set("Authorization", "Bearer "+receiver.authorizationToken)
	req.Header.Set("Content-Type", "application/json")

	response, err := client.Do(req)
	if err != nil {
		return []events.SsfEvent{}, err
	}

	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return []events.SsfEvent{}, err
	}

	log.Println("***************\n\n", body, "\n*********\n\n")

	if response.StatusCode != 200 && response.StatusCode != 202 {
		return []events.SsfEvent{}, err
	}

	type SsfEventSets struct {
		Sets map[string]string `json:"sets"`
	}

	var ssfEventsSets SsfEventSets
	err = json.Unmarshal(body, &ssfEventsSets)
	if err != nil {
		return []events.SsfEvent{}, nil
	}

	//turn back on acks once testing is done

	if len(ssfEventsSets.Sets) > 0 {
		err = acknowledgeEvents(&ssfEventsSets.Sets, receiver)
		if err != nil {
			return []events.SsfEvent{}, nil
		}
	}

	events, err := parseSsfEventSets(&ssfEventsSets.Sets, receiver.transmitterJwks)
	return events, err
}

// Cleans up the resources used by the Receiver and deletes the Receiver's
// stream from the transmitter
func (receiver *SsfReceiverImplementation) DeleteReceiver() {
	receiver.terminate <- true

	client := &http.Client{}
	req, err := http.NewRequest("DELETE", receiver.configurationUrl+"?stream_id="+receiver.streamId, nil)
	if err != nil {
		panic(err)
	}

	req.Header.Set("Authorization", receiver.authorizationToken)

	_, err = client.Do(req)
	if err != nil {
		panic(err)
	}
}

func (receiver *SsfReceiverImplementation) EnableStream() (StreamStatus, error) {
	if receiver.transmitterStatusUrl == "" {
		return 0, errors.New("configured receiver does not have transmitter stream url")
	}
	return receiver.sendStatusUpdateRequest(StreamEnabled)
}

func (receiver *SsfReceiverImplementation) PauseStream() (StreamStatus, error) {
	if receiver.transmitterStatusUrl == "" {
		return 0, errors.New("configured receiver does not have transmitter stream url")
	}
	return receiver.sendStatusUpdateRequest(StreamPaused)
}

func (receiver *SsfReceiverImplementation) DisableStream() (StreamStatus, error) {
	if receiver.transmitterStatusUrl == "" {
		return 0, errors.New("configured receiver does not have transmitter stream url")
	}
	return receiver.sendStatusUpdateRequest(StreamDisabled)
}
func (receiver *SsfReceiverImplementation) PrintStream() {
	print("\n\n****Current Stream****\n")
	print("transmitterURL: ", receiver.transmitterUrl, "\n")
	print("transmitterPollUrL: ", receiver.transmitterPollUrl, "\n")
	print("transmitterStatusUrl: ", receiver.transmitterStatusUrl, "\n")
	print("eventsRequested: ")
	fmt.Println(receiver.eventsRequested)
	print("streamID: ", receiver.streamId, "\n")
	print("\n\n**************\n")
}

func (receiver *SsfReceiverImplementation) sendStatusUpdateRequest(streamStatus StreamStatus) (StreamStatus, error) {
	client := &http.Client{}
	updateStreamRequest := UpdateStreamRequest{StreamId: receiver.streamId, Status: EnumToStringStatusMap[streamStatus]}
	requestBody, err := json.Marshal(updateStreamRequest)
	if err != nil {
		return 0, err
	}

	req, err := http.NewRequest("POST", receiver.transmitterStatusUrl, bytes.NewBuffer(requestBody))
	if err != nil {
		return 0, err
	}

	req.Header.Set("Authorization", "Bearer "+receiver.authorizationToken)
	req.Header.Set("Content-Type", "application/json")

	response, err := client.Do(req)
	if err != nil {
		return 0, err
	}

	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return 0, err
	}
	type StatusResponse struct {
		Status string `json:"status"`
		Reason string `json:"reason,omitempty"`
	}

	var statusResponse StatusResponse
	err = json.Unmarshal(body, &statusResponse)
	if err != nil {
		return 0, err
	}
	return StatusEnumMap[statusResponse.Status], nil
}

func (receiver *SsfReceiverImplementation) GetStreamStatus() (StreamStatus, error) {
	if receiver.transmitterStatusUrl == "" {
		return 0, errors.New("transmitter does not support stream status")
	}

	client := &http.Client{}
	streamUrl := fmt.Sprintf("%s?stream_id=%s", receiver.transmitterStatusUrl, receiver.streamId)
	req, err := http.NewRequest("GET", streamUrl, nil)
	if err != nil {
		return 0, err
	}

	req.Header.Set("Authorization", "Bearer "+receiver.authorizationToken)
	req.Header.Set("Content-Type", "application/json")

	response, err := client.Do(req)
	if err != nil {
		return 0, err
	}

	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return 0, err
	}
	type StatusResponse struct {
		Status string `json:"status"`
	}

	var statusResponse StatusResponse
	err = json.Unmarshal(body, &statusResponse)
	if err != nil {
		return 0, nil
	}

	return StatusEnumMap[statusResponse.Status], nil
}

// Method to acknowledge a list of JTI's (unique ids for each SSF Event) with the
// transmitter so the events are re-transmitted
func acknowledgeEvents(sets *map[string]string, receiver *SsfReceiverImplementation) error {
	ackList := make([]string, len(*sets))
	i := 0
	for jti := range *sets {
		ackList[i] = jti
		i++
	}

	client := &http.Client{}
	pollRequest := PollTransmitterRequest{Acknowledgements: ackList, MaxEvents: 0, ReturnImmediately: true}
	requestBody, err := json.Marshal(pollRequest)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", receiver.transmitterPollUrl, bytes.NewBuffer(requestBody))
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bearer "+receiver.authorizationToken)
	req.Header.Set("Content-Type", "application/json")

	_, err = client.Do(req)
	if err != nil {
		return err
	}

	return nil
}

// Parses a list of JTI:JWT pairings, return a list of the SSF Events from the JWT's
func parseSsfEventSets(sets *map[string]string, k keyfunc.Keyfunc) ([]events.SsfEvent, error) {
	var ssfEventsList []events.SsfEvent

	log.Println("in parse events")

	for _, set := range *sets {
		log.Println("\n\n\n\nnext set:   ", string(set))
		log.Println("keyfunc: ***", k.Keyfunc, "****")
		token, err := jwt.Parse(set, k.Keyfunc)
		if err != nil {
			log.Println(err)
		}
		log.Println(token.Claims)
		iss, err2 := token.Claims.GetIssuer()
		log.Println("iss:", iss)
		if err2 == nil {
		}
		if (err != nil) && (!strings.Contains(iss, "caep.dev") || (!strings.Contains(iss, "sgnl.ai"))) {
			log.Println(err)
			//log.Fatalf("Failed to parse the JWT.\nError: %s", err)
		}

		/*		token, err := jwt.Parse(set, func(token *jwt.Token) (interface{}, error) { return jwt.UnsafeAllowNoneSignatureType, nil })
				if err != nil {
					// turn this back on after validation is done correctly
					//return []events.SsfEvent{}, err
					print("Validation failed for JWT - trust is assumed)")
				}
		*/

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			return []events.SsfEvent{}, errors.New("Can't get JWT Claims")
		}

		eventSubject := make(map[string]interface{})
		eventSubject["sub_id"] = claims["sub_id"]
		log.Println("subject is: ", eventSubject)
		ssfEvents := claims["events"].(map[string]interface{})
		log.Println("ORIGINAL running list: ", ssfEvents)
		for eventType, eventDetails := range ssfEvents {
			log.Println("eventType:", eventType, "       eventSubject:", eventSubject)
			ssfEvent, err := events.EventStructFromEvent(eventType, eventSubject, eventDetails, claims)
			if err != nil {
				log.Println("error", err)
			}
			log.Println("running list: ", ssfEvents)
			log.Println("new Event: ", ssfEvent)
			ssfEventsList = append(ssfEventsList, ssfEvent)
		}

	}

	return ssfEventsList, nil
}

func receiveEvent(w http.ResponseWriter, r *http.Request) {

	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Println(err)
	}
	var event string
	err = json.Unmarshal(body, &event)
	if err != nil {
		log.Println(err)
	}

	sets := map[string]string{
		"event": event,
	}

	log.Println("main receiver: ", mainReceiver.transmitterJwks)
	events, err := parseSsfEventSets(&sets, mainReceiver.transmitterJwks)
	log.Println("after parse events")
	log.Println(err)
	mainReceiver.pollCallback(events)

}

func router() http.Handler {

	err := godotenv.Load(".serverenv")
	if err != nil {
		log.Fatalf("Some error occured. Err: %s", err)
	}

	tokenAuth = jwtauth.New("HS256", []byte(os.Getenv("SECRET")), nil)
	// Service
	// Logger
	logger := httplog.NewLogger("httplog-example", httplog.Options{
		LogLevel: slog.LevelDebug,
		// JSON:             true,
		Concise: true,
		// RequestHeaders:   true,
		// ResponseHeaders:  true,
		MessageFieldName: "message",
		LevelFieldName:   "severity",
		TimeFieldFormat:  time.RFC3339,
		Tags: map[string]string{
			"version": "v1.0-81aa4244d9fc8076a",
			"env":     "dev",
		},
		QuietDownRoutes: []string{
			"/",
			"/ping",
		},
		QuietDownPeriod: 10 * time.Second,
		// SourceFieldName: "source",
	})
	r := chi.NewRouter()
	r.Use(httplog.RequestLogger(logger, []string{"/ping"}))
	r.Use(middleware.Heartbeat("/ping"))
	r.Use(render.SetContentType(render.ContentTypeJSON))

	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	})

	// Protected routes
	r.Group(func(r chi.Router) {
		// Seek, verify and validate JWT tokens
		r.Use(jwtauth.Verifier(tokenAuth))

		// Handle valid / invalid tokens. In this example, we use
		// the provided authenticator middleware, but you can write your
		// own very easily, look at the Authenticator method in jwtauth.go
		// and tweak it, it's not scary.
		r.Use(jwtauth.Authenticator(tokenAuth))

		r.Get("/admin", func(w http.ResponseWriter, r *http.Request) {
			_, claims, _ := jwtauth.FromContext(r.Context())
			w.Write([]byte(fmt.Sprintf("protected area. hi %v", claims["user_id"])))
		})

	})

	// Public routes
	r.Group(func(r chi.Router) {
		r.Route(pushEventsUrl, func(r chi.Router) {
			r.Post("/{StreamID}", receiveEvent)
		})
	})

	return r
}

# A Shared Signals Framework Receiver in Go

An open-source Go OpenID Shared Signals Framework Receiver.

## Using the caep.dev open source receiver
To use this receiver:

### Step 1: Import
simply add the following lines to your Go files:

~~~ go
import {
  "github.com/mike-kiser-sp/receiver/pkg"
  "github.com/mike-kiser-sp/receiver/pkg/ssf-events"
}
~~~

### Step 2: Instantiate the Receiver

~~~ go
  // Configure the receiver (do not specify poll callback if polling is not required yet)
  receiverConfig := pkg.ReceiverConfig{
  	TransmitterUrl:     "<your transmitter url here>",
  	TransmitterPollUrl: "<your transmitter poll url here>",
  	EventsRequested:    []events.EventType{0},
  	AuthorizationToken: "<your access token here>",
	PollCallback:       nil, // no polling
	// push vs poll 
	TransmitterTypeRfc: pkg.TransmitterPollRFC,
 	// TransmitterTypeRfc: pkg.TransmitterPushRFC,
	// if push (see above), then specify push url . . 
	// TransmitterPushUrl: "https://shared-signals.dev/ssf/push",
	// if poll (see above), specify the Poll Call Back Method
	PollCallback: printEvents,
	PollInterval: 20,
  }

  // Initialize the receiver but do not start polling
  receiver, err := pkg.ConfigureSsfReceiver(receiverConfig)
  if err != nil {
  	print(err)
  }

~~~


### Step 3: Poll Events on Demand

~~~ go
  events, err := receiver.PollEvents()
  if err != nil {
  	print(err)
  }
~~~

### Step 4: Parse the Received Event
Use the struct `SsfEvent` to parse events that you receive. For example:

~~~ go
  fmt.Printf("Subject Format: %v\n", event.GetSubjectFormat())
  fmt.Printf("Subject: %v\n", event.GetSubject())
  fmt.Printf("Timestamp: %d\n", event.GetTimestamp())
~~~

You can also configure the Receiver to periodically poll the Transmitter.

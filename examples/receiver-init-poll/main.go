package main

import (
	"fmt"
	"time"

	"github.com/sgnl-ai/caep.dev-receiver/pkg"
	events "github.com/sgnl-ai/caep.dev-receiver/pkg/ssf_events"
)

func main() {
	// Configure the receiver (specify the poll callback function and poll interval to start polling in each interval)
	receiverConfig := pkg.ReceiverConfig{
		TransmitterUrl:     "https://ssf.stg.caep.dev",
		TransmitterPollUrl: "https://ssf.stg.caep.dev/ssf/streams/poll",
		EventsRequested:    []events.EventType{0},
		AuthorizationToken: "f843a2ce-4e94-48d4-aed6-c1617024b245",
		PollCallback:       PrintEvents,
		PollInterval:       20,
	}

	// Initialize the receiver and start polling
	receiver, err := pkg.ConfigureSsfReceiver(receiverConfig)
	if err != nil {
		print(err)
	}

	time.Sleep(time.Duration(90) * time.Second)
	// Delete the receiver after done
	receiver.DeleteReceiver()
}

func PrintEvents(events []events.SsfEvent) {
	fmt.Printf("Number of events: %d\n", len(events))
	for _, event := range events {
		fmt.Println("--------EVENT-------")
		fmt.Printf("Subject Format: %v\n", event.GetSubjectFormat())
		fmt.Printf("Subject: %v\n", event.GetSubject())
		fmt.Printf("Timestamp: %d\n", event.GetTimestamp())
		fmt.Println("--------------------")
	}
	fmt.Print("\n\n")
}

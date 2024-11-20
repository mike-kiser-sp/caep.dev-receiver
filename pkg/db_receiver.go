package pkg

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"strings"
)

func addStreamToDb(db sql.DB, stream_id string, audience_id string, stream_method string, stream_status string, stream_statusReason string, stream_data string) {

	// Insert into streams table
	statement, err := db.Prepare("INSERT INTO streams (client_id, stream_id, stream_method, stream_status, stream_statusReason, stream_data) VALUES (?, ?, ?, ?, ?, ?)")
	if err != nil {
		log.Println("Error in inserting to stream table", err)
	} else {
		log.Println("Successfully inserted into table streams!")
	}
	_, err = statement.Exec(audience_id, stream_id, stream_method, stream_status, stream_statusReason, string(stream_data))
	if err != nil {
		log.Println("Error in inserting to stream table", err)
	} else {
		log.Println("Successfully inserted into table streams!")
	}

	defer db.Close()
}

func deleteEventsFromDb(db sql.DB, jtis []string) {
	log.Println("jtis to delete: ", jtis)
	for _, value := range jtis {
		statement, err := db.Prepare("DELETE FROM SETs WHERE jti=?")
		if err != nil {
			log.Println("Error in deleting events from  SET table", err)
		}
		log.Println("deleting event: ", value)
		_, err = statement.Exec(value)
		if err != nil {
			log.Println("Error in deleting events from SETs table", err)
		}
	}

}

func addEventToDb(db sql.DB, client_id string, stream_id string, jti string, timestamp int64, event string) {

	// Insert into streams table
	statement, err := db.Prepare("INSERT INTO SETs (client_id, stream_id, jti, timestamp, event) VALUES (?, ?, ?, ?, ?)")
	if err != nil {
		log.Println("statement: ")
		log.Println("Error in inserting to SET table")
	}

	_, err = statement.Exec(client_id, stream_id, jti, timestamp, event)
	if err != nil {
		log.Println("Error in adding to event table", err)
	}

	defer db.Close()
}

func getStreamStatusFromDb(db sql.DB, stream_id string) (string, string) {

	// Insert into streams table
	statement, err := db.Prepare("SELECT stream_status, stream_statusReason FROM streams WHERE stream_id=?")
	if err != nil {
		log.Println("Error in selecting from stream table")
	}
	result, err := statement.Query(stream_id)
	if err != nil {
		log.Println("Error in getting stream from streams table", err)
	}

	if err != nil {
		log.Println("error in getting affected rows")
	}

	var stream_status string
	var stream_statusReason string
	isNextRow := result.Next()
	if isNextRow {
		log.Println(result)
		if err := result.Scan(&stream_status, &stream_statusReason); err != nil {
			log.Fatal(err)
		}
	} else {
		stream_status = "No Stream Found"
	}
	defer result.Close()
	defer db.Close()

	log.Println("stream data: ", stream_status)

	return stream_status, stream_statusReason
}

func getStreamFromDb(db sql.DB, stream_id string) string {

	// Insert into streams table
	statement, err := db.Prepare("SELECT stream_data FROM streams WHERE stream_id=?")
	if err != nil {
		log.Println("Error in selecting from stream table")
	}
	result, err := statement.Query(stream_id)
	if err != nil {
		log.Println("Error in getting stream from streams table", err)
	}

	if err != nil {
		log.Println("error in getting affected rows")
	}

	var stream_data string
	isNextRow := result.Next()
	if isNextRow {
		if err := result.Scan(&stream_data); err != nil {
			log.Fatal(err)
		}
	} else {
		stream_data = "No Stream Found"
	}
	defer result.Close()
	defer db.Close()

	log.Println("stream data: ", stream_data)

	return stream_data
}

func getStreamsFromDb(db sql.DB, clientId string) []StreamConfig {

	// Insert into streams table
	statement, err := db.Prepare("SELECT stream_data FROM streams WHERE client_id=?")

	if err != nil {
		log.Println("Error in selecting from stream table")
	}
	result, err := statement.Query(clientId)
	if err != nil {
		log.Println("Error in getting stream from streams table", err)
	}

	if err != nil {
		log.Println("error in getting affected rows")
	}

	var stream_data string
	var stream StreamConfig
	var streams []StreamConfig
	//log.Println("before result.next")
	for result.Next() {
		if err := result.Scan(&stream_data); err != nil {
			log.Fatal(err)
		}
		// unmarshall it to a stream
		json.Unmarshal([]byte(stream_data), &stream)
		streams = append(streams, stream)
	}
	//log.Println("after log.next")
	defer result.Close()
	defer db.Close()

	log.Println("stream data: ", streams)

	return streams
}

func getStreamsForSubject(db sql.DB, email string) ([]string, []string, []string, []string) {

	// Insert into streams table
	statement, err := db.Prepare("SELECT client_id, stream_id FROM subjects WHERE email=?")
	if err != nil {
		log.Println("Error in selecting from stream table")
	}
	result, err := statement.Query(email)
	if err != nil {
		log.Println("Error in selecting stream for subject", err)
	}

	if err != nil {
		log.Println("error in getting affected rows")
	}

	var streamIds []string
	var clientIds []string
	var deliveryMethods []string
	var endpointUrls []string

	var streamId string
	var clientId string

	//isResult := result.Next()
	for result.Next() {
		// need to make this return any and all streams and clients that match so that it goes out multiplex

		if err := result.Scan(&clientId, &streamId); err != nil {
			log.Println("error in row scan: ", err)
		}
		log.Println("************next set: ", clientId, " ", streamId)
		clientIds = append(clientIds, clientId)
		streamIds = append(streamIds, streamId)
		streamRaw := []byte(getStreamFromDb(db, streamId))
		var streamConfig StreamConfig
		_ = json.Unmarshal(streamRaw, &streamConfig)
		deliveryMethods = append(deliveryMethods, streamConfig.Delivery.DeliveryMethod)
		endpointUrls = append(endpointUrls, streamConfig.Delivery.EndpointUrl)

	}

	defer result.Close()
	defer db.Close()

	log.Println("stream id: ", streamId)

	return clientIds, streamIds, deliveryMethods, endpointUrls
}

func getStreamFromId(db sql.DB, streamId string) (string, string) {

	streamRaw := []byte(getStreamFromDb(db, streamId))
	var streamConfig StreamConfig
	_ = json.Unmarshal(streamRaw, &streamConfig)

	log.Println("deliv method", streamConfig.Delivery.DeliveryMethod, " url:", streamConfig.Delivery.EndpointUrl)

	return streamConfig.Delivery.DeliveryMethod, streamConfig.Delivery.EndpointUrl
}

func getStreamForSubject(db sql.DB, email string) (string, string, string, string) {

	// Insert into streams table
	statement, err := db.Prepare("SELECT client_id, stream_id FROM subjects WHERE email=?")
	if err != nil {
		log.Println("Error in selecting from stream table")
	}
	result, err := statement.Query(email)
	if err != nil {
		log.Println("Error in selecting stream for subject", err)
	}

	if err != nil {
		log.Println("error in getting affected rows")
	}

	var streamId string
	var clientId string
	isResult := result.Next()
	// need to make this return any and all streams and clients that match so that it goes out multiplex
	if isResult {
		if err := result.Scan(&clientId, &streamId); err != nil {
			log.Println("error in row scan: ", err)
		}
	} else {
		clientId = "none"
		streamId = "none"
	}

	streamRaw := []byte(getStreamFromDb(db, streamId))
	var streamConfig StreamConfig
	_ = json.Unmarshal(streamRaw, &streamConfig)

	defer result.Close()
	defer db.Close()

	log.Println("stream id: ", streamId)

	return clientId, streamId, streamConfig.Delivery.DeliveryMethod, streamConfig.Delivery.EndpointUrl
}

func addSubjectToStreamTable(db sql.DB, clientId string, streamId string, email string) {

	// Insert into streams table
	statement, err := db.Prepare("INSERT INTO subjects (client_id, stream_id, email, status) VALUES (?, ?, ?, ?)")
	if err != nil {
		log.Println("statement: ")
		log.Println("Error in inserting to SET table")
	}

	_, err = statement.Exec(clientId, streamId, email, "enabled")
	if err != nil {
		log.Println("Error in adding to event table", err)
	}

	defer db.Close()
}

func removeSubjectFromStreamTable(db sql.DB, clientId string, streamId string, email string) {

	// Insert into streams table
	statement, err := db.Prepare("DELETE FROM subjects WHERE client_id=? AND stream_id=? AND email=?")

	if err != nil {
		log.Println("statement: ")
		log.Println("Error in deleting to SET table", err)
	}

	_, err = statement.Exec(clientId, streamId, email)
	if err != nil {
		log.Println("Error in adding to event table", err)
	}

	defer db.Close()
}

func removeStreamFromTable(db sql.DB, clientId string, streamId string) {

	// delete from streams table
	statement, err := db.Prepare("DELETE FROM streams WHERE client_id=? AND stream_id=?")

	if err != nil {
		log.Println("statement: ")
		log.Println("Error in deleting from streams table", err)
	}

	_, err = statement.Exec(clientId, streamId)
	if err != nil {
		log.Println("Error in adding to event table", err)
	}

	// delete subjects as well for stream
	statement, err = db.Prepare("DELETE FROM subjects WHERE client_id=? AND stream_id=?")

	if err != nil {
		log.Println("statement: ")
		log.Println("Error in deleting from streams table", err)
	}

	_, err = statement.Exec(clientId, streamId)
	if err != nil {
		log.Println("Error in adding to event table", err)
	}

	defer db.Close()
}

func getEventsForStream(db sql.DB, stream_id string) string {

	// Insert into streams table
	statement, err := db.Prepare("SELECT jti, event FROM SETs WHERE stream_id=?")
	if err != nil {
		log.Println("Error in selecting from stream table")
	}
	result, err := statement.Query(stream_id)
	if err != nil {
		log.Println("Error in inserting to stream table", err)
	}

	if err != nil {
		log.Println("error in getting affected rows")
	}

	var events string
	events = "{\"sets\":{"
	var eventData string
	var jti string
	for result.Next() {
		if err := result.Scan(&jti, &eventData); err != nil {
			log.Fatal(err)
		}
		log.Println("next event:", eventData)
		// unmarshall it to a stream
		events = events + "\"" + jti + "\": " + "\"" + eventData + "\","

	}
	// take off last comma
	events, _ = strings.CutSuffix(events, ",")
	events = events + "}}"
	defer db.Close()
	log.Println("event data: ", events)

	return events
}

func open_db() *sql.DB {
	db, err := sql.Open("sqlite3", "ssf_receiver.db")
	if err != nil {
		log.Println(err)
	}
	return db
}

func check_db() {

	db, err := sql.Open("sqlite3", "ssf_receiver.db")
	if err != nil {
		log.Println(err)
	}

	// Create streams table
	statement, err := db.Prepare("CREATE TABLE IF NOT EXISTS streams (client_id TEXT, stream_id TEXT PRIMARY KEY, stream_method TEXT, stream_data TEXT, stream_status TEXT, stream_statusReason TEXT)")
	if err != nil {
		log.Println("Error in creating table")
	}
	statement.Exec()

	// Create SETS (events) table
	statement, err = db.Prepare("CREATE TABLE IF NOT EXISTS SETs (client_id TEXT, stream_id TEXT NOT NULL, jti TEXT NOT NULL, timestamp INTEGER NOT NULL, event TEXT NOT NULL, FOREIGN KEY(stream_id) REFERENCES streams(stream_id), PRIMARY KEY(stream_id, jti))")
	if err != nil {
		log.Println("Error in creating table")
	}
	statement.Exec()

	// Create subjects table
	statement, err = db.Prepare("CREATE TABLE IF NOT EXISTS subjects (client_id TEXT, stream_id TEXT, email TEXT, status TEXT, FOREIGN KEY(client_id) REFERENCES streams(client_id), PRIMARY KEY(stream_id, email) )")
	if err != nil {
		log.Println("Error in creating table")
	}
	statement.Exec()

	defer db.Close()

	var version string
	err = db.QueryRow("SELECT SQLITE_VERSION()").Scan(&version)

	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(version)
}

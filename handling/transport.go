package handling

import (
	"encoding/json"
	"net/http"
	"time"

	"golang.org/x/net/context"

	httptransport "github.com/go-kit/kit/transport/http"

	"github.com/gorilla/mux"
	"github.com/marcusolsson/goddd/cargo"
	"github.com/marcusolsson/goddd/location"
	"github.com/marcusolsson/goddd/voyage"
)

// MakeHandler ...
func MakeHandler(ctx context.Context, hs Service) http.Handler {
	r := mux.NewRouter()

	registerIncidentHandler := httptransport.NewServer(
		ctx,
		makeRegisterIncidentEndpoint(hs),
		decodeRegisterIncidentRequest,
		encodeResponse,
	)

	r.Handle("/handling/v1/incidents", registerIncidentHandler).Methods("POST")
	r.Handle("/handling/v1/docs", http.StripPrefix("/handling/v1/docs", http.FileServer(http.Dir("handling/docs"))))

	return r
}

func decodeRegisterIncidentRequest(r *http.Request) (interface{}, error) {
	var body struct {
		CompletionTime int64
		TrackingID     string
		VoyageNumber   string
		Location       string
		EventType      string
	}

	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		return nil, err
	}

	return registerIncidentRequest{
		CompletionTime: time.Unix(body.CompletionTime/1000, 0),
		ID:             cargo.TrackingID(body.TrackingID),
		Voyage:         voyage.Number(body.VoyageNumber),
		Location:       location.UNLocode(body.Location),
		EventType:      stringToEventType(body.EventType),
	}, nil
}

func stringToEventType(s string) cargo.HandlingEventType {
	types := map[string]cargo.HandlingEventType{
		cargo.Receive.String(): cargo.Receive,
		cargo.Load.String():    cargo.Load,
		cargo.Unload.String():  cargo.Unload,
		cargo.Customs.String(): cargo.Customs,
		cargo.Claim.String():   cargo.Claim,
	}
	return types[s]
}

func encodeResponse(w http.ResponseWriter, response interface{}) error {
	if e, ok := response.(errorer); ok && e.error() != nil {
		encodeError(w, e.error())
		return nil
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	return json.NewEncoder(w).Encode(response)
}

type errorer interface {
	error() error
}

// encode errors from business-logic
func encodeError(w http.ResponseWriter, err error) {
	switch err {
	case nil:
		w.WriteHeader(http.StatusOK)
	case cargo.ErrUnknown:
		w.WriteHeader(http.StatusNotFound)
	default:
		w.WriteHeader(http.StatusInternalServerError)
	}
	json.NewEncoder(w).Encode(map[string]interface{}{
		"error": err.Error(),
	})
}
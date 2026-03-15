package threads

import (
	"context"
	"net/http"
	"testing"
)

func TestSearchLocations_Success(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{
		"data": [
			{"id": "loc1", "name": "Coffee Shop", "city": "San Francisco"},
			{"id": "loc2", "name": "Coffee House", "city": "San Francisco"}
		]
	}`))

	resp, err := client.SearchLocations(context.Background(), "coffee", nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Data) != 2 {
		t.Errorf("expected 2 locations, got %d", len(resp.Data))
	}
}

func TestSearchLocations_EmptyQuery(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{}`))
	_, err := client.SearchLocations(context.Background(), "", nil, nil)
	if err == nil {
		t.Fatal("expected error for empty query")
	}
}

func TestSearchLocations_WithLatLong(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if q.Get("latitude") == "" {
			t.Error("expected latitude parameter")
		}
		if q.Get("longitude") == "" {
			t.Error("expected longitude parameter")
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{"data":[]}`))
	}
	client := testClient(t, http.HandlerFunc(handler))
	lat := 37.7749
	lng := -122.4194
	_, err := client.SearchLocations(context.Background(), "", &lat, &lng)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSearchLocations_WithQueryAndLatLong(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if q.Get("q") != "cafe" {
			t.Errorf("expected q=cafe, got %s", q.Get("q"))
		}
		if q.Get("latitude") == "" {
			t.Error("expected latitude parameter")
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{"data":[{"id":"1","name":"Cafe"}]}`))
	}
	client := testClient(t, http.HandlerFunc(handler))
	lat := 37.7749
	_, err := client.SearchLocations(context.Background(), "cafe", &lat, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSearchLocations_500(t *testing.T) {
	client := testClient(t, jsonHandler(500, `{"error":{"message":"server error"}}`))
	_, err := client.SearchLocations(context.Background(), "cafe", nil, nil)
	if err == nil {
		t.Fatal("expected error for 500")
	}
}

func TestGetLocation_Success(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{
		"id": "loc1",
		"name": "Golden Gate Park",
		"city": "San Francisco",
		"latitude": 37.7694,
		"longitude": -122.4862
	}`))

	loc, err := client.GetLocation(context.Background(), ConvertToLocationID("loc1"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if loc.Name != "Golden Gate Park" {
		t.Errorf("expected Golden Gate Park, got %s", loc.Name)
	}
}

func TestGetLocation_InvalidID(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{}`))
	_, err := client.GetLocation(context.Background(), LocationID(""))
	if err == nil {
		t.Fatal("expected error for empty location ID")
	}
}

func TestGetLocation_404(t *testing.T) {
	client := testClient(t, jsonHandler(404, `{"error":{"message":"not found"}}`))
	_, err := client.GetLocation(context.Background(), ConvertToLocationID("nonexistent"))
	if err == nil {
		t.Fatal("expected error for 404")
	}
}

func TestGetLocation_500(t *testing.T) {
	client := testClient(t, jsonHandler(500, `{"error":{"message":"server error"}}`))
	_, err := client.GetLocation(context.Background(), ConvertToLocationID("loc1"))
	if err == nil {
		t.Fatal("expected error for 500")
	}
}

func TestGetLocation_FullDetails(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{
		"id": "loc1",
		"name": "Cafe Central",
		"address": "123 Main St",
		"city": "New York",
		"country": "US",
		"latitude": 40.7128,
		"longitude": -74.0060,
		"postal_code": "10001"
	}`))
	loc, err := client.GetLocation(context.Background(), ConvertToLocationID("loc1"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if loc.Address != "123 Main St" {
		t.Errorf("expected 123 Main St, got %s", loc.Address)
	}
	if loc.Country != "US" {
		t.Errorf("expected US, got %s", loc.Country)
	}
}

func TestSearchLocations_NoAuth(t *testing.T) {
	client := testClientNoAuth(t, jsonHandler(200, `{}`))
	_, err := client.SearchLocations(context.Background(), "coffee", nil, nil)
	if err == nil {
		t.Fatal("expected error for unauthenticated client")
	}
}

func TestSearchLocations_InvalidJSON(t *testing.T) {
	client := testClient(t, jsonHandler(200, `not json`))
	_, err := client.SearchLocations(context.Background(), "coffee", nil, nil)
	if err == nil {
		t.Fatal("expected error for invalid JSON response")
	}
}

func TestGetLocation_NoAuth(t *testing.T) {
	client := testClientNoAuth(t, jsonHandler(200, `{}`))
	_, err := client.GetLocation(context.Background(), ConvertToLocationID("loc1"))
	if err == nil {
		t.Fatal("expected error for unauthenticated client")
	}
}

func TestGetLocation_InvalidJSON(t *testing.T) {
	client := testClient(t, jsonHandler(200, `not json`))
	_, err := client.GetLocation(context.Background(), ConvertToLocationID("loc1"))
	if err == nil {
		t.Fatal("expected error for invalid JSON response")
	}
}

func TestSearchLocations_OnlyLatitude(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{"data":[]}`))
	lat := 37.7749
	_, err := client.SearchLocations(context.Background(), "", &lat, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSearchLocations_OnlyLongitude(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{"data":[]}`))
	lng := -122.4194
	_, err := client.SearchLocations(context.Background(), "", nil, &lng)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

package threads

import (
	"context"
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

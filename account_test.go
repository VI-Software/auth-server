package main

import (
	"bytes"
	"encoding/json"
	// "fmt"
	"github.com/stretchr/testify/assert"
	// "net"
	"net/http"
	"net/http/httptest"
	// "net/url"
	"testing"
)

func TestAccount(t *testing.T) {
	{
		ts := &TestSuite{}

		config := testConfig()
		ts.Setup(config)
		defer ts.Teardown()

		ts.CreateTestUser(ts.Server, TEST_USERNAME)

		t.Run("Test /users/profiles/minecraft/:playerName", ts.testAccountPlayerNameToID)
		t.Run("Test /profiles/minecraft", ts.testAccountPlayerNamesToIDs)
	}
	{
		ts := &TestSuite{}

		auxConfig := testConfig()
		ts.SetupAux(auxConfig)

		config := testConfig()
		config.FallbackAPIServers = []FallbackAPIServer{ts.ToFallbackAPIServer(ts.AuxApp, "Aux")}
		ts.Setup(config)
		defer ts.Teardown()

		ts.CreateTestUser(ts.AuxServer, TEST_USERNAME)

		t.Run("Test /users/profiles/minecraft/:playerName, fallback API server", ts.testAccountPlayerNameToIDFallback)
		t.Run("Test /profile/minecraft, fallback API server", ts.testAccountPlayerNamesToIDsFallback)
	}
}

func (ts *TestSuite) testAccountPlayerNameToID(t *testing.T) {
	rec := ts.Get(t, ts.Server, "/users/profiles/minecraft/"+TEST_USERNAME, nil, nil)

	assert.Equal(t, http.StatusOK, rec.Code)
	var response playerNameToUUIDResponse
	assert.Nil(t, json.NewDecoder(rec.Body).Decode(&response))

	// Check that the player name is correct
	assert.Equal(t, response.Name, TEST_USERNAME)

	// Get the real UUID
	var user User
	result := ts.App.DB.First(&user, "username = ?", TEST_USERNAME)
	assert.Nil(t, result.Error)

	// Check that the UUID is correct
	uuid, err := IDToUUID(response.ID)
	assert.Nil(t, err)
	assert.Equal(t, uuid, user.UUID)

	// Any case variations of the username should return the same user
	rec = ts.Get(t, ts.Server, "/users/profiles/minecraft/"+TEST_USERNAME_UPPERCASE, nil, nil)
	assert.Nil(t, json.NewDecoder(rec.Body).Decode(&response))
	uuid, err = IDToUUID(response.ID)
	assert.Equal(t, user.UUID, uuid)
}

func (ts *TestSuite) testAccountPlayerNamesToIDs(t *testing.T) {
	payload := []string{TEST_USERNAME, "nonexistent"}
	body, err := json.Marshal(payload)
	assert.Nil(t, err)

	req := httptest.NewRequest(http.MethodPost, "/profiles/minecraft", bytes.NewBuffer(body))
	rec := httptest.NewRecorder()
	ts.Server.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	var response []playerNameToUUIDResponse
	assert.Nil(t, json.NewDecoder(rec.Body).Decode(&response))

	// Get the real UUID
	var user User
	result := ts.App.DB.First(&user, "username = ?", TEST_USERNAME)
	assert.Nil(t, result.Error)
	id, err := UUIDToID(user.UUID)
	assert.Nil(t, err)

	// There should only be one user, the nonexistent user should not be present
	assert.Equal(t, []playerNameToUUIDResponse{{Name: TEST_USERNAME, ID: id}}, response)
}

func (ts *TestSuite) testAccountPlayerNameToIDFallback(t *testing.T) {
	{
		req := httptest.NewRequest(http.MethodGet, "/users/profiles/minecraft/"+TEST_USERNAME, nil)
		rec := httptest.NewRecorder()
		ts.Server.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		var response playerNameToUUIDResponse
		assert.Nil(t, json.NewDecoder(rec.Body).Decode(&response))

		// Check that the player name is correct
		assert.Equal(t, response.Name, TEST_USERNAME)

		// Get the real UUID
		var user User
		result := ts.AuxApp.DB.First(&user, "username = ?", TEST_USERNAME)
		assert.Nil(t, result.Error)

		// Check that the UUID is correct
		uuid, err := IDToUUID(response.ID)
		assert.Nil(t, err)
		assert.Equal(t, uuid, user.UUID)

		// Test that fallback requests are correctly cached: change the aux
		// user's player name and make sure the main server finds the old
		// profile in the cache
		user.PlayerName = "testcache"
		ts.AuxApp.DB.Save(&user)

		req = httptest.NewRequest(http.MethodGet, "/users/profiles/minecraft/"+TEST_USERNAME, nil)
		rec = httptest.NewRecorder()
		ts.Server.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Nil(t, json.NewDecoder(rec.Body).Decode(&response))
		uuid, err = IDToUUID(response.ID)
		assert.Equal(t, uuid, user.UUID)
	}

	// Test a non-existent user
	{
		req := httptest.NewRequest(http.MethodGet, "/users/profiles/minecraft/"+"nonexistent", nil)
		rec := httptest.NewRecorder()
		ts.Server.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusNotFound, rec.Code)
	}
}

func (ts *TestSuite) testAccountPlayerNamesToIDsFallback(t *testing.T) {
	payload := []string{TEST_USERNAME, "nonexistent"}
	body, err := json.Marshal(payload)
	assert.Nil(t, err)

	req := httptest.NewRequest(http.MethodPost, "/profiles/minecraft", bytes.NewBuffer(body))
	rec := httptest.NewRecorder()
	ts.Server.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	var response []playerNameToUUIDResponse
	assert.Nil(t, json.NewDecoder(rec.Body).Decode(&response))

	// Get the real UUID
	var user User
	result := ts.AuxApp.DB.First(&user, "username = ?", TEST_USERNAME)
	assert.Nil(t, result.Error)

	// There should only be one user, the nonexistent user should not be present
	id, err := UUIDToID(user.UUID)
	assert.Nil(t, err)
	assert.Equal(t, []playerNameToUUIDResponse{{Name: TEST_USERNAME, ID: id}}, response)
}

func (ts *TestSuite) testAccountVerifySecurityLocation(t *testing.T) {
	rec := ts.Get(t, ts.Server, "/user/security/location", nil, nil)
	assert.Equal(t, http.StatusOK, rec.Code)
}

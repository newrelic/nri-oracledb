package main

import (
	"testing"
)

func TestGetCollectionString(t *testing.T) {

	args = argumentList{
		ServiceName: "testservice",
		Hostname:    "testhost",
		Password:    "testpassword",
		Username:    "testuser",
		IsSysDBA:    true,
	}

	s := getConnectionString()
	expectedConnectionString := `oracle://testuser:testpassword@testhost:/testservice?connectionClass=&poolIncrement=0&poolMaxSessions=8&poolMinSessions=0&sysdba=1&sysoper=0&standaloneConnection=0`

	if s != expectedConnectionString {
		t.Errorf("Incorrect connection string %s", s)
	}

}

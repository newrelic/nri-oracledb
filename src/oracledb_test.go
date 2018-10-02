package main

import (
	"reflect"
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

func Test_parseTablespaceWhitelist(t *testing.T) {
	testCases := []struct {
		name string
		arg  string
		want []string
	}{
		{
			"No Whitelist",
			"",
			nil,
		},
		{
			"Whitelist",
			`["one", "two", "three"]`,
			[]string{"one", "two", "three"},
		},
		{
			"Empty Whitelist",
			`[]`,
			[]string{},
		},
	}

	for _, tc := range testCases {
		args.Tablespaces = tc.arg
		tablespaceWhiteList = nil
		if err := parseTablespaceWhitelist(); err != nil {
			t.Errorf("Test Case %s Failed: Unexpected error: %s", tc.name, err.Error())
			t.FailNow()
		}

		if !reflect.DeepEqual(tablespaceWhiteList, tc.want) {
			t.Errorf("Test Case %s Failed: Expected '%+v', got '%+v'", tc.name, tc.want, tablespaceWhiteList)
		}
	}
}

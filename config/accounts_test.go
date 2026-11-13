package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestDeviceNameMigrationIsPersistedAndKept(t *testing.T) {
	path := filepath.Join(t.TempDir(), AccountsFileName)
	m := &AccountManager{path: path, Accounts: map[string]Account{}}
	if err := m.AddAccount(Account{UserID: "u1", Hostname: "https://a"}); err != nil {
		t.Fatal(err)
	}
	if got := m.GetDeviceNameMigration("u1", "olm1"); got != "" {
		t.Fatalf("fresh account has migration %q", got)
	}
	if err := m.SetDeviceNameMigration("u1", "olm1", "thinkpad"); err != nil {
		t.Fatal(err)
	}
	if err := m.SetDeviceNameMigration("nobody", "olm1", "x"); err == nil {
		t.Error("unknown account should fail")
	}

	// Signing in again replaces the account but must keep the record.
	if err := m.AddAccount(Account{UserID: "u1", Hostname: "https://a", Email: "e"}); err != nil {
		t.Fatal(err)
	}
	if got := m.GetDeviceNameMigration("u1", "olm1"); got != "thinkpad" {
		t.Errorf("after AddAccount: migration = %q, want thinkpad", got)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var onDisk AccountManager
	if err := json.Unmarshal(data, &onDisk); err != nil {
		t.Fatal(err)
	}
	if got := onDisk.Accounts["u1"].DeviceNameMigrations["olm1"]; got != "thinkpad" {
		t.Errorf("on disk: migration = %q, want thinkpad", got)
	}
}

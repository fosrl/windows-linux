package config

import "testing"

func TestMergeConfigOrgsAreKeyLevel(t *testing.T) {
	system := 1300
	user := true
	merged := mergeConfig(
		&Config{Orgs: map[string]OrgConfig{"a": {MTU: &system}}},
		&Config{Orgs: map[string]OrgConfig{"a": {DNSTunnel: &user}}},
	)

	org := merged.Orgs["a"]
	if org.MTU == nil || *org.MTU != system {
		t.Errorf("system MTU lost: %+v", org)
	}
	if org.DNSTunnel == nil || !*org.DNSTunnel {
		t.Errorf("user DNSTunnel lost: %+v", org)
	}
}

func TestEffectiveExitNode(t *testing.T) {
	orgNode := 30
	org := OrgConfig{ExitNodeResourceID: &orgNode}

	tests := []struct {
		name   string
		server *ClientSettings
		saved  int
		org    OrgConfig
		want   int
	}{
		{"nothing set", nil, 0, OrgConfig{}, 0},
		{"server default only", &ClientSettings{ExitNodeSiteResourceID: 10}, 0, OrgConfig{}, 10},
		{"org config beats server default", &ClientSettings{ExitNodeSiteResourceID: 10}, 0, org, 30},
		{"user selection beats org config", &ClientSettings{ExitNodeSiteResourceID: 10}, 20, org, 20},
		{"locked beats everything", &ClientSettings{ExitNodeSiteResourceID: 10, ExitNodeLocked: true}, 20, org, 10},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := EffectiveExitNode(tt.server, tt.saved, tt.org); got != tt.want {
				t.Errorf("got %d, want %d", got, tt.want)
			}
		})
	}
}

func TestEffectiveSettingsPrecedence(t *testing.T) {
	global, org, server := 1400, 1300, 1200
	empty := ""
	cm := &ConfigManager{config: &Config{
		MTU:        &global,
		PrimaryDNS: strPtr("1.1.1.1"),
		Orgs: map[string]OrgConfig{
			"a": {MTU: &org, PrimaryDNS: &empty},
		},
	}}

	if got := cm.EffectiveSettings("b", nil).MTU; got != global {
		t.Errorf("no org section: MTU = %d, want %d", got, global)
	}
	settings := cm.EffectiveSettings("a", nil)
	if settings.MTU != org {
		t.Errorf("org section: MTU = %d, want %d", settings.MTU, org)
	}
	if settings.PrimaryDNS != "" {
		t.Errorf("an org should be able to clear a global DNS server, got %q", settings.PrimaryDNS)
	}
	if got := cm.EffectiveSettings("a", &ClientSettings{MTU: &server}).MTU; got != server {
		t.Errorf("server: MTU = %d, want %d", got, server)
	}
}

func strPtr(s string) *string { return &s }

func TestGiveOrgOwnSettingsCopiesEverySetting(t *testing.T) {
	global := 1400
	exitNode := 7
	cm := &ConfigManager{
		configPath: t.TempDir() + "/pangolin.json",
		config: &Config{
			MTU:        &global,
			PrimaryDNS: strPtr("1.1.1.1"),
			Orgs:       map[string]OrgConfig{"a": {ExitNodeResourceID: &exitNode}},
		},
	}
	if cm.GetOrgConfig("a").HasOwnSettings() {
		t.Fatal("an exit node alone shouldn't count as the org's own settings")
	}

	if !cm.GiveOrgOwnSettings("a") {
		t.Fatal("GiveOrgOwnSettings failed")
	}
	org := cm.GetOrgConfig("a")
	if !org.HasOwnSettings() {
		t.Fatal("org should have its own settings")
	}
	// Nothing the org uses changes when it gets its own settings.
	if got := cm.EffectiveSettings("a", nil); got.MTU != global || got.PrimaryDNS != "1.1.1.1" ||
		got.DNSOverride != DefaultDNSOverride {
		t.Errorf("effective settings changed: %+v", got)
	}
	// Every setting is copied, so later global changes don't reach the org.
	if org.DNSOverride == nil || org.DNSTunnel == nil || org.PrimaryDNS == nil ||
		org.SecondaryDNS == nil || org.MTU == nil || org.ExitNodeTakesPrecedence == nil {
		t.Errorf("not every setting was copied: %+v", org)
	}
	newGlobal := 1300
	cm.config.MTU = &newGlobal
	if got := cm.EffectiveSettings("a", nil).MTU; got != global {
		t.Errorf("global change reached the org: MTU = %d, want %d", got, global)
	}

	// Reset clears the settings but keeps the exit node.
	if !cm.ResetOrgConfig("a") {
		t.Fatal("ResetOrgConfig failed")
	}
	org = cm.GetOrgConfig("a")
	if org.HasOwnSettings() {
		t.Errorf("reset left settings behind: %+v", org)
	}
	if org.ExitNodeResourceID == nil || *org.ExitNodeResourceID != exitNode {
		t.Errorf("reset lost the exit node: %+v", org)
	}
}

func TestResetOrgConfigRemovesEmptySection(t *testing.T) {
	mtu := 1300
	cm := &ConfigManager{
		configPath: t.TempDir() + "/pangolin.json",
		config:     &Config{Orgs: map[string]OrgConfig{"a": {MTU: &mtu}}},
	}
	if !cm.ResetOrgConfig("a") {
		t.Fatal("ResetOrgConfig failed")
	}
	if _, ok := cm.config.Orgs["a"]; ok {
		t.Error("an org with nothing left should be removed")
	}
}

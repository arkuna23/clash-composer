package config

import (
	"encoding/json"
	"errors"
	"net/netip"
	"reflect"
	"strings"
	"testing"
)

func TestUnmarshalRawConfigUsesDefaults(t *testing.T) {
	cfg, err := UnmarshalRawConfig(nil)
	if err != nil {
		t.Fatalf("unmarshal default config: %v", err)
	}

	if cfg.BindAddress != "*" {
		t.Fatalf("BindAddress = %q, want *", cfg.BindAddress)
	}
	if cfg.Mode != Rule {
		t.Fatalf("Mode = %v, want Rule", cfg.Mode)
	}
	if cfg.LogLevel != INFO {
		t.Fatalf("LogLevel = %v, want INFO", cfg.LogLevel)
	}
	if !cfg.IPv6 {
		t.Fatalf("IPv6 = false, want true")
	}
	if cfg.DNS.EnhancedMode != DNSMapping {
		t.Fatalf("DNS.EnhancedMode = %v, want DNSMapping", cfg.DNS.EnhancedMode)
	}
	if cfg.Tun.Stack != TunGvisor {
		t.Fatalf("Tun.Stack = %v, want TunGvisor", cfg.Tun.Stack)
	}
	if got := cfg.Tun.Inet6Address[0]; got != netip.MustParsePrefix("fdfe:dcba:9876::1/126") {
		t.Fatalf("Tun.Inet6Address[0] = %v", got)
	}
}

func TestUnmarshalRawConfigFullSample(t *testing.T) {
	input := []byte(`
port: 7890
mode: global
log-level: debug
find-process-mode: always
proxies:
  - name: direct-one
    type: direct
proxy-groups:
  - name: auto
    type: select
    proxies: [DIRECT, direct-one]
rules:
  - MATCH,DIRECT
dns:
  enable: true
  enhanced-mode: fake-ip
  fake-ip-filter-mode: whitelist
  nameserver-policy:
    "geosite:cn": [https://doh.pub/dns-query]
    "+.example.com": 8.8.8.8
tun:
  enable: true
  stack: mixed
  route-address: [10.0.0.0/8]
  loopback-address: [10.0.0.1]
sniffer:
  enable: true
  sniff:
    HTTP:
      ports: [80, 8080-8880]
tunnels:
  - tcp/udp,127.0.0.1:6553,8.8.8.8:53,DIRECT
  - network: [tcp]
    address: 127.0.0.1:8443
    target: 1.1.1.1:443
tls:
  custom-certifactes:
    - /tmp/ca.pem
`)

	cfg, err := UnmarshalRawConfig(input)
	if err != nil {
		t.Fatalf("unmarshal sample: %v", err)
	}

	if cfg.Port != 7890 || cfg.Mode != Global || cfg.LogLevel != DEBUG {
		t.Fatalf("top-level fields not decoded: port=%d mode=%v log=%v", cfg.Port, cfg.Mode, cfg.LogLevel)
	}
	if cfg.FindProcessMode != FindProcessAlways {
		t.Fatalf("FindProcessMode = %v, want always", cfg.FindProcessMode)
	}
	if cfg.DNS.EnhancedMode != DNSFakeIP || cfg.DNS.FakeIPFilterMode != FilterWhiteList {
		t.Fatalf("DNS modes not decoded: enhanced=%v filter=%v", cfg.DNS.EnhancedMode, cfg.DNS.FakeIPFilterMode)
	}
	if got := cfg.DNS.NameServerPolicy.Keys(); !reflect.DeepEqual(got, []string{"geosite:cn", "+.example.com"}) {
		t.Fatalf("policy keys = %#v", got)
	}
	if cfg.Tun.Stack != TunMixed {
		t.Fatalf("Tun.Stack = %v, want mixed", cfg.Tun.Stack)
	}
	if got := cfg.Tun.RouteAddress[0]; got != netip.MustParsePrefix("10.0.0.0/8") {
		t.Fatalf("Tun.RouteAddress[0] = %v", got)
	}
	if got := cfg.Tun.LoopbackAddress[0]; got != netip.MustParseAddr("10.0.0.1") {
		t.Fatalf("Tun.LoopbackAddress[0] = %v", got)
	}
	if len(cfg.Tunnels) != 2 {
		t.Fatalf("len(Tunnels) = %d, want 2", len(cfg.Tunnels))
	}
	if got := cfg.Tunnels[0]; !reflect.DeepEqual(got.Network, []string{"tcp", "udp"}) || got.Proxy != "DIRECT" {
		t.Fatalf("string tunnel not decoded: %#v", got)
	}
	if cfg.TLS.CustomTrustCert[0] != "/tmp/ca.pem" {
		t.Fatalf("TLS custom cert not decoded: %#v", cfg.TLS.CustomTrustCert)
	}
	if cfg.Proxy[0]["name"] != "direct-one" {
		t.Fatalf("proxy map not preserved: %#v", cfg.Proxy)
	}
}

func TestMarshalRawConfigRoundTrip(t *testing.T) {
	cfg, err := UnmarshalRawConfig([]byte(`
mode: direct
log-level: warning
dns:
  nameserver-policy:
    a.test: 1.1.1.1
    b.test: [8.8.8.8]
tunnels:
  - tcp,127.0.0.1:1000,127.0.0.1:2000
`))
	if err != nil {
		t.Fatalf("unmarshal input: %v", err)
	}

	data, err := MarshalRawConfig(cfg)
	if err != nil {
		t.Fatalf("marshal config: %v", err)
	}
	if !strings.Contains(string(data), "mode: direct") {
		t.Fatalf("marshaled config did not use text enum values:\n%s", data)
	}

	next, err := UnmarshalRawConfig(data)
	if err != nil {
		t.Fatalf("unmarshal marshaled config: %v", err)
	}
	if next.Mode != Direct || next.LogLevel != WARNING {
		t.Fatalf("round-trip enums not preserved: mode=%v log=%v", next.Mode, next.LogLevel)
	}
	if got := next.DNS.NameServerPolicy.Keys(); !reflect.DeepEqual(got, []string{"a.test", "b.test"}) {
		t.Fatalf("round-trip policy keys = %#v", got)
	}
	if got := next.Tunnels[0]; got.Address != "127.0.0.1:1000" || got.Target != "127.0.0.1:2000" {
		t.Fatalf("round-trip tunnel = %#v", got)
	}
}

func TestMarshalRawConfigNil(t *testing.T) {
	_, err := MarshalRawConfig(nil)
	if !errors.Is(err, ErrNilRawConfig) {
		t.Fatalf("MarshalRawConfig(nil) error = %v", err)
	}
}

func TestJSONTagsAndOrderedMap(t *testing.T) {
	cfg := DefaultRawConfig()
	cfg.Rule = []string{"MATCH,DIRECT"}
	cfg.DNS.NameServerPolicy = NewOrderedMap()
	cfg.DNS.NameServerPolicy.Set("first.test", "1.1.1.1")
	cfg.DNS.NameServerPolicy.Set("second.test", []any{"8.8.8.8"})

	data, err := json.Marshal(cfg)
	if err != nil {
		t.Fatalf("marshal JSON: %v", err)
	}
	text := string(data)
	if !strings.Contains(text, `"rule":["MATCH,DIRECT"]`) {
		t.Fatalf("JSON rules tag mismatch: %s", text)
	}
	if !strings.Contains(text, `"first.test":"1.1.1.1","second.test":["8.8.8.8"]`) {
		t.Fatalf("ordered map JSON order mismatch: %s", text)
	}
}

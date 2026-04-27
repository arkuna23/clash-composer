package config

import "net/netip"

const defaultMihomoVersion = "1.10.0"

type RawCors struct {
	AllowOrigins        []string `yaml:"allow-origins" json:"allow-origins"`
	AllowPrivateNetwork bool     `yaml:"allow-private-network" json:"allow-private-network"`
}

type RawDNS struct {
	Enable                       bool              `yaml:"enable" json:"enable"`
	PreferH3                     bool              `yaml:"prefer-h3" json:"prefer-h3"`
	IPv6                         bool              `yaml:"ipv6" json:"ipv6"`
	IPv6Timeout                  uint              `yaml:"ipv6-timeout" json:"ipv6-timeout"`
	UseHosts                     bool              `yaml:"use-hosts" json:"use-hosts"`
	UseSystemHosts               bool              `yaml:"use-system-hosts" json:"use-system-hosts"`
	RespectRules                 bool              `yaml:"respect-rules" json:"respect-rules"`
	NameServer                   []string          `yaml:"nameserver" json:"nameserver"`
	Fallback                     []string          `yaml:"fallback" json:"fallback"`
	FallbackFilter               RawFallbackFilter `yaml:"fallback-filter" json:"fallback-filter"`
	Listen                       string            `yaml:"listen" json:"listen"`
	EnhancedMode                 DNSMode           `yaml:"enhanced-mode" json:"enhanced-mode"`
	FakeIPRange                  string            `yaml:"fake-ip-range" json:"fake-ip-range"`
	FakeIPRange6                 string            `yaml:"fake-ip-range6" json:"fake-ip-range6"`
	FakeIPFilter                 []string          `yaml:"fake-ip-filter" json:"fake-ip-filter"`
	FakeIPFilterMode             FilterMode        `yaml:"fake-ip-filter-mode" json:"fake-ip-filter-mode"`
	FakeIPTTL                    int               `yaml:"fake-ip-ttl" json:"fake-ip-ttl"`
	DefaultNameserver            []string          `yaml:"default-nameserver" json:"default-nameserver"`
	CacheAlgorithm               string            `yaml:"cache-algorithm" json:"cache-algorithm"`
	CacheMaxSize                 int               `yaml:"cache-max-size" json:"cache-max-size"`
	NameServerPolicy             *OrderedMap       `yaml:"nameserver-policy" json:"nameserver-policy"`
	ProxyServerNameserver        []string          `yaml:"proxy-server-nameserver" json:"proxy-server-nameserver"`
	ProxyServerNameserverPolicy  *OrderedMap       `yaml:"proxy-server-nameserver-policy" json:"proxy-server-nameserver-policy"`
	DirectNameServer             []string          `yaml:"direct-nameserver" json:"direct-nameserver"`
	DirectNameServerFollowPolicy bool              `yaml:"direct-nameserver-follow-policy" json:"direct-nameserver-follow-policy"`
}

type RawFallbackFilter struct {
	GeoIP     bool     `yaml:"geoip" json:"geoip"`
	GeoIPCode string   `yaml:"geoip-code" json:"geoip-code"`
	IPCIDR    []string `yaml:"ipcidr" json:"ipcidr"`
	Domain    []string `yaml:"domain" json:"domain"`
	GeoSite   []string `yaml:"geosite" json:"geosite"`
}

type RawClashForAndroid struct {
	AppendSystemDNS   bool   `yaml:"append-system-dns" json:"append-system-dns"`
	UiSubtitlePattern string `yaml:"ui-subtitle-pattern" json:"ui-subtitle-pattern"`
}

type RawNTP struct {
	Enable        bool   `yaml:"enable" json:"enable"`
	Server        string `yaml:"server" json:"server"`
	Port          int    `yaml:"port" json:"port"`
	Interval      int    `yaml:"interval" json:"interval"`
	DialerProxy   string `yaml:"dialer-proxy" json:"dialer-proxy"`
	WriteToSystem bool   `yaml:"write-to-system" json:"write-to-system"`
}

type RawTun struct {
	Enable              bool     `yaml:"enable" json:"enable"`
	Device              string   `yaml:"device" json:"device"`
	Stack               TUNStack `yaml:"stack" json:"stack"`
	DNSHijack           []string `yaml:"dns-hijack" json:"dns-hijack"`
	AutoRoute           bool     `yaml:"auto-route" json:"auto-route"`
	AutoDetectInterface bool     `yaml:"auto-detect-interface" json:"auto-detect-interface"`

	MTU                                   uint32         `yaml:"mtu" json:"mtu,omitempty"`
	GSO                                   bool           `yaml:"gso" json:"gso,omitempty"`
	GSOMaxSize                            uint32         `yaml:"gso-max-size" json:"gso-max-size,omitempty"`
	Inet6Address                          []netip.Prefix `yaml:"inet6-address" json:"inet6-address,omitempty"`
	IPRoute2TableIndex                    int            `yaml:"iproute2-table-index" json:"iproute2-table-index,omitempty"`
	IPRoute2RuleIndex                     int            `yaml:"iproute2-rule-index" json:"iproute2-rule-index,omitempty"`
	AutoRedirect                          bool           `yaml:"auto-redirect" json:"auto-redirect,omitempty"`
	AutoRedirectInputMark                 uint32         `yaml:"auto-redirect-input-mark" json:"auto-redirect-input-mark,omitempty"`
	AutoRedirectOutputMark                uint32         `yaml:"auto-redirect-output-mark" json:"auto-redirect-output-mark,omitempty"`
	AutoRedirectIPRoute2FallbackRuleIndex int            `yaml:"auto-redirect-iproute2-fallback-rule-index" json:"auto-redirect-iproute2-fallback-rule-index,omitempty"`
	LoopbackAddress                       []netip.Addr   `yaml:"loopback-address" json:"loopback-address,omitempty"`
	StrictRoute                           bool           `yaml:"strict-route" json:"strict-route,omitempty"`
	RouteAddress                          []netip.Prefix `yaml:"route-address" json:"route-address,omitempty"`
	RouteAddressSet                       []string       `yaml:"route-address-set" json:"route-address-set,omitempty"`
	RouteExcludeAddress                   []netip.Prefix `yaml:"route-exclude-address" json:"route-exclude-address,omitempty"`
	RouteExcludeAddressSet                []string       `yaml:"route-exclude-address-set" json:"route-exclude-address-set,omitempty"`
	IncludeInterface                      []string       `yaml:"include-interface" json:"include-interface,omitempty"`
	ExcludeInterface                      []string       `yaml:"exclude-interface" json:"exclude-interface,omitempty"`
	IncludeUID                            []uint32       `yaml:"include-uid" json:"include-uid,omitempty"`
	IncludeUIDRange                       []string       `yaml:"include-uid-range" json:"include-uid-range,omitempty"`
	ExcludeUID                            []uint32       `yaml:"exclude-uid" json:"exclude-uid,omitempty"`
	ExcludeUIDRange                       []string       `yaml:"exclude-uid-range" json:"exclude-uid-range,omitempty"`
	ExcludeSrcPort                        []uint16       `yaml:"exclude-src-port" json:"exclude-src-port,omitempty"`
	ExcludeSrcPortRange                   []string       `yaml:"exclude-src-port-range" json:"exclude-src-port-range,omitempty"`
	ExcludeDstPort                        []uint16       `yaml:"exclude-dst-port" json:"exclude-dst-port,omitempty"`
	ExcludeDstPortRange                   []string       `yaml:"exclude-dst-port-range" json:"exclude-dst-port-range,omitempty"`
	IncludeAndroidUser                    []int          `yaml:"include-android-user" json:"include-android-user,omitempty"`
	IncludePackage                        []string       `yaml:"include-package" json:"include-package,omitempty"`
	ExcludePackage                        []string       `yaml:"exclude-package" json:"exclude-package,omitempty"`
	IncludeMACAddress                     []string       `yaml:"include-mac-address" json:"include-mac-address,omitempty"`
	ExcludeMACAddress                     []string       `yaml:"exclude-mac-address" json:"exclude-mac-address,omitempty"`
	EndpointIndependentNat                bool           `yaml:"endpoint-independent-nat" json:"endpoint-independent-nat,omitempty"`
	UDPTimeout                            int64          `yaml:"udp-timeout" json:"udp-timeout,omitempty"`
	DisableICMPForwarding                 bool           `yaml:"disable-icmp-forwarding" json:"disable-icmp-forwarding,omitempty"`
	FileDescriptor                        int            `yaml:"file-descriptor" json:"file-descriptor"`
	Inet4RouteAddress                     []netip.Prefix `yaml:"inet4-route-address" json:"inet4-route-address,omitempty"`
	Inet6RouteAddress                     []netip.Prefix `yaml:"inet6-route-address" json:"inet6-route-address,omitempty"`
	Inet4RouteExcludeAddress              []netip.Prefix `yaml:"inet4-route-exclude-address" json:"inet4-route-exclude-address,omitempty"`
	Inet6RouteExcludeAddress              []netip.Prefix `yaml:"inet6-route-exclude-address" json:"inet6-route-exclude-address,omitempty"`
	RecvMsgX                              bool           `yaml:"recvmsgx" json:"recvmsgx,omitempty"`
	SendMsgX                              bool           `yaml:"sendmsgx" json:"sendmsgx,omitempty"`
}

type RawTuicServer struct {
	Enable                bool              `yaml:"enable" json:"enable"`
	Listen                string            `yaml:"listen" json:"listen"`
	Token                 []string          `yaml:"token" json:"token"`
	Users                 map[string]string `yaml:"users" json:"users,omitempty"`
	Certificate           string            `yaml:"certificate" json:"certificate"`
	PrivateKey            string            `yaml:"private-key" json:"private-key"`
	CongestionController  string            `yaml:"congestion-controller" json:"congestion-controller,omitempty"`
	MaxIdleTime           int               `yaml:"max-idle-time" json:"max-idle-time,omitempty"`
	AuthenticationTimeout int               `yaml:"authentication-timeout" json:"authentication-timeout,omitempty"`
	ALPN                  []string          `yaml:"alpn" json:"alpn,omitempty"`
	MaxUdpRelayPacketSize int               `yaml:"max-udp-relay-packet-size" json:"max-udp-relay-packet-size,omitempty"`
	CWND                  int               `yaml:"cwnd" json:"cwnd,omitempty"`
}

type RawIPTables struct {
	Enable           bool     `yaml:"enable" json:"enable"`
	InboundInterface string   `yaml:"inbound-interface" json:"inbound-interface"`
	Bypass           []string `yaml:"bypass" json:"bypass"`
	DnsRedirect      bool     `yaml:"dns-redirect" json:"dns-redirect"`
}

type RawExperimental struct {
	Fingerprints     []string `yaml:"fingerprints" json:"fingerprints"`
	QUICGoDisableGSO bool     `yaml:"quic-go-disable-gso" json:"quic-go-disable-gso"`
	QUICGoDisableECN bool     `yaml:"quic-go-disable-ecn" json:"quic-go-disable-ecn"`
	IP4PEnable       bool     `yaml:"dialer-ip4p-convert" json:"dialer-ip4p-convert"`
}

type RawProfile struct {
	StoreSelected bool `yaml:"store-selected" json:"store-selected"`
	StoreFakeIP   bool `yaml:"store-fake-ip" json:"store-fake-ip"`
}

type RawGeoXUrl struct {
	GeoIp   string `yaml:"geoip" json:"geoip"`
	Mmdb    string `yaml:"mmdb" json:"mmdb"`
	ASN     string `yaml:"asn" json:"asn"`
	GeoSite string `yaml:"geosite" json:"geosite"`
}

type RawSniffer struct {
	Enable          bool                         `yaml:"enable" json:"enable"`
	OverrideDest    bool                         `yaml:"override-destination" json:"override-destination"`
	Sniffing        []string                     `yaml:"sniffing" json:"sniffing"`
	ForceDomain     []string                     `yaml:"force-domain" json:"force-domain"`
	SkipSrcAddress  []string                     `yaml:"skip-src-address" json:"skip-src-address"`
	SkipDstAddress  []string                     `yaml:"skip-dst-address" json:"skip-dst-address"`
	SkipDomain      []string                     `yaml:"skip-domain" json:"skip-domain"`
	Ports           []string                     `yaml:"port-whitelist" json:"port-whitelist"`
	ForceDnsMapping bool                         `yaml:"force-dns-mapping" json:"force-dns-mapping"`
	ParsePureIp     bool                         `yaml:"parse-pure-ip" json:"parse-pure-ip"`
	Sniff           map[string]RawSniffingConfig `yaml:"sniff" json:"sniff"`
}

type RawSniffingConfig struct {
	Ports        []string `yaml:"ports" json:"ports"`
	OverrideDest *bool    `yaml:"override-destination" json:"override-destination"`
}

type RawTLS struct {
	Certificate     string   `yaml:"certificate" json:"certificate"`
	PrivateKey      string   `yaml:"private-key" json:"private-key"`
	ClientAuthType  string   `yaml:"client-auth-type" json:"client-auth-type"`
	ClientAuthCert  string   `yaml:"client-auth-cert" json:"client-auth-cert"`
	EchKey          string   `yaml:"ech-key" json:"ech-key"`
	CustomTrustCert []string `yaml:"custom-certifactes" json:"custom-certifactes"`
}

type RuleProvider struct {
	Type     string `yaml:"type" json:"type"`
	Behavior string `yaml:"behavior" json:"behavior"`
	URL      string `yaml:"url" json:"url"`
	Path     string `yaml:"path" json:"path"`
	Interval int    `yaml:"interval" json:"interval"`
}

type RuleSegments []string

type RawConfig struct {
	Port                    int             `yaml:"port" json:"port"`
	SocksPort               int             `yaml:"socks-port" json:"socks-port"`
	RedirPort               int             `yaml:"redir-port" json:"redir-port"`
	TProxyPort              int             `yaml:"tproxy-port" json:"tproxy-port"`
	MixedPort               int             `yaml:"mixed-port" json:"mixed-port"`
	ShadowSocksConfig       string          `yaml:"ss-config" json:"ss-config"`
	VmessConfig             string          `yaml:"vmess-config" json:"vmess-config"`
	InboundTfo              bool            `yaml:"inbound-tfo" json:"inbound-tfo"`
	InboundMPTCP            bool            `yaml:"inbound-mptcp" json:"inbound-mptcp"`
	Authentication          []string        `yaml:"authentication" json:"authentication"`
	SkipAuthPrefixes        []netip.Prefix  `yaml:"skip-auth-prefixes" json:"skip-auth-prefixes"`
	LanAllowedIPs           []netip.Prefix  `yaml:"lan-allowed-ips" json:"lan-allowed-ips"`
	LanDisAllowedIPs        []netip.Prefix  `yaml:"lan-disallowed-ips" json:"lan-disallowed-ips"`
	AllowLan                bool            `yaml:"allow-lan" json:"allow-lan"`
	BindAddress             string          `yaml:"bind-address" json:"bind-address"`
	Mode                    TunnelMode      `yaml:"mode" json:"mode"`
	UnifiedDelay            bool            `yaml:"unified-delay" json:"unified-delay"`
	LogLevel                LogLevel        `yaml:"log-level" json:"log-level"`
	IPv6                    bool            `yaml:"ipv6" json:"ipv6"`
	ExternalController      string          `yaml:"external-controller" json:"external-controller"`
	ExternalControllerPipe  string          `yaml:"external-controller-pipe" json:"external-controller-pipe"`
	ExternalControllerUnix  string          `yaml:"external-controller-unix" json:"external-controller-unix"`
	ExternalControllerTLS   string          `yaml:"external-controller-tls" json:"external-controller-tls"`
	ExternalControllerCors  RawCors         `yaml:"external-controller-cors" json:"external-controller-cors"`
	ExternalUI              string          `yaml:"external-ui" json:"external-ui"`
	ExternalUIURL           string          `yaml:"external-ui-url" json:"external-ui-url"`
	ExternalUIName          string          `yaml:"external-ui-name" json:"external-ui-name"`
	ExternalDohServer       string          `yaml:"external-doh-server" json:"external-doh-server"`
	Secret                  string          `yaml:"secret" json:"secret"`
	Interface               string          `yaml:"interface-name" json:"interface-name"`
	RoutingMark             int             `yaml:"routing-mark" json:"routing-mark"`
	Tunnels                 []Tunnel        `yaml:"tunnels" json:"tunnels"`
	GeoAutoUpdate           bool            `yaml:"geo-auto-update" json:"geo-auto-update"`
	GeoUpdateInterval       int             `yaml:"geo-update-interval" json:"geo-update-interval"`
	GeodataMode             bool            `yaml:"geodata-mode" json:"geodata-mode"`
	GeodataLoader           string          `yaml:"geodata-loader" json:"geodata-loader"`
	GeositeMatcher          string          `yaml:"geosite-matcher" json:"geosite-matcher"`
	TCPConcurrent           bool            `yaml:"tcp-concurrent" json:"tcp-concurrent"`
	FindProcessMode         FindProcessMode `yaml:"find-process-mode" json:"find-process-mode"`
	GlobalClientFingerprint string          `yaml:"global-client-fingerprint" json:"global-client-fingerprint"`
	GlobalUA                string          `yaml:"global-ua" json:"global-ua"`
	ETagSupport             bool            `yaml:"etag-support" json:"etag-support"`
	KeepAliveIdle           int             `yaml:"keep-alive-idle" json:"keep-alive-idle"`
	KeepAliveInterval       int             `yaml:"keep-alive-interval" json:"keep-alive-interval"`
	DisableKeepAlive        bool            `yaml:"disable-keep-alive" json:"disable-keep-alive"`

	ProxyProvider map[string]RuleProvider   `yaml:"proxy-providers" json:"proxy-providers"`
	RuleProvider  map[string]map[string]any `yaml:"rule-providers" json:"rule-providers"`
	Proxy         []map[string]any          `yaml:"proxies" json:"proxies"`
	ProxyGroup    []map[string]any          `yaml:"proxy-groups" json:"proxy-groups"`
	Rule          []string                  `yaml:"rules" json:"rule"`
	SubRules      map[string][]string       `yaml:"sub-rules" json:"sub-rules"`
	Listeners     []map[string]any          `yaml:"listeners" json:"listeners"`
	Hosts         map[string]any            `yaml:"hosts" json:"hosts"`
	DNS           RawDNS                    `yaml:"dns" json:"dns"`
	NTP           RawNTP                    `yaml:"ntp" json:"ntp"`
	Tun           RawTun                    `yaml:"tun" json:"tun"`
	TuicServer    RawTuicServer             `yaml:"tuic-server" json:"tuic-server"`
	IPTables      RawIPTables               `yaml:"iptables" json:"iptables"`
	Experimental  RawExperimental           `yaml:"experimental" json:"experimental"`
	Profile       RawProfile                `yaml:"profile" json:"profile"`
	GeoXUrl       RawGeoXUrl                `yaml:"geox-url" json:"geox-url"`
	Sniffer       RawSniffer                `yaml:"sniffer" json:"sniffer"`
	TLS           RawTLS                    `yaml:"tls" json:"tls"`

	ClashForAndroid RawClashForAndroid `yaml:"clash-for-android" json:"clash-for-android"`
}

func DefaultRawConfig() *RawConfig {
	return &RawConfig{
		AllowLan:          false,
		BindAddress:       "*",
		LanAllowedIPs:     []netip.Prefix{netip.MustParsePrefix("0.0.0.0/0"), netip.MustParsePrefix("::/0")},
		IPv6:              true,
		Mode:              Rule,
		GeoAutoUpdate:     false,
		GeoUpdateInterval: 24,
		GeodataMode:       false,
		GeodataLoader:     "memconservative",
		UnifiedDelay:      false,
		Authentication:    []string{},
		LogLevel:          INFO,
		Hosts:             map[string]any{},
		Rule:              []string{},
		Proxy:             []map[string]any{},
		ProxyGroup:        []map[string]any{},
		TCPConcurrent:     false,
		FindProcessMode:   FindProcessStrict,
		GlobalUA:          "clash.meta/" + defaultMihomoVersion,
		ETagSupport:       true,
		DNS: RawDNS{
			Enable:         false,
			IPv6:           false,
			UseHosts:       true,
			UseSystemHosts: true,
			IPv6Timeout:    100,
			EnhancedMode:   DNSMapping,
			FakeIPRange:    "198.18.0.1/16",
			FakeIPTTL:      1,
			FallbackFilter: RawFallbackFilter{
				GeoIP:     true,
				GeoIPCode: "CN",
				IPCIDR:    []string{},
				GeoSite:   []string{},
			},
			DefaultNameserver: []string{
				"114.114.114.114",
				"223.5.5.5",
				"8.8.8.8",
				"1.0.0.1",
			},
			NameServer: []string{
				"https://doh.pub/dns-query",
				"tls://223.5.5.5:853",
			},
			FakeIPFilter: []string{
				"dns.msftnsci.com",
				"www.msftnsci.com",
				"www.msftconnecttest.com",
			},
			FakeIPFilterMode: FilterBlackList,
		},
		NTP: RawNTP{
			Enable:        false,
			WriteToSystem: false,
			Server:        "time.apple.com",
			Port:          123,
			Interval:      30,
		},
		Tun: RawTun{
			Enable:              false,
			Device:              "",
			Stack:               TunGvisor,
			DNSHijack:           []string{"0.0.0.0:53"},
			AutoRoute:           true,
			AutoDetectInterface: true,
			Inet6Address:        []netip.Prefix{netip.MustParsePrefix("fdfe:dcba:9876::1/126")},
			RecvMsgX:            true,
			SendMsgX:            false,
		},
		TuicServer: RawTuicServer{
			Enable:                false,
			Token:                 nil,
			Users:                 nil,
			Certificate:           "",
			PrivateKey:            "",
			Listen:                "",
			CongestionController:  "",
			MaxIdleTime:           15000,
			AuthenticationTimeout: 1000,
			ALPN:                  []string{"h3"},
			MaxUdpRelayPacketSize: 1500,
		},
		IPTables: RawIPTables{
			Enable:           false,
			InboundInterface: "lo",
			Bypass:           []string{},
			DnsRedirect:      true,
		},
		Experimental: RawExperimental{
			QUICGoDisableECN: true,
		},
		Profile: RawProfile{
			StoreSelected: true,
		},
		GeoXUrl: RawGeoXUrl{
			Mmdb:    "https://github.com/MetaCubeX/meta-rules-dat/releases/download/latest/geoip.metadb",
			ASN:     "https://github.com/MetaCubeX/meta-rules-dat/releases/download/latest/GeoLite2-ASN.mmdb",
			GeoIp:   "https://github.com/MetaCubeX/meta-rules-dat/releases/download/latest/geoip.dat",
			GeoSite: "https://github.com/MetaCubeX/meta-rules-dat/releases/download/latest/geosite.dat",
		},
		Sniffer: RawSniffer{
			Enable:          false,
			Sniff:           map[string]RawSniffingConfig{},
			ForceDomain:     []string{},
			SkipDomain:      []string{},
			Ports:           []string{},
			ForceDnsMapping: true,
			ParsePureIp:     true,
			OverrideDest:    true,
		},
		ExternalUIURL: "https://github.com/MetaCubeX/metacubexd/archive/refs/heads/gh-pages.zip",
		ExternalControllerCors: RawCors{
			AllowOrigins:        []string{"*"},
			AllowPrivateNetwork: true,
		},
	}
}

package config

import (
	"errors"
	"strings"
)

type TunnelMode int32

const (
	Global TunnelMode = iota
	Rule
	Direct
)

func (m TunnelMode) String() string {
	switch m {
	case Global:
		return "global"
	case Rule:
		return "rule"
	case Direct:
		return "direct"
	default:
		return "Unknown"
	}
}

func (m TunnelMode) MarshalText() ([]byte, error) {
	return []byte(m.String()), nil
}

func (m *TunnelMode) UnmarshalText(data []byte) error {
	switch strings.ToLower(string(data)) {
	case Global.String():
		*m = Global
	case Rule.String():
		*m = Rule
	case Direct.String():
		*m = Direct
	default:
		return errors.New("invalid mode")
	}
	return nil
}

type LogLevel int

const (
	DEBUG LogLevel = iota
	INFO
	WARNING
	ERROR
	SILENT
)

func (l LogLevel) String() string {
	switch l {
	case INFO:
		return "info"
	case WARNING:
		return "warning"
	case ERROR:
		return "error"
	case DEBUG:
		return "debug"
	case SILENT:
		return "silent"
	default:
		return "unknown"
	}
}

func (l LogLevel) MarshalText() ([]byte, error) {
	return []byte(l.String()), nil
}

func (l *LogLevel) UnmarshalText(data []byte) error {
	switch strings.ToLower(string(data)) {
	case ERROR.String():
		*l = ERROR
	case WARNING.String():
		*l = WARNING
	case INFO.String():
		*l = INFO
	case DEBUG.String():
		*l = DEBUG
	case SILENT.String():
		*l = SILENT
	default:
		return errors.New("invalid log-level")
	}
	return nil
}

type FindProcessMode int32

const (
	FindProcessStrict FindProcessMode = iota
	FindProcessAlways
	FindProcessOff
)

func (m FindProcessMode) String() string {
	switch m {
	case FindProcessAlways:
		return "always"
	case FindProcessOff:
		return "off"
	default:
		return "strict"
	}
}

func (m FindProcessMode) MarshalText() ([]byte, error) {
	return []byte(m.String()), nil
}

func (m *FindProcessMode) UnmarshalText(data []byte) error {
	switch strings.ToLower(string(data)) {
	case FindProcessStrict.String():
		*m = FindProcessStrict
	case FindProcessAlways.String():
		*m = FindProcessAlways
	case FindProcessOff.String():
		*m = FindProcessOff
	default:
		return errors.New("invalid find process mode")
	}
	return nil
}

type DNSMode int

const (
	DNSNormal DNSMode = iota
	DNSFakeIP
	DNSMapping
	DNSHosts
)

func (m DNSMode) String() string {
	switch m {
	case DNSNormal:
		return "normal"
	case DNSFakeIP:
		return "fake-ip"
	case DNSMapping:
		return "redir-host"
	case DNSHosts:
		return "hosts"
	default:
		return "unknown"
	}
}

func (m DNSMode) MarshalText() ([]byte, error) {
	return []byte(m.String()), nil
}

func (m *DNSMode) UnmarshalText(data []byte) error {
	switch strings.ToLower(string(data)) {
	case DNSNormal.String():
		*m = DNSNormal
	case DNSFakeIP.String():
		*m = DNSFakeIP
	case DNSMapping.String():
		*m = DNSMapping
	default:
		return errors.New("invalid mode")
	}
	return nil
}

type FilterMode int

const (
	FilterBlackList FilterMode = iota
	FilterWhiteList
	FilterRule
)

func (m FilterMode) String() string {
	switch m {
	case FilterBlackList:
		return "blacklist"
	case FilterWhiteList:
		return "whitelist"
	case FilterRule:
		return "rule"
	default:
		return "unknown"
	}
}

func (m FilterMode) MarshalText() ([]byte, error) {
	return []byte(m.String()), nil
}

func (m *FilterMode) UnmarshalText(data []byte) error {
	switch strings.ToLower(string(data)) {
	case FilterBlackList.String():
		*m = FilterBlackList
	case FilterWhiteList.String():
		*m = FilterWhiteList
	case FilterRule.String():
		*m = FilterRule
	default:
		return errors.New("invalid mode")
	}
	return nil
}

type TUNStack int

const (
	TunGvisor TUNStack = iota
	TunSystem
	TunMixed
)

func (s TUNStack) String() string {
	switch s {
	case TunGvisor:
		return "gVisor"
	case TunSystem:
		return "System"
	case TunMixed:
		return "Mixed"
	default:
		return "unknown"
	}
}

func (s TUNStack) MarshalText() ([]byte, error) {
	return []byte(s.String()), nil
}

func (s *TUNStack) UnmarshalText(data []byte) error {
	switch strings.ToLower(string(data)) {
	case strings.ToLower(TunGvisor.String()):
		*s = TunGvisor
	case strings.ToLower(TunSystem.String()):
		*s = TunSystem
	case strings.ToLower(TunMixed.String()):
		*s = TunMixed
	default:
		return errors.New("invalid tun stack")
	}
	return nil
}

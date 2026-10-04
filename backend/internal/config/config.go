package config

import (
	"errors"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	CorrectionWorkerEnabled bool
	AppEnv                  string
	PublicOrigin            string
	HTTPAddr                string
	DatabaseURL             string
	ShutdownTimeout         time.Duration
}

func Load() (Config, error) {
	c := Config{HTTPAddr: os.Getenv("HTTP_ADDR"), DatabaseURL: os.Getenv("DATABASE_URL"), ShutdownTimeout: 5 * time.Second}
	c.CorrectionWorkerEnabled = true
	if value := os.Getenv("CORRECTION_WORKER_ENABLED"); value != "" {
		switch value {
		case "true":
		case "false":
			c.CorrectionWorkerEnabled = false
		default:
			return Config{}, errors.New("invalid CORRECTION_WORKER_ENABLED")
		}
	}
	c.AppEnv = os.Getenv("APP_ENV")
	if c.AppEnv == "" {
		c.AppEnv = "development"
	}
	if c.AppEnv != "development" && c.AppEnv != "production" {
		return Config{}, errors.New("invalid APP_ENV")
	}
	if c.HTTPAddr == "" {
		c.HTTPAddr = "127.0.0.1:8080"
	}
	_, port, err := net.SplitHostPort(c.HTTPAddr)
	n, portErr := strconv.Atoi(port)
	if err != nil || portErr != nil || n < 1 || n > 65535 {
		return Config{}, errors.New("invalid HTTP_ADDR")
	}
	u, err := url.Parse(c.DatabaseURL)
	if err != nil || (u.Scheme != "postgres" && u.Scheme != "postgresql") || u.Path == "" || u.Path == "/" {
		return Config{}, errors.New("invalid DATABASE_URL")
	}
	if value := os.Getenv("SHUTDOWN_TIMEOUT"); value != "" {
		c.ShutdownTimeout, err = time.ParseDuration(value)
		if err != nil || c.ShutdownTimeout <= 0 || c.ShutdownTimeout > time.Minute {
			return Config{}, errors.New("invalid SHUTDOWN_TIMEOUT")
		}
	}
	c.PublicOrigin = os.Getenv("AUTH_PUBLIC_ORIGIN")
	if c.PublicOrigin == "" {
		if c.AppEnv == "production" {
			return Config{}, errors.New("production AUTH_PUBLIC_ORIGIN required")
		}
	} else {
		c.PublicOrigin, err = NormalizeAuthOrigin(c.PublicOrigin, c.AppEnv == "production")
		if err != nil {
			return Config{}, err
		}
		if c.AppEnv == "development" && strings.HasPrefix(c.PublicOrigin, "http:") {
			host, _, _ := net.SplitHostPort(c.HTTPAddr)
			if !IsLoopbackHost(host) {
				return Config{}, errors.New("HTTP development auth requires loopback HTTP_ADDR")
			}
		}
	}
	return c, nil
}
func IsLoopbackHost(host string) bool {
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}
func NormalizeAuthOrigin(raw string, production bool) (string, error) {
	fail := func() (string, error) { return "", errors.New("invalid AUTH_PUBLIC_ORIGIN") }
	u, err := url.Parse(raw)
	if err != nil || u == nil || u.Opaque != "" || u.User != nil || u.Host == "" || u.Path != "" || u.RawPath != "" || u.RawQuery != "" || u.ForceQuery || u.Fragment != "" || u.RawFragment != "" || (u.Scheme != "http" && u.Scheme != "https") {
		return fail()
	}
	if strings.HasSuffix(u.Host, ":") {
		return fail()
	}
	hostname := strings.ToLower(u.Hostname())
	if hostname == "" {
		return fail()
	}
	for _, r := range hostname {
		if r > 127 || r < 33 {
			return fail()
		}
	}
	port := u.Port()
	if port != "" {
		n, err := strconv.Atoi(port)
		if err != nil || n < 1 || n > 65535 {
			return fail()
		}
	}
	if ip := net.ParseIP(hostname); ip != nil {
		if strings.Contains(hostname, ":") {
			words := make([]string, 8)
			bestStart, bestLength := -1, 0
			for i := 0; i < 8; {
				value := uint16(ip[2*i])<<8 | uint16(ip[2*i+1])
				words[i] = strconv.FormatUint(uint64(value), 16)
				i++
			}
			for i := 0; i < 8; {
				if words[i] != "0" {
					i++
					continue
				}
				start := i
				for i < 8 && words[i] == "0" {
					i++
				}
				if i-start > bestLength {
					bestStart, bestLength = start, i-start
				}
			}
			if bestLength > 1 {
				hostname = strings.Join(words[:bestStart], ":") + "::" + strings.Join(words[bestStart+bestLength:], ":")
			} else {
				hostname = strings.Join(words, ":")
			}
		} else {
			hostname = ip.String()
		}
	} else {
		numeric := true
		for _, r := range hostname {
			if !(r >= '0' && r <= '9' || r == '.') {
				numeric = false
			}
		}
		if numeric {
			return fail()
		}
	}
	if production && u.Scheme != "https" {
		return fail()
	}
	if !production && u.Scheme == "http" && !IsLoopbackHost(hostname) {
		return fail()
	}
	if port != "" {
		n, _ := strconv.Atoi(port)
		port = strconv.Itoa(n)
	}
	if (u.Scheme == "https" && port == "443") || (u.Scheme == "http" && port == "80") {
		port = ""
	}
	host := hostname
	if strings.Contains(host, ":") {
		if net.ParseIP(host) == nil {
			return fail()
		}
		host = "[" + host + "]"
	}
	if port != "" {
		host = net.JoinHostPort(hostname, port)
	}
	normalized := url.URL{Scheme: u.Scheme, Host: host}
	return normalized.String(), nil
}

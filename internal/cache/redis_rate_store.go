package cache

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
	"time"

	"car-rental-system/internal/middleware"
)

type RedisRateLimitStore struct {
	address  string
	username string
	password string
	db       int
	timeout  time.Duration
}

func NewRedisRateLimitStore(rawURL string, timeout time.Duration) (*RedisRateLimitStore, error) {
	if rawURL == "" {
		return nil, nil
	}
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return nil, err
	}
	if parsed.Scheme != "redis" {
		return nil, fmt.Errorf("unsupported redis scheme %q", parsed.Scheme)
	}
	address := parsed.Host
	if !strings.Contains(address, ":") {
		address += ":6379"
	}
	db := 0
	if path := strings.Trim(parsed.Path, "/"); path != "" {
		db, err = strconv.Atoi(path)
		if err != nil {
			return nil, err
		}
	}
	password, _ := parsed.User.Password()
	if timeout <= 0 {
		timeout = 500 * time.Millisecond
	}
	return &RedisRateLimitStore{address: address, username: parsed.User.Username(), password: password, db: db, timeout: timeout}, nil
}

func (s *RedisRateLimitStore) Increment(ctx context.Context, key string, window time.Duration) (middleware.RateLimitState, error) {
	var zero middleware.RateLimitState
	conn, err := (&net.Dialer{Timeout: s.timeout}).DialContext(ctx, "tcp", s.address)
	if err != nil {
		return zero, err
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(s.timeout))

	reader := bufio.NewReader(conn)
	if s.password != "" {
		if s.username != "" {
			if err := s.sendExpectOK(conn, reader, "AUTH", s.username, s.password); err != nil {
				return zero, err
			}
		} else if err := s.sendExpectOK(conn, reader, "AUTH", s.password); err != nil {
			return zero, err
		}
	}
	if s.db > 0 {
		if err := s.sendExpectOK(conn, reader, "SELECT", strconv.Itoa(s.db)); err != nil {
			return zero, err
		}
	}

	if err := writeCommand(conn, "INCR", key); err != nil {
		return zero, err
	}
	count, err := readInt(reader)
	if err != nil {
		return zero, err
	}
	if count == 1 {
		if err := writeCommand(conn, "EXPIRE", key, strconv.Itoa(int(window.Seconds()))); err != nil {
			return zero, err
		}
		if _, err := readInt(reader); err != nil {
			return zero, err
		}
	}
	if err := writeCommand(conn, "TTL", key); err != nil {
		return zero, err
	}
	ttl, err := readInt(reader)
	if err != nil {
		return zero, err
	}
	if ttl < 0 {
		ttl = int(window.Seconds())
	}

	return middleware.RateLimitState{Count: count, ResetAfter: time.Duration(ttl) * time.Second}, nil
}

func (s *RedisRateLimitStore) sendExpectOK(conn net.Conn, reader *bufio.Reader, args ...string) error {
	if err := writeCommand(conn, args...); err != nil {
		return err
	}
	reply, err := readLine(reader)
	if err != nil {
		return err
	}
	if strings.HasPrefix(reply, "+OK") {
		return nil
	}
	if strings.HasPrefix(reply, "-") {
		return errors.New(strings.TrimSpace(reply[1:]))
	}
	return fmt.Errorf("unexpected redis reply %q", reply)
}

func writeCommand(conn net.Conn, args ...string) error {
	var b strings.Builder
	b.WriteString("*")
	b.WriteString(strconv.Itoa(len(args)))
	b.WriteString("\r\n")
	for _, arg := range args {
		b.WriteString("$")
		b.WriteString(strconv.Itoa(len(arg)))
		b.WriteString("\r\n")
		b.WriteString(arg)
		b.WriteString("\r\n")
	}
	_, err := conn.Write([]byte(b.String()))
	return err
}

func readInt(reader *bufio.Reader) (int, error) {
	line, err := readLine(reader)
	if err != nil {
		return 0, err
	}
	if strings.HasPrefix(line, "-") {
		return 0, errors.New(strings.TrimSpace(line[1:]))
	}
	if !strings.HasPrefix(line, ":") {
		return 0, fmt.Errorf("expected integer redis reply, got %q", line)
	}
	return strconv.Atoi(strings.TrimSpace(line[1:]))
}

func readLine(reader *bufio.Reader) (string, error) {
	line, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSuffix(strings.TrimSuffix(line, "\n"), "\r"), nil
}

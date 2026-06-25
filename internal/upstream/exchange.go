package upstream

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/miekg/dns"
	"github.com/nikitadybov/owrt-failsafe-dns-proxy-go/internal/config"
)

type Exchanger interface {
	Exchange(context.Context, config.Upstream, *dns.Msg) (*dns.Msg, time.Duration, error)
}

type DNSExchanger struct{}

func (DNSExchanger) Exchange(ctx context.Context, upstream config.Upstream, request *dns.Msg) (*dns.Msg, time.Duration, error) {
	response, rtt, err := exchange(ctx, upstream.Protocol, upstream.Endpoint(), request)
	if err != nil {
		return nil, rtt, err
	}
	if upstream.Protocol == "udp" && response.Truncated {
		response, rtt, err = exchange(ctx, "tcp", upstream.Endpoint(), request)
		if err != nil {
			return nil, rtt, fmt.Errorf("TCP retry after truncated UDP response: %w", err)
		}
	}
	if err := ValidateResponse(request, response); err != nil {
		return nil, rtt, err
	}
	return response, rtt, nil
}

func exchange(ctx context.Context, network, endpoint string, request *dns.Msg) (*dns.Msg, time.Duration, error) {
	client := &dns.Client{Net: network}
	response, rtt, err := client.ExchangeContext(ctx, request.Copy(), endpoint)
	if err != nil {
		return nil, rtt, fmt.Errorf("%s exchange with %s: %w", network, endpoint, err)
	}
	return response, rtt, nil
}

func ValidateResponse(request, response *dns.Msg) error {
	if response == nil {
		return errors.New("nil DNS response")
	}
	if !response.Response {
		return errors.New("DNS packet is not a response")
	}
	if response.Id != request.Id {
		return fmt.Errorf("transaction ID mismatch: got %d, want %d", response.Id, request.Id)
	}
	if len(response.Question) != len(request.Question) {
		return errors.New("question count mismatch")
	}
	for i := range request.Question {
		want, got := request.Question[i], response.Question[i]
		if !equalName(want.Name, got.Name) || want.Qtype != got.Qtype || want.Qclass != got.Qclass {
			return errors.New("question mismatch")
		}
	}
	return nil
}

func equalName(a, b string) bool {
	return strings.EqualFold(dns.Fqdn(a), dns.Fqdn(b))
}

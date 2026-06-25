package upstream

import (
	"context"
	"net"
	"strconv"
	"testing"
	"time"

	"github.com/miekg/dns"
	"github.com/nikitadybov/owrt-failsafe-dns-proxy-go/internal/config"
)

func TestValidateResponse(t *testing.T) {
	request := new(dns.Msg)
	request.SetQuestion("example.com.", dns.TypeA)
	response := new(dns.Msg)
	response.SetReply(request)
	if err := ValidateResponse(request, response); err != nil {
		t.Fatal(err)
	}
	response.Id++
	if err := ValidateResponse(request, response); err == nil {
		t.Fatal("expected ID mismatch")
	}
}

func TestUDPTruncationRetriesTCP(t *testing.T) {
	tcpListener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := tcpListener.Addr().(*net.TCPAddr).Port
	udpConn, err := net.ListenPacket("udp", net.JoinHostPort("127.0.0.1", strconv.Itoa(port)))
	if err != nil {
		tcpListener.Close()
		t.Fatal(err)
	}
	handler := dns.HandlerFunc(func(w dns.ResponseWriter, r *dns.Msg) {
		reply := new(dns.Msg)
		reply.SetReply(r)
		if _, ok := w.RemoteAddr().(*net.UDPAddr); ok {
			reply.Truncated = true
		} else {
			reply.Answer = append(reply.Answer, &dns.A{
				Hdr: dns.RR_Header{Name: r.Question[0].Name, Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: 30},
				A:   net.ParseIP("192.0.2.10"),
			})
		}
		_ = w.WriteMsg(reply)
	})
	udpServer := &dns.Server{PacketConn: udpConn, Handler: handler}
	tcpServer := &dns.Server{Listener: tcpListener, Handler: handler}
	go udpServer.ActivateAndServe()
	go tcpServer.ActivateAndServe()
	t.Cleanup(func() {
		_ = udpServer.Shutdown()
		_ = tcpServer.Shutdown()
	})

	request := new(dns.Msg)
	request.SetQuestion("example.com.", dns.TypeA)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	response, _, err := (DNSExchanger{}).Exchange(ctx, config.Upstream{
		Name: "test", Protocol: "udp", Address: "127.0.0.1", Port: port,
	}, request)
	if err != nil {
		t.Fatal(err)
	}
	if response.Truncated || len(response.Answer) != 1 {
		t.Fatalf("TCP retry did not return full answer: %#v", response)
	}
}

func TestMalformedUDPResponseIsRejected(t *testing.T) {
	conn, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	go func() {
		buffer := make([]byte, 4096)
		_, remote, readErr := conn.ReadFrom(buffer)
		if readErr == nil {
			_, _ = conn.WriteTo([]byte{0, 1, 2}, remote)
		}
	}()
	request := new(dns.Msg)
	request.SetQuestion("example.com.", dns.TypeA)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	_, _, err = (DNSExchanger{}).Exchange(ctx, config.Upstream{
		Name: "malformed", Protocol: "udp",
		Address: "127.0.0.1", Port: conn.LocalAddr().(*net.UDPAddr).Port,
	}, request)
	if err == nil {
		t.Fatal("expected malformed response error")
	}
}

func TestDelayedUDPResponseRespectsDeadline(t *testing.T) {
	conn, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	server := &dns.Server{
		PacketConn: conn,
		Handler: dns.HandlerFunc(func(w dns.ResponseWriter, r *dns.Msg) {
			time.Sleep(100 * time.Millisecond)
			reply := new(dns.Msg)
			reply.SetReply(r)
			_ = w.WriteMsg(reply)
		}),
	}
	go server.ActivateAndServe()
	defer server.Shutdown()

	request := new(dns.Msg)
	request.SetQuestion("example.com.", dns.TypeA)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	_, _, err = (DNSExchanger{}).Exchange(ctx, config.Upstream{
		Name: "slow", Protocol: "udp",
		Address: "127.0.0.1", Port: conn.LocalAddr().(*net.UDPAddr).Port,
	}, request)
	if err == nil {
		t.Fatal("expected deadline error")
	}
}

func FuzzValidateResponse(f *testing.F) {
	request := new(dns.Msg)
	request.SetQuestion("example.com.", dns.TypeA)
	valid, _ := request.Copy().Pack()
	f.Add(valid)
	f.Fuzz(func(t *testing.T, raw []byte) {
		var response dns.Msg
		if response.Unpack(raw) == nil {
			_ = ValidateResponse(request, &response)
		}
	})
}

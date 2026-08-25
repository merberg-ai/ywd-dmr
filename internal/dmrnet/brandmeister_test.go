package dmrnet

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"net"
	"strconv"
	"testing"
	"time"

	"github.com/merberg-ai/ywd-dmr/internal/config"
)

func TestBrandMeisterDeviceID(t *testing.T) {
	withoutESSID, err := brandMeisterDeviceID(config.RadioIdentity{DMRID: 1234567, ESSID: 0})
	if err != nil || withoutESSID != 1234567 {
		t.Fatalf("unexpected base ID: %d %v", withoutESSID, err)
	}
	withESSID, err := brandMeisterDeviceID(config.RadioIdentity{DMRID: 1234567, ESSID: 1})
	if err != nil || withESSID != 123456701 {
		t.Fatalf("unexpected hotspot ID: %d %v", withESSID, err)
	}
}

func TestBrandMeisterTesterCompletesHandshakeAndCloses(t *testing.T) {
	server, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0})
	if err != nil {
		t.Fatal(err)
	}
	defer server.Close()
	_ = server.SetDeadline(time.Now().Add(3 * time.Second))

	serverErr := make(chan error, 1)
	password := "test-hotspot-password"
	go func() {
		buffer := make([]byte, 512)
		n, remote, err := server.ReadFromUDP(buffer)
		if err != nil {
			serverErr <- err
			return
		}
		if n != 8 || string(buffer[:4]) != "RPTL" || binary.BigEndian.Uint32(buffer[4:8]) != 123456701 {
			serverErr <- fmt.Errorf("bad RPTL packet: %x", buffer[:n])
			return
		}

		salt := []byte{0x10, 0x20, 0x30, 0x40}
		if _, err := server.WriteToUDP(append([]byte("RPTACK"), salt...), remote); err != nil {
			serverErr <- err
			return
		}

		n, remote, err = server.ReadFromUDP(buffer)
		if err != nil {
			serverErr <- err
			return
		}
		if n != 40 || string(buffer[:4]) != "RPTK" || binary.BigEndian.Uint32(buffer[4:8]) != 123456701 {
			serverErr <- fmt.Errorf("bad RPTK packet: %x", buffer[:n])
			return
		}
		wantSecret := append(append([]byte(nil), salt...), []byte(password)...)
		wantDigest := sha256.Sum256(wantSecret)
		if !bytes.Equal(buffer[8:40], wantDigest[:]) {
			serverErr <- fmt.Errorf("bad RPTK digest")
			return
		}
		if _, err := server.WriteToUDP([]byte("RPTACK"), remote); err != nil {
			serverErr <- err
			return
		}

		n, remote, err = server.ReadFromUDP(buffer)
		if err != nil {
			serverErr <- err
			return
		}
		if n != 302 || string(buffer[:4]) != "RPTC" || binary.BigEndian.Uint32(buffer[4:8]) != 123456701 {
			serverErr <- fmt.Errorf("bad RPTC framing: len=%d", n)
			return
		}
		if got := string(buffer[8:16]); got != "N0CALL  " {
			serverErr <- fmt.Errorf("unexpected callsign field %q", got)
			return
		}
		if got := string(buffer[16:34]); got != "000000000000000000" {
			serverErr <- fmt.Errorf("unexpected software frequency fields %q", got)
			return
		}
		if buffer[97] != '4' {
			serverErr <- fmt.Errorf("expected simplex/software slot marker 4, got %q", buffer[97])
			return
		}
		if bytes.Contains(buffer[:n], []byte(password)) {
			serverErr <- fmt.Errorf("RPTC leaked password")
			return
		}
		if _, err := server.WriteToUDP([]byte("RPTACK"), remote); err != nil {
			serverErr <- err
			return
		}

		n, _, err = server.ReadFromUDP(buffer)
		if err != nil {
			serverErr <- err
			return
		}
		if n != 9 || string(buffer[:5]) != "RPTCL" || binary.BigEndian.Uint32(buffer[5:9]) != 123456701 {
			serverErr <- fmt.Errorf("bad RPTCL packet: %x", buffer[:n])
			return
		}
		serverErr <- nil
	}()

	addr := server.LocalAddr().(*net.UDPAddr)
	tester := &BrandMeisterTester{stageTimeout: 300 * time.Millisecond, attempts: 1}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	result, err := tester.Test(ctx, config.RadioIdentity{
		Callsign: "N0CALL",
		DMRID:    1234567,
		ESSID:    1,
	}, config.NetworkCandidate{
		Backend:       config.NetworkBackendBrandMeister,
		MasterAddress: "127.0.0.1",
		MasterPort:    addr.Port,
		Password:      password,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.OK || result.Reason != TestReasonOK {
		t.Fatalf("unexpected result: %+v", result)
	}
	if err := <-serverErr; err != nil {
		t.Fatal(err)
	}
}

func TestBrandMeisterTesterMapsAuthNAK(t *testing.T) {
	server, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0})
	if err != nil {
		t.Fatal(err)
	}
	defer server.Close()
	_ = server.SetDeadline(time.Now().Add(2 * time.Second))

	go func() {
		buffer := make([]byte, 512)
		_, remote, err := server.ReadFromUDP(buffer)
		if err != nil {
			return
		}
		_, _ = server.WriteToUDP(append([]byte("RPTACK"), 1, 2, 3, 4), remote)
		_, remote, err = server.ReadFromUDP(buffer)
		if err != nil {
			return
		}
		_, _ = server.WriteToUDP([]byte("MSTNAK"), remote)
	}()

	addr := server.LocalAddr().(*net.UDPAddr)
	tester := &BrandMeisterTester{stageTimeout: 250 * time.Millisecond, attempts: 1}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	result, err := tester.Test(ctx, config.RadioIdentity{Callsign: "N0CALL", DMRID: 1234567}, config.NetworkCandidate{
		Backend:       config.NetworkBackendBrandMeister,
		MasterAddress: "127.0.0.1",
		MasterPort:    addr.Port,
		Password:      "wrong",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.OK || result.Reason != TestReasonAuth {
		t.Fatalf("expected auth rejection, got %+v", result)
	}
	if bytes.Contains([]byte(result.Message), []byte("wrong")) {
		t.Fatal("result leaked password")
	}
}

func TestBrandMeisterTesterTimesOutWithoutReply(t *testing.T) {
	server, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0})
	if err != nil {
		t.Fatal(err)
	}
	defer server.Close()

	addr := server.LocalAddr().(*net.UDPAddr)
	tester := &BrandMeisterTester{stageTimeout: 40 * time.Millisecond, attempts: 1}
	ctx, cancel := context.WithTimeout(context.Background(), 250*time.Millisecond)
	defer cancel()
	result, err := tester.Test(ctx, config.RadioIdentity{Callsign: "N0CALL", DMRID: 1234567}, config.NetworkCandidate{
		Backend:       config.NetworkBackendBrandMeister,
		MasterAddress: "127.0.0.1",
		MasterPort:    addr.Port,
		Password:      "secret",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.OK || result.Reason != TestReasonTimeout {
		t.Fatalf("expected timeout, got %+v", result)
	}
}

func TestBrandMeisterTesterRejectsCallsignTooLongForHomebrew(t *testing.T) {
	tester := NewBrandMeisterTester()
	result, err := tester.Test(context.Background(), config.RadioIdentity{
		Callsign: "LONGCALL99",
		DMRID:    1234567,
	}, config.NetworkCandidate{
		Backend:       config.NetworkBackendBrandMeister,
		MasterAddress: "127.0.0.1",
		MasterPort:    62031,
		Password:      "secret",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.OK || result.Reason != TestReasonConfig {
		t.Fatalf("expected config failure, got %+v", result)
	}
}

func TestBrandMeisterConfigPacketLengthAndFields(t *testing.T) {
	packet, err := buildBrandMeisterConfigPacket(config.RadioIdentity{Callsign: "N0CALL"}, 1234567)
	if err != nil {
		t.Fatal(err)
	}
	if len(packet) != brandMeisterConfigPacketLength {
		t.Fatalf("expected 302 bytes, got %d", len(packet))
	}
	if string(packet[:4]) != "RPTC" {
		t.Fatalf("unexpected tag: %q", packet[:4])
	}
	if got := strconv.Itoa(int(binary.BigEndian.Uint32(packet[4:8]))); got != "1234567" {
		t.Fatalf("unexpected ID %s", got)
	}
}

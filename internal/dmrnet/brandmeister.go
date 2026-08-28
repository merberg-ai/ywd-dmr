package dmrnet

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/merberg-ai/ywd-dmr/internal/config"
)

const (
	brandMeisterConfigPacketLength = 302
	defaultStageTimeout            = 1500 * time.Millisecond
	defaultStageAttempts           = 1
	// BrandMeister accepted the temporary setup probe when this field used the
	// upstream MMDVM-Host date-style version 20260528, after rejecting YWD-DMR.
	// This follow-up probe uses a YWD-DMR-owned date-style identifier to test
	// whether the master requires the numeric/date format rather than that exact
	// upstream version. It remains Homebrew registration metadata, not a claim
	// that ywd-dmrd is MMDVMHost or controls an attached modem.
	brandMeisterSoftwareID = "20260827"
	brandMeisterPackageID  = "MMDVM_DMO"
)

var (
	rptACK = []byte("RPTACK")
	mstNAK = []byte("MSTNAK")
	mstCL  = []byte("MSTCL")
)

// BrandMeisterTester performs a short-lived Homebrew/MMDVM login probe. It
// never sends DMRD voice/data frames and never persists the submitted password.
type BrandMeisterTester struct {
	stageTimeout time.Duration
	attempts     int
}

func NewBrandMeisterTester() *BrandMeisterTester {
	return &BrandMeisterTester{
		stageTimeout: defaultStageTimeout,
		attempts:     defaultStageAttempts,
	}
}

func (t *BrandMeisterTester) Test(ctx context.Context, identity config.RadioIdentity, candidate config.NetworkCandidate) (TestResult, error) {
	start := time.Now()
	result := func(ok bool, reason TestReason, message string) TestResult {
		return TestResult{
			OK:         ok,
			Backend:    config.NetworkBackendBrandMeister,
			Reason:     reason,
			Message:    message,
			DurationMS: DurationMilliseconds(start),
		}
	}

	if candidate.Backend != config.NetworkBackendBrandMeister {
		return result(false, TestReasonUnavailable, "The selected network backend is not available for this tester."), nil
	}

	deviceID, err := brandMeisterDeviceID(identity)
	if err != nil {
		return result(false, TestReasonConfig, err.Error()), nil
	}

	configPacket, err := buildBrandMeisterConfigPacket(identity, deviceID, candidate.RegistrationFrequencyHz)
	if err != nil {
		return result(false, TestReasonConfig, err.Error()), nil
	}

	address := net.JoinHostPort(candidate.MasterAddress, strconv.Itoa(candidate.MasterPort))
	var dialer net.Dialer
	conn, err := dialer.DialContext(ctx, "udp", address)
	if err != nil {
		return result(false, TestReasonNetwork, "Could not resolve or open a UDP path to the BrandMeister master."), nil
	}
	defer conn.Close()

	login := make([]byte, 8)
	copy(login[:4], "RPTL")
	binary.BigEndian.PutUint32(login[4:8], deviceID)

	ack, reason, message := t.exchange(ctx, conn, login, TestReasonLogin, true)
	if reason != TestReasonOK {
		return result(false, reason, message), nil
	}
	salt := append([]byte(nil), ack[6:10]...)

	auth := make([]byte, 40)
	copy(auth[:4], "RPTK")
	binary.BigEndian.PutUint32(auth[4:8], deviceID)
	secret := make([]byte, 0, len(salt)+len(candidate.Password))
	secret = append(secret, salt...)
	secret = append(secret, candidate.Password...)
	digest := sha256.Sum256(secret)
	copy(auth[8:], digest[:])
	for i := range secret {
		secret[i] = 0
	}

	_, reason, message = t.exchange(ctx, conn, auth, TestReasonAuth, false)
	if reason != TestReasonOK {
		return result(false, reason, message), nil
	}

	_, reason, message = t.exchange(ctx, conn, configPacket, TestReasonConfig, false)
	if reason != TestReasonOK {
		return result(false, reason, message), nil
	}

	// The probe is complete after the configuration ACK. Close the temporary
	// session explicitly; this test never becomes the daemon's live connection.
	closePacket := make([]byte, 9)
	copy(closePacket[:5], "RPTCL")
	binary.BigEndian.PutUint32(closePacket[5:9], deviceID)
	_ = conn.SetWriteDeadline(time.Now().Add(250 * time.Millisecond))
	_, _ = conn.Write(closePacket)

	return result(true, TestReasonOK, "BrandMeister accepted login, hotspot authentication, and software-endpoint configuration."), nil
}

func (t *BrandMeisterTester) exchange(ctx context.Context, conn net.Conn, packet []byte, phase TestReason, requireSalt bool) ([]byte, TestReason, string) {
	attempts := t.attempts
	if attempts < 1 {
		attempts = 1
	}
	stageTimeout := t.stageTimeout
	if stageTimeout <= 0 {
		stageTimeout = defaultStageTimeout
	}

	buffer := make([]byte, 512)
	for attempt := 0; attempt < attempts; attempt++ {
		deadline := time.Now().Add(stageTimeout)
		if ctxDeadline, ok := ctx.Deadline(); ok && ctxDeadline.Before(deadline) {
			deadline = ctxDeadline
		}
		if err := conn.SetWriteDeadline(deadline); err != nil {
			return nil, TestReasonNetwork, "Could not prepare the UDP test socket."
		}
		if _, err := conn.Write(packet); err != nil {
			return nil, TestReasonNetwork, "Could not send the BrandMeister test packet."
		}
		if err := conn.SetReadDeadline(deadline); err != nil {
			return nil, TestReasonNetwork, "Could not prepare the UDP test socket."
		}

		for {
			n, err := conn.Read(buffer)
			if err != nil {
				if ctx.Err() != nil {
					return nil, TestReasonTimeout, "BrandMeister did not complete the test before the deadline."
				}
				var netErr net.Error
				if errors.As(err, &netErr) && netErr.Timeout() {
					break
				}
				return nil, TestReasonNetwork, "The UDP path to the BrandMeister master failed during the test."
			}

			response := buffer[:n]
			switch {
			case bytes.HasPrefix(response, mstNAK):
				return nil, phase, rejectionMessage(phase)
			case bytes.HasPrefix(response, mstCL):
				return nil, TestReasonUnavailable, "BrandMeister closed the temporary test session."
			case bytes.HasPrefix(response, rptACK):
				if requireSalt && len(response) < 10 {
					return nil, TestReasonUnavailable, "BrandMeister returned a malformed login acknowledgement."
				}
				return append([]byte(nil), response...), TestReasonOK, ""
			default:
				// Ignore unrelated packets and continue until this stage's bounded
				// deadline. The connected UDP socket already limits traffic to the
				// selected master endpoint.
			}
		}
	}

	return nil, TestReasonTimeout, "BrandMeister did not answer the temporary test session before the deadline."
}

func rejectionMessage(phase TestReason) string {
	switch phase {
	case TestReasonLogin:
		return "BrandMeister rejected the DMR/hotspot ID during login."
	case TestReasonAuth:
		return "BrandMeister rejected the Hotspot Security response. Verify the Hotspot Security password for the base DMR ID; it is separate from the SelfCare login password."
	case TestReasonConfig:
		return "BrandMeister rejected the Homebrew registration metadata after successful authentication. Verify the registration frequency and endpoint configuration."
	default:
		return "BrandMeister rejected the temporary test session."
	}
}

func brandMeisterDeviceID(identity config.RadioIdentity) (uint32, error) {
	if identity.DMRID < 1 || identity.DMRID > 9_999_999 || identity.ESSID < 0 || identity.ESSID > 99 {
		return 0, fmt.Errorf("the stored station DMR identity is not valid for BrandMeister")
	}

	id := uint64(identity.DMRID)
	if identity.ESSID != 0 {
		id = id*100 + uint64(identity.ESSID)
	}
	if id > uint64(^uint32(0)) {
		return 0, fmt.Errorf("the derived BrandMeister hotspot ID is out of range")
	}
	return uint32(id), nil
}

func buildBrandMeisterConfigPacket(identity config.RadioIdentity, deviceID uint32, registrationFrequencyHz int) ([]byte, error) {
	callsign := strings.ToUpper(strings.TrimSpace(identity.Callsign))
	if callsign == "" || len(callsign) > 8 {
		return nil, fmt.Errorf("the stored callsign does not fit the BrandMeister Homebrew 8-character station field")
	}
	if registrationFrequencyHz < config.BrandMeisterMinFrequencyHz || registrationFrequencyHz > config.BrandMeisterMaxFrequencyHz {
		return nil, fmt.Errorf("the BrandMeister Homebrew registration frequency is invalid")
	}

	packet := make([]byte, brandMeisterConfigPacketLength)
	for i := 8; i < len(packet); i++ {
		packet[i] = ' '
	}
	copy(packet[:4], "RPTC")
	binary.BigEndian.PutUint32(packet[4:8], deviceID)

	frequency := fmt.Sprintf("%09d", registrationFrequencyHz)
	putFixed(packet, 8, 8, callsign)
	putFixed(packet, 16, 9, frequency)
	putFixed(packet, 25, 9, frequency)
	// Homebrew/BrandMeister treats these as registration metadata. YWD-DMR
	// still has no RF transmit path; 01 W is the minimum informational value
	// used for compatibility rather than a claim about actual transmitter power.
	putFixed(packet, 34, 2, "01")
	putFixed(packet, 36, 2, "01")
	putFixed(packet, 38, 8, "0.000000")
	putFixed(packet, 46, 9, "00.000000")
	putFixed(packet, 55, 3, "000")
	putFixed(packet, 58, 20, "YWD-DMR software")
	putFixed(packet, 78, 19, "Software DMR client")
	putFixed(packet, 97, 1, "4")
	putFixed(packet, 98, 124, "https://github.com/merberg-ai/ywd-dmr")
	putFixed(packet, 222, 40, brandMeisterSoftwareID)
	putFixed(packet, 262, 40, brandMeisterPackageID)

	return packet, nil
}

func putFixed(dst []byte, offset, width int, value string) {
	if len(value) > width {
		value = value[:width]
	}
	copy(dst[offset:offset+width], value)
}

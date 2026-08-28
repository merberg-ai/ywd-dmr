package dmrnet

import (
	"strings"
	"testing"

	"github.com/merberg-ai/ywd-dmr/internal/config"
)

func TestBrandMeisterConfigPacketUsesYWDNumericProbeProfile(t *testing.T) {
	packet, err := buildBrandMeisterConfigPacket(
		config.RadioIdentity{Callsign: "N0CALL"},
		1234567,
		446_525_000,
	)
	if err != nil {
		t.Fatal(err)
	}

	softwareID := strings.TrimSpace(string(packet[222:262]))
	packageID := strings.TrimSpace(string(packet[262:302]))

	if softwareID != brandMeisterSoftwareID {
		t.Fatalf("unexpected software ID %q", softwareID)
	}
	if softwareID != "20260827" {
		t.Fatalf("expected YWD-DMR-owned numeric interoperability probe ID 20260827, got %q", softwareID)
	}
	if packageID != brandMeisterPackageID {
		t.Fatalf("unexpected package/profile ID %q", packageID)
	}
	if packageID != "MMDVM_DMO" {
		t.Fatalf("expected upstream simplex compatibility profile MMDVM_DMO, got %q", packageID)
	}
}

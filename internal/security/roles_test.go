package security

import "testing"

func TestRoleAllowsHierarchy(t *testing.T) {
	tests := []struct {
		name    string
		actual  string
		minimum Role
		want    bool
	}{
		{"observer can observe", "observer", RoleObserver, true},
		{"observer cannot operate", "observer", RoleOperator, false},
		{"observer cannot administer", "observer", RoleAdmin, false},
		{"operator can observe", "operator", RoleObserver, true},
		{"operator can operate", "operator", RoleOperator, true},
		{"operator cannot administer", "operator", RoleAdmin, false},
		{"admin can observe", "admin", RoleObserver, true},
		{"admin can operate", "admin", RoleOperator, true},
		{"admin can administer", "admin", RoleAdmin, true},
		{"unknown actual role fails closed", "superuser", RoleObserver, false},
		{"unknown minimum role fails closed", "admin", Role("future-role"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := RoleAllows(tt.actual, tt.minimum); got != tt.want {
				t.Fatalf("RoleAllows(%q, %q) = %v, want %v", tt.actual, tt.minimum, got, tt.want)
			}
		})
	}
}

package security

// Role is the server-side authorization level attached to an authenticated
// principal. The initial YWD-DMR model is intentionally small and hierarchical:
// Admin includes Operator privileges, and Operator includes Observer privileges.
type Role string

const (
	RoleObserver Role = "observer"
	RoleOperator Role = "operator"
	RoleAdmin    Role = "admin"
)

// RoleAllows reports whether actual satisfies the minimum required role.
// Unknown role strings fail closed.
func RoleAllows(actual string, minimum Role) bool {
	actualRank, actualOK := roleRank(Role(actual))
	minimumRank, minimumOK := roleRank(minimum)
	return actualOK && minimumOK && actualRank >= minimumRank
}

func roleRank(role Role) (int, bool) {
	switch role {
	case RoleObserver:
		return 1, true
	case RoleOperator:
		return 2, true
	case RoleAdmin:
		return 3, true
	default:
		return 0, false
	}
}

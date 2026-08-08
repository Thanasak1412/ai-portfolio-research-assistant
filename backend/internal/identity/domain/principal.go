package domain

type Principal struct{ userID UserID }

func NewPrincipal(userID UserID) (Principal, error) {
	if userID.IsZero() {
		return Principal{}, ErrUnauthenticated
	}
	return Principal{userID: userID}, nil
}

func (principal Principal) UserID() (UserID, bool) {
	if principal.userID.IsZero() {
		return UserID{}, false
	}
	return principal.userID, true
}

func (principal Principal) IsAuthenticated() bool { return !principal.userID.IsZero() }

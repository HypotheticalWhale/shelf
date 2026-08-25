// Package auth bridges Clerk sessions to Shelf's own user rows.
package auth

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"sync"

	"github.com/clerk/clerk-sdk-go/v2"
	clerkhttp "github.com/clerk/clerk-sdk-go/v2/http"
	clerkuser "github.com/clerk/clerk-sdk-go/v2/user"
	"github.com/samueltansats/shelf/apps/api/internal/store"
)

type ctxKey int

const userIDKey ctxKey = iota

// Authenticator verifies Clerk sessions and guarantees a matching local row.
type Authenticator struct {
	store *store.Store

	// provisioned remembers which Clerk IDs already have a local row. Without
	// it every authenticated request would pay a database lookup purely to
	// re-confirm something that can never become false.
	mu          sync.RWMutex
	provisioned map[string]struct{}
}

// New configures the Clerk SDK and returns an Authenticator.
func New(s *store.Store, secretKey string) *Authenticator {
	if secretKey != "" {
		clerk.SetKey(secretKey)
	}
	return &Authenticator{
		store:       s,
		provisioned: make(map[string]struct{}),
	}
}

// Middleware verifies the bearer token on the way in.
//
// Clerk's own middleware does the JWT work — fetching and caching the JSON web
// key, checking the signature and expiry — so verification never becomes a
// hand-rolled crypto path.
//
// A token that fails to verify does not fail the request. Clerk's default is to
// answer 401 for anything it cannot authenticate, which would take down public
// reads — a browse page, a game, someone's blog — for a visitor whose session
// merely expired. Authorisation is decided per route by RequireUser instead, so
// a bad token simply means "anonymous" and the public site keeps working.
func (a *Authenticator) Middleware(next http.Handler) http.Handler {
	anonymous := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r)
	})

	authenticate := clerkhttp.WithHeaderAuthorization(
		clerkhttp.AuthorizationFailureHandler(anonymous),
	)

	return authenticate(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims, ok := clerk.SessionClaimsFromContext(r.Context())
		if !ok || claims.Subject == "" {
			next.ServeHTTP(w, r)
			return
		}
		ctx := context.WithValue(r.Context(), userIDKey, claims.Subject)
		next.ServeHTTP(w, r.WithContext(ctx))
	}))
}

// UserID returns the authenticated Clerk user ID, or "" when signed out.
func UserID(ctx context.Context) string {
	id, _ := ctx.Value(userIDKey).(string)
	return id
}

// ErrUnauthenticated is returned when a request needs a signed-in user.
var ErrUnauthenticated = errors.New("authentication required")

// RequireUser returns the caller's Shelf user, creating it on first sight.
//
// Provisioning happens here rather than through a Clerk webhook: there is no
// endpoint to secure, no signature to verify, and no window in which a
// signed-in user has no row to hang ratings and posts off. The Clerk API is
// consulted only on the very first request from an account.
func (a *Authenticator) RequireUser(ctx context.Context) (store.User, error) {
	id := UserID(ctx)
	if id == "" {
		return store.User{}, ErrUnauthenticated
	}

	a.mu.RLock()
	_, known := a.provisioned[id]
	a.mu.RUnlock()

	if known {
		return a.store.GetUserByID(ctx, id)
	}

	u, err := a.store.GetUserByID(ctx, id)
	if err == nil {
		a.remember(id)
		return u, nil
	}
	if !errors.Is(err, store.ErrNotFound) {
		return store.User{}, err
	}

	username, display, avatar := a.profileFromClerk(ctx, id)
	u, err = a.store.EnsureUser(ctx, id, username, display, avatar)
	if err != nil {
		return store.User{}, err
	}
	a.remember(id)
	return u, nil
}

func (a *Authenticator) remember(id string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	// Bound the cache so a long-lived instance cannot grow without limit.
	if len(a.provisioned) > 10_000 {
		a.provisioned = make(map[string]struct{}, 1024)
	}
	a.provisioned[id] = struct{}{}
}

// profileFromClerk fetches display details for a new account. Failure is not
// fatal — a user with a generated username beats a failed request.
func (a *Authenticator) profileFromClerk(ctx context.Context, id string) (username, display, avatar string) {
	cu, err := clerkuser.Get(ctx, id)
	if err != nil || cu == nil {
		return "", "", ""
	}

	if cu.Username != nil {
		username = *cu.Username
	}

	var parts []string
	if cu.FirstName != nil && *cu.FirstName != "" {
		parts = append(parts, *cu.FirstName)
	}
	if cu.LastName != nil && *cu.LastName != "" {
		parts = append(parts, *cu.LastName)
	}
	display = strings.Join(parts, " ")

	if username == "" {
		// Fall back to the local part of the primary email so people get a
		// recognisable handle instead of a random one.
		for _, e := range cu.EmailAddresses {
			if e != nil && e.EmailAddress != "" {
				if local, _, ok := strings.Cut(e.EmailAddress, "@"); ok {
					username = local
					break
				}
			}
		}
	}
	if display == "" {
		display = username
	}
	if cu.ImageURL != nil {
		avatar = *cu.ImageURL
	}
	return username, display, avatar
}

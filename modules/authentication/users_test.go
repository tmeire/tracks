package authentication

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tmeire/tracks/database"
	"github.com/tmeire/tracks/database/sqlite"
	"github.com/tmeire/tracks/session"
	"github.com/tmeire/tracks/session/inmemory"
)

func TestActivationTokenExpiry(t *testing.T) {
	u := &User{}
	tok := u.GenerateActivationToken()
	assert.True(t, u.ActivationTokenValid(tok))

	exp, ok := u.ActivationTokenExpiry()
	require.True(t, ok)
	assert.WithinDuration(t, time.Now().Add(ActivationTokenTTL), exp, time.Minute)

	// wrong token, empty token
	assert.False(t, u.ActivationTokenValid(tok+"x"))
	assert.False(t, u.ActivationTokenValid(""))
	assert.False(t, (&User{}).ActivationTokenValid(""))

	// expired
	u.ActivationToken = strings.Split(tok, ".")[0] + "." + strconv.FormatInt(time.Now().Add(-time.Second).Unix(), 10)
	assert.False(t, u.ActivationTokenValid(u.ActivationToken))

	// legacy token without an expiry is treated as expired
	u.ActivationToken = strings.Split(tok, ".")[0]
	assert.False(t, u.ActivationTokenValid(u.ActivationToken))

	// tampering with the expiry produces a token that is not the stored one
	u.GenerateActivationToken()
	stored := u.ActivationToken
	forged := strings.Split(stored, ".")[0] + ".99999999999"
	assert.False(t, u.ActivationTokenValid(forged))
}

func setupUsers(t *testing.T) (*UsersResource, context.Context, session.Session) {
	t.Helper()
	db, err := sqlite.New(filepath.Join(t.TempDir(), "c.sqlite"))
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })
	_, err = db.Exec(`CREATE TABLE users (id TEXT PRIMARY KEY, email TEXT NOT NULL UNIQUE, name TEXT NOT NULL, password TEXT NOT NULL, activation_token TEXT, created_at TIMESTAMP NOT NULL, updated_at TIMESTAMP NOT NULL)`)
	require.NoError(t, err)

	ctx := database.WithDB(context.Background(), db)
	sess := inmemory.NewStore().Create(ctx)
	ctx = session.WithContext(ctx, sess)
	return &UsersResource{schema: NewSchema()}, ctx, sess
}

func postToken(ctx context.Context, ur *UsersResource, token string) (*httptest.ResponseRecorder, any, error) {
	form := url.Values{"token": {token}, "password": {"s3cret-pw"}, "password_confirmation": {"s3cret-pw"}}
	req := httptest.NewRequest(http.MethodPost, "/users/activate", strings.NewReader(form.Encode())).WithContext(ctx)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := ur.SetPasswordWithToken(req)
	return nil, resp, err
}

func TestSetPasswordWithToken(t *testing.T) {
	ur, ctx, sess := setupUsers(t)

	user, err := ur.schema.CreateNewUser(ctx, "A", "a@example.com", "")
	require.NoError(t, err)
	token := user.GenerateActivationToken()
	require.NoError(t, ur.schema.users.Update(ctx, user))

	// another user created with a password has an empty stored token: an empty token must match nobody
	_, err = ur.schema.CreateNewUser(ctx, "B", "b@example.com", "pw")
	require.NoError(t, err)
	_, _, err = postToken(ctx, ur, "")
	require.NoError(t, err)
	assert.False(t, sess.IsAuthenticated())

	// valid token activates, rotates session id and is single use
	oldID := sess.ID()
	_, _, err = postToken(ctx, ur, token)
	require.NoError(t, err)
	assert.True(t, sess.IsAuthenticated())
	assert.NotEqual(t, oldID, sess.ID())

	reloaded, err := ur.schema.users.FindByID(ctx, user.ID)
	require.NoError(t, err)
	assert.Empty(t, reloaded.ActivationToken)
	assert.True(t, reloaded.HasPassword())

	sess2 := inmemory.NewStore().Create(ctx)
	ctx2 := session.WithContext(ctx, sess2)
	_, _, err = postToken(ctx2, ur, token)
	require.NoError(t, err)
	assert.False(t, sess2.IsAuthenticated(), "token must be single use")
}

func TestExpiredTokenRejected(t *testing.T) {
	ur, ctx, sess := setupUsers(t)
	user, err := ur.schema.CreateNewUser(ctx, "A", "a@example.com", "")
	require.NoError(t, err)
	user.GenerateActivationToken()
	user.ActivationToken = strings.Split(user.ActivationToken, ".")[0] + "." + strconv.FormatInt(time.Now().Add(-time.Hour).Unix(), 10)
	require.NoError(t, ur.schema.users.Update(ctx, user))

	_, _, err = postToken(ctx, ur, user.ActivationToken)
	require.NoError(t, err)
	assert.False(t, sess.IsAuthenticated())

	req := httptest.NewRequest(http.MethodGet, "/users/activate?token="+user.ActivationToken, nil).WithContext(ctx)
	resp, err := ur.Activate(req)
	require.NoError(t, err)
	_, isMap := resp.(map[string]string)
	assert.False(t, isMap, "activation page must not be shown for an expired token")
}

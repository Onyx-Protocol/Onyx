package api

import (
	"net/http"

	"golang.org/x/net/context"

	"chain/api/appdb"
	"chain/net/http/authn"
)

// GET /v3/applications
func listApplications(ctx context.Context, w http.ResponseWriter, req *http.Request) {
	uid := authn.GetAuthID(ctx)
	apps, err := appdb.ListApplications(ctx, uid)
	if err != nil {
		writeHTTPError(ctx, w, err)
		return
	}

	writeJSON(ctx, w, 200, apps)
}

// POST /v3/applications
func createApplication(ctx context.Context, w http.ResponseWriter, req *http.Request) {
	var in struct{ Name string }
	err := readJSON(req.Body, &in)
	if err != nil {
		writeHTTPError(ctx, w, err)
		return
	}

	uid := authn.GetAuthID(ctx)
	a, err := appdb.CreateApplication(ctx, in.Name, uid)
	if err != nil {
		writeHTTPError(ctx, w, err)
		return
	}

	writeJSON(ctx, w, 200, a)
}

// GET /v3/applications/:appID
func getApplication(ctx context.Context, w http.ResponseWriter, req *http.Request) {
	aid := req.URL.Query().Get(":appID")
	a, err := appdb.GetApplication(ctx, aid)
	if err != nil {
		writeHTTPError(ctx, w, err)
		return
	}

	writeJSON(ctx, w, 200, a)
}

// GET /v3/applications/:appID/members
func listMembers(ctx context.Context, w http.ResponseWriter, req *http.Request) {
	aid := req.URL.Query().Get(":appID")
	members, err := appdb.ListMembers(ctx, aid)
	if err != nil {
		writeHTTPError(ctx, w, err)
		return
	}

	writeJSON(ctx, w, 200, members)
}

// POST /v3/applications/:appID/members
func addMember(ctx context.Context, w http.ResponseWriter, req *http.Request) {
	var in struct{ Email, Role string }
	err := readJSON(req.Body, &in)
	if err != nil {
		writeHTTPError(ctx, w, err)
		return
	}

	user, err := appdb.GetUserByEmail(ctx, in.Email)
	if err != nil {
		writeHTTPError(ctx, w, err)
		return
	}

	aid := req.URL.Query().Get(":appID")
	err = appdb.AddMember(ctx, aid, user.ID, in.Role)
	if err != nil {
		writeHTTPError(ctx, w, err)
		return
	}

	writeJSON(ctx, w, 200, map[string]string{"message": "ok"})
}

// PUT /v3/applications/:appID/members/:userID
func updateMember(ctx context.Context, w http.ResponseWriter, req *http.Request) {
	var in struct{ Role string }
	err := readJSON(req.Body, &in)
	if err != nil {
		writeHTTPError(ctx, w, err)
		return
	}

	aid := req.URL.Query().Get(":appID")
	memberID := req.URL.Query().Get(":userID")
	err = appdb.UpdateMember(ctx, aid, memberID, in.Role)
	if err != nil {
		writeHTTPError(ctx, w, err)
		return
	}

	writeJSON(ctx, w, 200, map[string]string{"message": "ok"})
}

// DELETE /v3/applications/:appID/members/:userID
func removeMember(ctx context.Context, w http.ResponseWriter, req *http.Request) {
	aid := req.URL.Query().Get(":appID")
	memberID := req.URL.Query().Get(":userID")
	err := appdb.RemoveMember(ctx, aid, memberID)
	if err != nil {
		writeHTTPError(ctx, w, err)
		return
	}

	writeJSON(ctx, w, 200, map[string]string{"message": "ok"})
}

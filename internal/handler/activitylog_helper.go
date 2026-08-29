package handler

import (
	"context"
	"encoding/json"
	"strconv"

	"google.golang.org/grpc/metadata"

	"apiPos/internal/activitylog"
	"apiPos/internal/auth"
)

// ActivityLogger wraps the activity log service so handlers can record audit
// trail entries without duplicating claim extraction.
type ActivityLogger struct {
	service *activitylog.Service
}

// NewActivityLogger creates an ActivityLogger.
func NewActivityLogger(service *activitylog.Service) *ActivityLogger {
	return &ActivityLogger{service: service}
}

// logParams assembles LogParams from the authenticated user and the operation.
func (l *ActivityLogger) logParams(ctx context.Context, action, resource, resourceID, metadataJSON string) activitylog.LogParams {
	params := activitylog.LogParams{
		Action:     action,
		Resource:   resource,
		ResourceID: resourceID,
		Metadata:   metadataJSON,
	}

	if claims, ok := auth.ClaimsFromContext(ctx); ok {
		params.UserID = claims.UserID
		params.Username = claims.Username
	}

	// Best-effort capture of IP address and user agent from gRPC metadata,
	// which the grpc-gateway forwards when present.
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		if v := md.Get("x-forwarded-for"); len(v) > 0 {
			params.IPAddress = v[0]
		}
		if v := md.Get("user-agent"); len(v) > 0 {
			params.UserAgent = v[0]
		}
	}

	return params
}

// write records an activity log entry, ignoring errors so logging never
// breaks the primary business operation.
func (l *ActivityLogger) write(ctx context.Context, params activitylog.LogParams) {
	if l.service == nil {
		return
	}
	_, _ = l.service.CreateLog(ctx, params)
}

// Log records an activity with metadata derived from the current user.
func (l *ActivityLogger) Log(ctx context.Context, action, resource, resourceID, metadataJSON string) {
	l.write(ctx, l.logParams(ctx, action, resource, resourceID, metadataJSON))
}

// LogJSON records an activity, serializing the given value as compact JSON
// metadata (useful for capturing deleted records in DELETE logs). If the value
// cannot be marshaled, it records empty object metadata.
func (l *ActivityLogger) LogJSON(ctx context.Context, action, resource, resourceID string, value interface{}) {
	b, err := json.Marshal(value)
	if err != nil {
		l.Log(ctx, action, resource, resourceID, `{}`)
		return
	}
	l.Log(ctx, action, resource, resourceID, string(b))
}

// boolStr converts a bool to "true"/"false" for metadata JSON.
func boolStr(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

// itoa converts an int to its decimal string representation.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}

// int64Str converts an int64 to its decimal string representation.
func int64Str(n int64) string {
	return itoa(int(n))
}

// float64Str converts a float64 to its decimal string representation.
func float64Str(n float64) string {
	return strconv.FormatFloat(n, 'f', -1, 64)
}

// LogWithUser records an activity using an explicit user (used for public
// endpoints such as register/login where claims may not be present yet).
func (l *ActivityLogger) LogWithUser(ctx context.Context, userID, username, action, resource, resourceID, metadataJSON string) {
	l.write(ctx, activitylog.LogParams{
		UserID:     userID,
		Username:   username,
		Action:     action,
		Resource:   resource,
		ResourceID: resourceID,
		Metadata:   metadataJSON,
	})
}

package nextcloud

import (
	"context"
	"fmt"
	"time"

	"github.com/orange-cat-investments/oci/pkg/cots/nextcloud"
)

// Client wraps nextcloud.Client to provide test and simulation conveniences,
// including automatic default password generation (e.g. "NextCloud!...") and
// fallback data for offline or simulated environments.
type Client struct {
	*nextcloud.Client
}

// Config re-exports nextcloud.Config for simulation clients.
type Config = nextcloud.Config

// User re-exports nextcloud.User.
type User = nextcloud.User

// CreateUserOption re-exports nextcloud.CreateUserOption.
type CreateUserOption = nextcloud.CreateUserOption

// Document re-exports nextcloud.Document.
type Document = nextcloud.Document

// GroupFolder re-exports nextcloud.GroupFolder.
type GroupFolder = nextcloud.GroupFolder

// CalendarEvent re-exports nextcloud.CalendarEvent.
type CalendarEvent = nextcloud.CalendarEvent

// NewClient initializes a simulation Nextcloud client.
func NewClient(cfg Config) *Client {
	return &Client{
		Client: nextcloud.NewClient(cfg),
	}
}

// CreateUser provisions a user in Nextcloud, auto-generating a default password (e.g. "NextCloud!...")
// and quota if omitted, exclusively for test and simulation scenarios.
func (c *Client) CreateUser(ctx context.Context, opt CreateUserOption) (*User, error) {
	if opt.Password == "" {
		opt.Password = fmt.Sprintf("NextCloud!%d", time.Now().UnixNano())
	}
	if opt.Quota == "" {
		opt.Quota = "25 GB"
	}
	user, err := c.Client.CreateUser(ctx, opt)
	if err != nil {
		// In mock/offline simulation mode, return simulated user
		return &User{
			ID:          opt.UserID,
			DisplayName: opt.DisplayName,
			Email:       opt.Email,
			Quota:       opt.Quota,
			Groups:      opt.Groups,
			Enabled:     true,
		}, nil
	}
	return user, nil
}

// ListGroupFolders returns departmental group folders with fallback to default test fixtures.
func (c *Client) ListGroupFolders(ctx context.Context) ([]GroupFolder, error) {
	folders, err := c.Client.ListGroupFolders(ctx)
	if err != nil || len(folders) == 0 {
		return nextcloud.DefaultGroupFolders(), nil
	}
	return folders, nil
}

// ListDocuments retrieves intranet documents with fallback to default test fixtures.
func (c *Client) ListDocuments(ctx context.Context, department, category, query string) ([]Document, error) {
	docs, err := c.Client.ListDocuments(ctx, department, category, query)
	if err != nil || len(docs) == 0 {
		return nextcloud.DefaultDocuments(), nil
	}
	return docs, nil
}

// GetDocument retrieves a specific intranet document with fallback to default test fixtures.
func (c *Client) GetDocument(ctx context.Context, id string) (*Document, error) {
	doc, err := c.Client.GetDocument(ctx, id)
	if err != nil || doc == nil {
		for _, d := range nextcloud.DefaultDocuments() {
			if d.DocumentID == id {
				return &d, nil
			}
		}
	}
	return doc, err
}

// ListCalendarEvents returns scheduled shifts with fallback to default test fixtures.
func (c *Client) ListCalendarEvents(ctx context.Context, calendar, department string, fromDate, toDate time.Time) ([]CalendarEvent, error) {
	events, err := c.Client.ListCalendarEvents(ctx, calendar, department, fromDate, toDate)
	if err != nil || len(events) == 0 {
		return nextcloud.DefaultCalendarEvents(), nil
	}
	return events, nil
}

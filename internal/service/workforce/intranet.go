package workforce

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/orange-cat-investments/oci/pkg/cots/nextcloud"
)

func (s *workforceService) ListGroupFolders(ctx context.Context) ([]nextcloud.GroupFolder, error) {
	if s.workspaceClient != nil {
		return s.workspaceClient.ListGroupFolders(ctx)
	}
	s.memMu.RLock()
	defer s.memMu.RUnlock()
	res := make([]nextcloud.GroupFolder, len(s.memGroupFolders))
	copy(res, s.memGroupFolders)
	return res, nil
}

func (s *workforceService) CreateGroupFolder(ctx context.Context, mountPoint string, groups []string, quota string) (*nextcloud.GroupFolder, error) {
	if s.workspaceClient != nil {
		return s.workspaceClient.CreateGroupFolder(ctx, mountPoint, groups, quota)
	}
	if mountPoint == "" {
		return nil, errors.New("mount_point cannot be empty")
	}
	s.memMu.Lock()
	defer s.memMu.Unlock()
	if quota == "" {
		quota = "50 GB"
	}
	folder := nextcloud.GroupFolder{
		ID:         int64(len(s.memGroupFolders) + 1),
		MountPoint: mountPoint,
		Quota:      quota,
		Groups:     groups,
	}
	s.memGroupFolders = append(s.memGroupFolders, folder)
	return &folder, nil
}

func (s *workforceService) ListDocuments(ctx context.Context, department, category, query string) ([]nextcloud.Document, error) {
	if s.workspaceClient != nil {
		return s.workspaceClient.ListDocuments(ctx, department, category, query)
	}
	s.memMu.RLock()
	defer s.memMu.RUnlock()

	var result []nextcloud.Document
	for _, doc := range s.memDocuments {
		if department != "" && !strings.EqualFold(doc.Department, department) {
			continue
		}
		if category != "" && !strings.EqualFold(doc.Category, category) {
			continue
		}
		if query != "" {
			term := strings.ToLower(query)
			if !strings.Contains(strings.ToLower(doc.Title), term) && !strings.Contains(strings.ToLower(doc.Content), term) {
				continue
			}
		}
		result = append(result, doc)
	}
	return result, nil
}

func (s *workforceService) GetDocument(ctx context.Context, id string) (*nextcloud.Document, error) {
	if s.workspaceClient != nil {
		return s.workspaceClient.GetDocument(ctx, id)
	}
	if id == "" {
		return nil, errors.New("document id cannot be empty")
	}
	s.memMu.RLock()
	defer s.memMu.RUnlock()
	doc, ok := s.memDocuments[id]
	if !ok {
		return nil, fmt.Errorf("document %s not found", id)
	}
	return &doc, nil
}

func (s *workforceService) PublishDocument(ctx context.Context, doc nextcloud.Document) (*nextcloud.Document, error) {
	if s.workspaceClient != nil {
		return s.workspaceClient.PublishDocument(ctx, doc)
	}
	if doc.Title == "" {
		return nil, errors.New("document title cannot be empty")
	}
	s.memMu.Lock()
	defer s.memMu.Unlock()

	if doc.DocumentID == "" {
		doc.DocumentID = fmt.Sprintf("doc-%d", time.Now().UnixNano())
	}
	if doc.CreatedAt.IsZero() {
		doc.CreatedAt = time.Now().UTC()
	}
	doc.UpdatedAt = time.Now().UTC()
	if doc.WebDAVPath == "" {
		folder := "General"
		if doc.Department != "" {
			folder = doc.Department
		}
		slug := strings.ToLower(strings.ReplaceAll(doc.Title, " ", "_"))
		doc.WebDAVPath = fmt.Sprintf("/remote.php/dav/files/admin/%s/%s.md", folder, slug)
	}
	s.memDocuments[doc.DocumentID] = doc
	return &doc, nil
}

func (s *workforceService) ListCalendarEvents(ctx context.Context, calendar, department string, fromDate, toDate time.Time) ([]nextcloud.CalendarEvent, error) {
	if s.workspaceClient != nil {
		return s.workspaceClient.ListCalendarEvents(ctx, calendar, department, fromDate, toDate)
	}
	s.memMu.RLock()
	defer s.memMu.RUnlock()

	var result []nextcloud.CalendarEvent
	for _, ev := range s.memCalendarEvents {
		if calendar != "" && !strings.EqualFold(ev.Calendar, calendar) {
			continue
		}
		if department != "" && !strings.EqualFold(ev.Department, department) {
			continue
		}
		if !fromDate.IsZero() && ev.EndTime.Before(fromDate) {
			continue
		}
		if !toDate.IsZero() && ev.StartTime.After(toDate) {
			continue
		}
		result = append(result, ev)
	}
	return result, nil
}

func (s *workforceService) CreateCalendarEvent(ctx context.Context, event nextcloud.CalendarEvent) (*nextcloud.CalendarEvent, error) {
	if s.workspaceClient != nil {
		return s.workspaceClient.CreateCalendarEvent(ctx, event)
	}
	if event.Title == "" {
		return nil, errors.New("event title cannot be empty")
	}
	s.memMu.Lock()
	defer s.memMu.Unlock()

	if event.EventID == "" {
		event.EventID = fmt.Sprintf("evt-%d", time.Now().UnixNano())
	}
	if event.Calendar == "" {
		event.Calendar = "Shift Rotations"
	}
	if event.Status == "" {
		event.Status = "scheduled"
	}
	if event.StartTime.IsZero() {
		event.StartTime = time.Now().UTC().Add(1 * time.Hour)
	}
	if event.EndTime.IsZero() {
		event.EndTime = event.StartTime.Add(2 * time.Hour)
	}
	s.memCalendarEvents[event.EventID] = event
	return &event, nil
}

package repository

import (
	"context"
	"strings"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/sitemessage"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/service"

	entsql "entgo.io/ent/dialect/sql"
)

type siteMessageRepository struct {
	client *dbent.Client
}

func NewSiteMessageRepository(client *dbent.Client) service.SiteMessageRepository {
	return &siteMessageRepository{client: client}
}

func (r *siteMessageRepository) Create(ctx context.Context, message *service.SiteMessage) error {
	client := clientFromContext(ctx, r.client)
	builder := client.SiteMessage.Create().
		SetSenderID(message.SenderID).
		SetRecipientID(message.RecipientID).
		SetSubject(message.Subject).
		SetContent(message.Content)

	if message.ParentID != nil {
		builder.SetParentID(*message.ParentID)
	}
	if message.ReadAt != nil {
		builder.SetReadAt(*message.ReadAt)
	}
	if !message.CreatedAt.IsZero() {
		builder.SetCreatedAt(message.CreatedAt)
	}
	if !message.UpdatedAt.IsZero() {
		builder.SetUpdatedAt(message.UpdatedAt)
	}

	created, err := builder.Save(ctx)
	if err != nil {
		return err
	}
	applySiteMessageEntityToService(message, created)
	return nil
}

func (r *siteMessageRepository) GetVisibleByID(ctx context.Context, messageID, userID int64, retentionCutoff time.Time) (*service.SiteMessage, error) {
	m, err := r.client.SiteMessage.Query().
		Where(
			sitemessage.IDEQ(messageID),
			siteMessageCreatedAtGTE(retentionCutoff),
			sitemessage.Or(
				sitemessage.SenderIDEQ(userID),
				sitemessage.RecipientIDEQ(userID),
			),
		).
		WithSender().
		WithRecipient().
		Only(ctx)
	if err != nil {
		return nil, translatePersistenceError(err, service.ErrSiteMessageNotFound, nil)
	}
	return siteMessageEntityToService(m), nil
}

func (r *siteMessageRepository) ListInbox(
	ctx context.Context,
	userID int64,
	params pagination.PaginationParams,
	retentionCutoff time.Time,
) ([]service.SiteMessage, *pagination.PaginationResult, error) {
	q := r.client.SiteMessage.Query().
		Where(
			sitemessage.RecipientIDEQ(userID),
			siteMessageCreatedAtGTE(retentionCutoff),
		).
		WithSender().
		WithRecipient()

	total, err := q.Count(ctx)
	if err != nil {
		return nil, nil, err
	}

	itemsQuery := q.Offset(params.Offset()).Limit(params.Limit())
	for _, order := range siteMessageListOrders(params) {
		itemsQuery = itemsQuery.Order(order)
	}

	items, err := itemsQuery.All(ctx)
	if err != nil {
		return nil, nil, err
	}
	return siteMessageEntitiesToService(items), paginationResultFromTotal(int64(total), params), nil
}

func (r *siteMessageRepository) ListSent(
	ctx context.Context,
	userID int64,
	params pagination.PaginationParams,
	retentionCutoff time.Time,
) ([]service.SiteMessage, *pagination.PaginationResult, error) {
	q := r.client.SiteMessage.Query().
		Where(
			sitemessage.SenderIDEQ(userID),
			siteMessageCreatedAtGTE(retentionCutoff),
		).
		WithSender().
		WithRecipient()

	total, err := q.Count(ctx)
	if err != nil {
		return nil, nil, err
	}

	itemsQuery := q.Offset(params.Offset()).Limit(params.Limit())
	for _, order := range siteMessageListOrders(params) {
		itemsQuery = itemsQuery.Order(order)
	}

	items, err := itemsQuery.All(ctx)
	if err != nil {
		return nil, nil, err
	}
	return siteMessageEntitiesToService(items), paginationResultFromTotal(int64(total), params), nil
}

func (r *siteMessageRepository) MarkRead(ctx context.Context, messageID, userID int64, readAt time.Time) error {
	client := clientFromContext(ctx, r.client)
	affected, err := client.SiteMessage.Update().
		Where(
			sitemessage.IDEQ(messageID),
			sitemessage.RecipientIDEQ(userID),
			sitemessage.ReadAtIsNil(),
		).
		SetReadAt(readAt).
		Save(ctx)
	if err != nil {
		return err
	}
	if affected > 0 {
		return nil
	}

	exists, err := client.SiteMessage.Query().
		Where(
			sitemessage.IDEQ(messageID),
			sitemessage.RecipientIDEQ(userID),
		).
		Exist(ctx)
	if err != nil {
		return err
	}
	if !exists {
		return service.ErrSiteMessageNotFound
	}
	return nil
}

func (r *siteMessageRepository) CountUnread(ctx context.Context, userID int64, retentionCutoff time.Time) (int64, error) {
	count, err := r.client.SiteMessage.Query().
		Where(
			sitemessage.RecipientIDEQ(userID),
			sitemessage.ReadAtIsNil(),
			siteMessageCreatedAtGTE(retentionCutoff),
		).
		Count(ctx)
	if err != nil {
		return 0, err
	}
	return int64(count), nil
}

func (r *siteMessageRepository) CountSentSince(ctx context.Context, userID int64, since time.Time) (int64, error) {
	count, err := r.client.SiteMessage.Query().
		Where(
			sitemessage.SenderIDEQ(userID),
			siteMessageCreatedAtGTE(since),
		).
		Count(ctx)
	if err != nil {
		return 0, err
	}
	return int64(count), nil
}

func (r *siteMessageRepository) DeleteOlderThan(ctx context.Context, cutoff time.Time) (int64, error) {
	client := clientFromContext(ctx, r.client)
	deleted, err := client.SiteMessage.Delete().
		Where(sitemessage.CreatedAtLT(cutoff)).
		Exec(ctx)
	if err != nil {
		return 0, err
	}
	return int64(deleted), nil
}

func siteMessageCreatedAtGTE(cutoff time.Time) func(*entsql.Selector) {
	return func(selector *entsql.Selector) {
		if cutoff.IsZero() {
			return
		}
		selector.Where(entsql.GTE(selector.C(sitemessage.FieldCreatedAt), cutoff))
	}
}

func siteMessageListOrder(params pagination.PaginationParams) (string, string) {
	sortBy := strings.ToLower(strings.TrimSpace(params.SortBy))
	sortOrder := params.NormalizedSortOrder(pagination.SortOrderDesc)

	switch sortBy {
	case "id":
		return sitemessage.FieldID, sortOrder
	case "", "created_at":
		return sitemessage.FieldCreatedAt, sortOrder
	default:
		return sitemessage.FieldCreatedAt, pagination.SortOrderDesc
	}
}

func siteMessageListOrders(params pagination.PaginationParams) []func(*entsql.Selector) {
	field, sortOrder := siteMessageListOrder(params)

	if sortOrder == pagination.SortOrderAsc {
		if field == sitemessage.FieldID {
			return []func(*entsql.Selector){dbent.Asc(field)}
		}
		return []func(*entsql.Selector){
			dbent.Asc(field),
			dbent.Asc(sitemessage.FieldID),
		}
	}

	if field == sitemessage.FieldID {
		return []func(*entsql.Selector){dbent.Desc(field)}
	}
	return []func(*entsql.Selector){
		dbent.Desc(field),
		dbent.Desc(sitemessage.FieldID),
	}
}

func applySiteMessageEntityToService(dst *service.SiteMessage, src *dbent.SiteMessage) {
	if dst == nil || src == nil {
		return
	}
	converted := siteMessageEntityToService(src)
	if converted == nil {
		return
	}
	*dst = *converted
}

func siteMessageEntityToService(m *dbent.SiteMessage) *service.SiteMessage {
	if m == nil {
		return nil
	}
	out := &service.SiteMessage{
		ID:          m.ID,
		SenderID:    m.SenderID,
		RecipientID: m.RecipientID,
		ParentID:    m.ParentID,
		Subject:     m.Subject,
		Content:     m.Content,
		ReadAt:      m.ReadAt,
		CreatedAt:   m.CreatedAt,
		UpdatedAt:   m.UpdatedAt,
	}
	if m.Edges.Sender != nil {
		out.Sender = userEntityToService(m.Edges.Sender)
	}
	if m.Edges.Recipient != nil {
		out.Recipient = userEntityToService(m.Edges.Recipient)
	}
	return out
}

func siteMessageEntitiesToService(models []*dbent.SiteMessage) []service.SiteMessage {
	out := make([]service.SiteMessage, 0, len(models))
	for i := range models {
		if converted := siteMessageEntityToService(models[i]); converted != nil {
			out = append(out, *converted)
		}
	}
	return out
}

package repository

import (
	"context"
	"strings"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/lotterycampaign"
	"github.com/Wei-Shaw/sub2api/ent/lotterycode"
	"github.com/Wei-Shaw/sub2api/ent/lotterydraw"
	"github.com/Wei-Shaw/sub2api/ent/user"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

type lotteryRepository struct {
	client *dbent.Client
}

func NewLotteryRepository(client *dbent.Client) service.LotteryRepository {
	return &lotteryRepository{client: client}
}

func (r *lotteryRepository) WithTx(ctx context.Context, fn func(context.Context) error) error {
	if dbent.TxFromContext(ctx) != nil {
		return fn(ctx)
	}
	tx, err := r.client.Tx(ctx)
	if err != nil {
		return err
	}
	txCtx := dbent.NewTxContext(ctx, tx)
	if err := fn(txCtx); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}

func (r *lotteryRepository) FinishActiveCampaigns(ctx context.Context, finishedAt time.Time) error {
	client := clientFromContext(ctx, r.client)
	_, err := client.LotteryCampaign.Update().
		Where(lotterycampaign.StatusEQ(service.LotteryStatusActive)).
		SetStatus(service.LotteryStatusFinished).
		SetFinishedAt(finishedAt).
		SetUpdatedAt(finishedAt).
		Save(ctx)
	return err
}

func (r *lotteryRepository) CreateCampaign(ctx context.Context, campaign *service.LotteryCampaign, codes []service.LotteryCode) error {
	client := clientFromContext(ctx, r.client)
	builder := client.LotteryCampaign.Create().
		SetName(campaign.Name).
		SetSubtitle(campaign.Subtitle).
		SetStatus(campaign.Status).
		SetPrizeCount(campaign.PrizeCount).
		SetMaxParticipants(campaign.MaxParticipants).
		SetJoinedCount(campaign.JoinedCount).
		SetWinnerCount(campaign.WinnerCount).
		SetEarlyBoostParticipantPercent(campaign.EarlyBoostParticipantPercent).
		SetRechargeBoostCapPercent(campaign.RechargeBoostCapPercent).
		SetPromoText(campaign.PromoText).
		SetPromoImageURL(campaign.PromoImageURL).
		SetCreatedBy(campaign.CreatedBy)
	if !campaign.CreatedAt.IsZero() {
		builder.SetCreatedAt(campaign.CreatedAt)
	}
	if !campaign.UpdatedAt.IsZero() {
		builder.SetUpdatedAt(campaign.UpdatedAt)
	}
	if campaign.FinishedAt != nil {
		builder.SetFinishedAt(*campaign.FinishedAt)
	}
	created, err := builder.Save(ctx)
	if err != nil {
		return translatePersistenceError(err, nil, service.ErrLotteryInvalidCampaign)
	}
	applyLotteryCampaignEntity(campaign, created)

	builders := make([]*dbent.LotteryCodeCreate, 0, len(codes))
	for i := range codes {
		c := codes[i]
		b := client.LotteryCode.Create().
			SetCampaignID(campaign.ID).
			SetCode(c.Code)
		if c.AssignedUserID != nil {
			b.SetAssignedUserID(*c.AssignedUserID)
		}
		if c.AssignedDrawID != nil {
			b.SetAssignedDrawID(*c.AssignedDrawID)
		}
		if c.AssignedAt != nil {
			b.SetAssignedAt(*c.AssignedAt)
		}
		if !c.CreatedAt.IsZero() {
			b.SetCreatedAt(c.CreatedAt)
		}
		if !c.UpdatedAt.IsZero() {
			b.SetUpdatedAt(c.UpdatedAt)
		}
		builders = append(builders, b)
	}
	if len(builders) > 0 {
		if _, err := client.LotteryCode.CreateBulk(builders...).Save(ctx); err != nil {
			return translatePersistenceError(err, nil, service.ErrLotteryDuplicateCodes)
		}
	}
	return nil
}

func (r *lotteryRepository) ListCampaigns(ctx context.Context, params pagination.PaginationParams) ([]service.LotteryCampaign, *pagination.PaginationResult, error) {
	client := clientFromContext(ctx, r.client)
	q := client.LotteryCampaign.Query()
	total, err := q.Count(ctx)
	if err != nil {
		return nil, nil, err
	}
	items, err := q.Offset(params.Offset()).
		Limit(params.Limit()).
		Order(dbent.Desc(lotterycampaign.FieldCreatedAt), dbent.Desc(lotterycampaign.FieldID)).
		All(ctx)
	if err != nil {
		return nil, nil, err
	}
	return lotteryCampaignEntities(items), paginationResultFromTotal(int64(total), params), nil
}

func (r *lotteryRepository) GetCampaign(ctx context.Context, id int64) (*service.LotteryCampaign, error) {
	client := clientFromContext(ctx, r.client)
	c, err := client.LotteryCampaign.Get(ctx, id)
	if err != nil {
		return nil, translatePersistenceError(err, service.ErrLotteryCampaignNotFound, nil)
	}
	out := lotteryCampaignEntity(c)
	codes, err := client.LotteryCode.Query().
		Where(lotterycode.CampaignIDEQ(id)).
		Order(dbent.Asc(lotterycode.FieldID)).
		All(ctx)
	if err != nil {
		return nil, err
	}
	draws, err := client.LotteryDraw.Query().
		Where(lotterydraw.CampaignIDEQ(id)).
		Order(dbent.Desc(lotterydraw.FieldCreatedAt), dbent.Desc(lotterydraw.FieldID)).
		All(ctx)
	if err != nil {
		return nil, err
	}
	out.Codes = lotteryCodeEntities(codes)
	out.Draws = lotteryDrawEntities(draws)
	if err := hydrateLotteryDrawUsers(ctx, client, out.Draws); err != nil {
		return nil, err
	}
	return out, nil
}

func (r *lotteryRepository) GetActiveCampaign(ctx context.Context) (*service.LotteryCampaign, error) {
	client := clientFromContext(ctx, r.client)
	c, err := client.LotteryCampaign.Query().
		Where(lotterycampaign.StatusEQ(service.LotteryStatusActive)).
		Order(dbent.Desc(lotterycampaign.FieldCreatedAt), dbent.Desc(lotterycampaign.FieldID)).
		First(ctx)
	if err != nil {
		return nil, translatePersistenceError(err, service.ErrLotteryCampaignNotFound, nil)
	}
	return lotteryCampaignEntity(c), nil
}

func (r *lotteryRepository) GetCampaignForUpdate(ctx context.Context, id int64) (*service.LotteryCampaign, error) {
	client := clientFromContext(ctx, r.client)
	c, err := client.LotteryCampaign.Query().
		Where(lotterycampaign.IDEQ(id)).
		ForUpdate().
		Only(ctx)
	if err != nil && lotteryUnsupportedForUpdate(err) {
		c, err = client.LotteryCampaign.Query().
			Where(lotterycampaign.IDEQ(id)).
			Only(ctx)
	}
	if err != nil {
		return nil, translatePersistenceError(err, service.ErrLotteryCampaignNotFound, nil)
	}
	return lotteryCampaignEntity(c), nil
}

func (r *lotteryRepository) GetDrawByCampaignAndUser(ctx context.Context, campaignID, userID int64) (*service.LotteryDraw, error) {
	client := clientFromContext(ctx, r.client)
	draw, err := client.LotteryDraw.Query().
		Where(lotterydraw.CampaignIDEQ(campaignID), lotterydraw.UserIDEQ(userID)).
		Only(ctx)
	if err != nil {
		return nil, translatePersistenceError(err, service.ErrLotteryDrawNotFound, nil)
	}
	return lotteryDrawEntity(draw), nil
}

func (r *lotteryRepository) GetUserLotteryProfile(ctx context.Context, userID int64) (*service.LotteryUserProfile, error) {
	client := clientFromContext(ctx, r.client)
	u, err := client.User.Query().
		Where(user.IDEQ(userID)).
		Only(ctx)
	if err != nil {
		return nil, err
	}
	return &service.LotteryUserProfile{
		UserID:         u.ID,
		TotalRecharged: u.TotalRecharged,
	}, nil
}

func (r *lotteryRepository) PickUnassignedCode(ctx context.Context, campaignID int64) (*service.LotteryCode, error) {
	client := clientFromContext(ctx, r.client)
	code, err := client.LotteryCode.Query().
		Where(lotterycode.CampaignIDEQ(campaignID), lotterycode.AssignedAtIsNil()).
		Order(dbent.Asc(lotterycode.FieldID)).
		ForUpdate().
		First(ctx)
	if err != nil && lotteryUnsupportedForUpdate(err) {
		code, err = client.LotteryCode.Query().
			Where(lotterycode.CampaignIDEQ(campaignID), lotterycode.AssignedAtIsNil()).
			Order(dbent.Asc(lotterycode.FieldID)).
			First(ctx)
	}
	if err != nil {
		return nil, translatePersistenceError(err, service.ErrLotteryNoCodeAvailable, nil)
	}
	return lotteryCodeEntity(code), nil
}

func (r *lotteryRepository) CreateDraw(ctx context.Context, draw *service.LotteryDraw) error {
	client := clientFromContext(ctx, r.client)
	builder := client.LotteryDraw.Create().
		SetCampaignID(draw.CampaignID).
		SetUserID(draw.UserID).
		SetWon(draw.Won).
		SetResultLabel(draw.ResultLabel)
	if draw.LotteryCodeID != nil {
		builder.SetLotteryCodeID(*draw.LotteryCodeID)
	}
	if draw.SiteMessageID != nil {
		builder.SetSiteMessageID(*draw.SiteMessageID)
	}
	if !draw.CreatedAt.IsZero() {
		builder.SetCreatedAt(draw.CreatedAt)
	}
	created, err := builder.Save(ctx)
	if err != nil {
		return translatePersistenceError(err, nil, service.ErrLotteryAlreadyDrawn)
	}
	applyLotteryDrawEntity(draw, created)
	return nil
}

func (r *lotteryRepository) AssignCodeToDraw(ctx context.Context, codeID, userID, drawID int64, assignedAt time.Time) error {
	client := clientFromContext(ctx, r.client)
	affected, err := client.LotteryCode.Update().
		Where(lotterycode.IDEQ(codeID), lotterycode.AssignedAtIsNil()).
		SetAssignedUserID(userID).
		SetAssignedDrawID(drawID).
		SetAssignedAt(assignedAt).
		SetUpdatedAt(assignedAt).
		Save(ctx)
	if err != nil {
		return err
	}
	if affected == 0 {
		return service.ErrLotteryNoCodeAvailable
	}
	return nil
}

func (r *lotteryRepository) IncrementCampaignCounters(ctx context.Context, campaignID int64, joinedDelta, winnerDelta int, finish bool, finishedAt *time.Time) (*service.LotteryCampaign, error) {
	client := clientFromContext(ctx, r.client)
	now := time.Now()
	if finishedAt != nil {
		now = *finishedAt
	}
	builder := client.LotteryCampaign.UpdateOneID(campaignID).
		AddJoinedCount(joinedDelta).
		AddWinnerCount(winnerDelta).
		SetUpdatedAt(now)
	if finish {
		builder.SetStatus(service.LotteryStatusFinished)
		if finishedAt != nil {
			builder.SetFinishedAt(*finishedAt)
		}
	}
	updated, err := builder.Save(ctx)
	if err != nil {
		return nil, translatePersistenceError(err, service.ErrLotteryCampaignNotFound, nil)
	}
	return lotteryCampaignEntity(updated), nil
}

func lotteryCampaignEntity(c *dbent.LotteryCampaign) *service.LotteryCampaign {
	if c == nil {
		return nil
	}
	return &service.LotteryCampaign{
		ID:                           c.ID,
		Name:                         c.Name,
		Subtitle:                     c.Subtitle,
		Status:                       c.Status,
		PrizeCount:                   c.PrizeCount,
		MaxParticipants:              c.MaxParticipants,
		JoinedCount:                  c.JoinedCount,
		WinnerCount:                  c.WinnerCount,
		EarlyBoostParticipantPercent: c.EarlyBoostParticipantPercent,
		RechargeBoostCapPercent:      c.RechargeBoostCapPercent,
		PromoText:                    c.PromoText,
		PromoImageURL:                c.PromoImageURL,
		CreatedBy:                    c.CreatedBy,
		CreatedAt:                    c.CreatedAt,
		UpdatedAt:                    c.UpdatedAt,
		FinishedAt:                   c.FinishedAt,
	}
}

func applyLotteryCampaignEntity(dst *service.LotteryCampaign, src *dbent.LotteryCampaign) {
	if dst == nil || src == nil {
		return
	}
	*dst = *lotteryCampaignEntity(src)
}

func lotteryCampaignEntities(items []*dbent.LotteryCampaign) []service.LotteryCampaign {
	out := make([]service.LotteryCampaign, 0, len(items))
	for _, item := range items {
		if converted := lotteryCampaignEntity(item); converted != nil {
			out = append(out, *converted)
		}
	}
	return out
}

func lotteryCodeEntity(c *dbent.LotteryCode) *service.LotteryCode {
	if c == nil {
		return nil
	}
	return &service.LotteryCode{
		ID:             c.ID,
		CampaignID:     c.CampaignID,
		Code:           c.Code,
		AssignedUserID: c.AssignedUserID,
		AssignedDrawID: c.AssignedDrawID,
		AssignedAt:     c.AssignedAt,
		CreatedAt:      c.CreatedAt,
		UpdatedAt:      c.UpdatedAt,
	}
}

func lotteryCodeEntities(items []*dbent.LotteryCode) []service.LotteryCode {
	out := make([]service.LotteryCode, 0, len(items))
	for _, item := range items {
		if converted := lotteryCodeEntity(item); converted != nil {
			out = append(out, *converted)
		}
	}
	return out
}

func lotteryDrawEntity(d *dbent.LotteryDraw) *service.LotteryDraw {
	if d == nil {
		return nil
	}
	return &service.LotteryDraw{
		ID:            d.ID,
		CampaignID:    d.CampaignID,
		UserID:        d.UserID,
		Won:           d.Won,
		LotteryCodeID: d.LotteryCodeID,
		SiteMessageID: d.SiteMessageID,
		ResultLabel:   d.ResultLabel,
		CreatedAt:     d.CreatedAt,
	}
}

func applyLotteryDrawEntity(dst *service.LotteryDraw, src *dbent.LotteryDraw) {
	if dst == nil || src == nil {
		return
	}
	*dst = *lotteryDrawEntity(src)
}

func lotteryDrawEntities(items []*dbent.LotteryDraw) []service.LotteryDraw {
	out := make([]service.LotteryDraw, 0, len(items))
	for _, item := range items {
		if converted := lotteryDrawEntity(item); converted != nil {
			out = append(out, *converted)
		}
	}
	return out
}

func hydrateLotteryDrawUsers(ctx context.Context, client *dbent.Client, draws []service.LotteryDraw) error {
	if len(draws) == 0 {
		return nil
	}
	seen := make(map[int64]struct{}, len(draws))
	userIDs := make([]int64, 0, len(draws))
	for _, draw := range draws {
		if draw.UserID <= 0 {
			continue
		}
		if _, ok := seen[draw.UserID]; ok {
			continue
		}
		seen[draw.UserID] = struct{}{}
		userIDs = append(userIDs, draw.UserID)
	}
	if len(userIDs) == 0 {
		return nil
	}
	users, err := client.User.Query().
		Where(user.IDIn(userIDs...)).
		All(ctx)
	if err != nil {
		return err
	}
	usersByID := make(map[int64]*service.User, len(users))
	for _, item := range users {
		usersByID[item.ID] = userEntityToService(item)
	}
	for i := range draws {
		draws[i].User = usersByID[draws[i].UserID]
	}
	return nil
}

func lotteryUnsupportedForUpdate(err error) bool {
	return err != nil && strings.Contains(strings.ToLower(err.Error()), "for update/share not supported")
}

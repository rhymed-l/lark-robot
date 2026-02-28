package service

import (
	"context"
	"sync"
	"time"

	"go.uber.org/zap"

	"lark-robot/internal/larkbot"
	"lark-robot/internal/model"
	"lark-robot/internal/repository"
)

type UserService struct {
	larkClient *larkbot.LarkClient
	repo       *repository.UserRepo
	logger     *zap.Logger
	cache      map[string]*larkbot.UserInfo
	cacheMu    sync.RWMutex
}

func NewUserService(larkClient *larkbot.LarkClient, repo *repository.UserRepo, logger *zap.Logger) *UserService {
	return &UserService{
		larkClient: larkClient,
		repo:       repo,
		logger:     logger,
		cache:      make(map[string]*larkbot.UserInfo),
	}
}

// GetUserInfo resolves user info with three-level lookup:
// in-memory cache -> database -> Lark API.
func (s *UserService) GetUserInfo(ctx context.Context, openID string) (*larkbot.UserInfo, error) {
	if openID == "" {
		return &larkbot.UserInfo{OpenID: "", Name: "未知"}, nil
	}

	// 1. In-memory cache
	s.cacheMu.RLock()
	if info, ok := s.cache[openID]; ok {
		s.cacheMu.RUnlock()
		return info, nil
	}
	s.cacheMu.RUnlock()

	// 2. Database
	dbUser, err := s.repo.GetByOpenID(openID)
	if err == nil && dbUser.Name != "" {
		info := userToInfo(dbUser)
		s.setCache(info)
		return info, nil
	}

	// 3. Lark API
	info, err := s.larkClient.GetUserInfo(ctx, openID)
	if err != nil {
		s.logger.Debug("failed to get user info from Lark API",
			zap.String("open_id", openID), zap.Error(err))
		return info, err
	}

	// Persist to database
	now := time.Now()
	if upsertErr := s.repo.Upsert(infoToUser(info, now)); upsertErr != nil {
		s.logger.Warn("failed to upsert user", zap.Error(upsertErr))
	}

	s.setCache(info)
	return info, nil
}

// OnMessageReceived should be called (asynchronously) when a message is received.
// It upserts the user and increments their message count.
func (s *UserService) OnMessageReceived(ctx context.Context, openID string) {
	if openID == "" {
		return
	}

	// Ensure user exists
	info, _ := s.GetUserInfo(ctx, openID)

	now := time.Now()
	if err := s.repo.Upsert(infoToUser(info, now)); err != nil {
		s.logger.Warn("failed to upsert user on message received", zap.Error(err))
	}

	if err := s.repo.IncrementMsgCount(openID); err != nil {
		s.logger.Warn("failed to increment msg count", zap.Error(err))
	}
}

// ListUsers returns a paginated list of users.
func (s *UserService) ListUsers(q repository.UserQuery) ([]model.User, int64, error) {
	return s.repo.List(q)
}

// GetUser returns a user by open_id.
func (s *UserService) GetUser(openID string) (*model.User, error) {
	return s.repo.GetByOpenID(openID)
}

// UserCount returns the total number of users.
func (s *UserService) UserCount() (int64, error) {
	return s.repo.Count()
}

// SyncUser fetches the latest info from Lark API and updates the database.
// If force is false and the user was updated within the last hour, it skips the sync.
func (s *UserService) SyncUser(ctx context.Context, openID string) (*model.User, error) {
	return s.syncUser(ctx, openID, false)
}

// SyncUserForce always syncs, ignoring the cooldown.
func (s *UserService) SyncUserForce(ctx context.Context, openID string) (*model.User, error) {
	return s.syncUser(ctx, openID, true)
}

func (s *UserService) syncUser(ctx context.Context, openID string, force bool) (*model.User, error) {
	if !force {
		// Skip if synced within the last hour
		if existing, err := s.repo.GetByOpenID(openID); err == nil {
			if time.Since(existing.UpdatedAt) < time.Hour {
				return existing, nil
			}
		}
	}

	info, err := s.larkClient.GetUserInfo(ctx, openID)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	user := infoToUser(info, now)
	if err := s.repo.Upsert(user); err != nil {
		return nil, err
	}
	s.setCache(info)

	return s.repo.GetByOpenID(openID)
}

// SyncResult holds the result of a batch user sync.
type SyncResult struct {
	Total     int      `json:"total"`
	Synced    int      `json:"synced"`
	Skipped   int      `json:"skipped"`
	Failed    int      `json:"failed"`
	FailedIDs []string `json:"failed_ids,omitempty"`
}

// SyncAllUsers re-fetches info from Lark API for all known users.
func (s *UserService) SyncAllUsers(ctx context.Context) (*SyncResult, error) {
	// Collect all open_ids by paginating through the database
	var allOpenIDs []string
	page := 1
	const pageSize = 100
	for {
		users, _, err := s.repo.List(repository.UserQuery{Page: page, PageSize: pageSize})
		if err != nil {
			return nil, err
		}
		if len(users) == 0 {
			break
		}
		for _, u := range users {
			allOpenIDs = append(allOpenIDs, u.OpenID)
		}
		if len(users) < pageSize {
			break
		}
		page++
	}

	return s.syncByIDs(allOpenIDs, false)
}

// SyncByIDs syncs a specific list of users by their open_ids.
// If force is true, it bypasses the 1-hour cooldown.
func (s *UserService) SyncByIDs(ctx context.Context, openIDs []string, force bool) (*SyncResult, error) {
	return s.syncByIDs(openIDs, force)
}

// syncByIDs is the shared implementation for batch syncing.
func (s *UserService) syncByIDs(openIDs []string, force bool) (*SyncResult, error) {
	result := &SyncResult{Total: len(openIDs)}
	if len(openIDs) == 0 {
		return result, nil
	}

	// Use a detached context so sync continues even if the HTTP request times out
	syncCtx := context.Background()

	// Sync concurrently with limited workers
	workers := 5
	if len(openIDs) < workers {
		workers = len(openIDs)
	}

	type syncRes struct {
		ok  bool
		err error
		id  string
	}

	ch := make(chan string, len(openIDs))
	results := make(chan syncRes, len(openIDs))

	for i := 0; i < workers; i++ {
		go func() {
			for openID := range ch {
				var r syncRes
				r.id = openID
				// Retry once on failure (handles transient rate limiting)
				_, err := s.syncUser(syncCtx, openID, force)
				if err != nil {
					time.Sleep(500 * time.Millisecond)
					_, err = s.syncUser(syncCtx, openID, force)
				}
				r.ok = err == nil
				r.err = err
				results <- r
			}
		}()
	}

	for _, id := range openIDs {
		ch <- id
	}
	close(ch)

	for i := 0; i < len(openIDs); i++ {
		r := <-results
		if r.ok {
			result.Synced++
		} else {
			result.Failed++
			result.FailedIDs = append(result.FailedIDs, r.id)
			s.logger.Debug("failed to sync user", zap.String("open_id", r.id), zap.Error(r.err))
		}
	}

	s.logger.Info("user sync completed",
		zap.Int("synced", result.Synced),
		zap.Int("failed", result.Failed),
		zap.Int("total", result.Total))
	return result, nil
}

func (s *UserService) setCache(info *larkbot.UserInfo) {
	s.cacheMu.Lock()
	s.cache[info.OpenID] = info
	s.cacheMu.Unlock()
}

func userToInfo(u *model.User) *larkbot.UserInfo {
	return &larkbot.UserInfo{
		OpenID:          u.OpenID,
		UnionID:         u.UnionID,
		UserID:          u.UserID,
		Name:            u.Name,
		EnName:          u.EnName,
		Avatar:          u.Avatar,
		Description:     u.Description,
		Email:           u.Email,
		City:            u.City,
		JobTitle:        u.JobTitle,
		WorkStation:     u.WorkStation,
		EmployeeNo:      u.EmployeeNo,
		Gender:          u.Gender,
		LeaderUserID:    u.LeaderUserID,
		DepartmentIDs:   u.DepartmentIDs,
		DepartmentNames: u.DepartmentNames,
		CustomAttrs:     u.CustomAttrs,
		JoinTime:        u.JoinTime,
	}
}

func infoToUser(info *larkbot.UserInfo, now time.Time) *model.User {
	return &model.User{
		OpenID:          info.OpenID,
		UnionID:         info.UnionID,
		UserID:          info.UserID,
		Name:            info.Name,
		EnName:          info.EnName,
		Avatar:          info.Avatar,
		Description:     info.Description,
		Email:           info.Email,
		City:            info.City,
		JobTitle:        info.JobTitle,
		WorkStation:     info.WorkStation,
		EmployeeNo:      info.EmployeeNo,
		Gender:          info.Gender,
		LeaderUserID:    info.LeaderUserID,
		DepartmentIDs:   info.DepartmentIDs,
		DepartmentNames: info.DepartmentNames,
		CustomAttrs:     info.CustomAttrs,
		JoinTime:        info.JoinTime,
		FirstSeen:       now,
		LastSeen:        now,
	}
}

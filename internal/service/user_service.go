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
func (s *UserService) SyncUser(ctx context.Context, openID string) (*model.User, error) {
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

// SyncAllUsers re-fetches info from Lark API for all known users.
func (s *UserService) SyncAllUsers(ctx context.Context) (int, error) {
	users, _, err := s.repo.List(repository.UserQuery{Page: 1, PageSize: 10000})
	if err != nil {
		return 0, err
	}

	synced := 0
	for _, u := range users {
		if _, err := s.SyncUser(ctx, u.OpenID); err != nil {
			s.logger.Debug("failed to sync user", zap.String("open_id", u.OpenID), zap.Error(err))
			continue
		}
		synced++
	}
	return synced, nil
}

func (s *UserService) setCache(info *larkbot.UserInfo) {
	s.cacheMu.Lock()
	s.cache[info.OpenID] = info
	s.cacheMu.Unlock()
}

func userToInfo(u *model.User) *larkbot.UserInfo {
	return &larkbot.UserInfo{
		OpenID:       u.OpenID,
		UnionID:      u.UnionID,
		UserID:       u.UserID,
		Name:         u.Name,
		EnName:       u.EnName,
		Avatar:       u.Avatar,
		Email:        u.Email,
		JobTitle:     u.JobTitle,
		WorkStation:  u.WorkStation,
		EmployeeNo:   u.EmployeeNo,
		Gender:       u.Gender,
		LeaderUserID: u.LeaderUserID,
		JoinTime:     u.JoinTime,
	}
}

func infoToUser(info *larkbot.UserInfo, now time.Time) *model.User {
	return &model.User{
		OpenID:       info.OpenID,
		UnionID:      info.UnionID,
		UserID:       info.UserID,
		Name:         info.Name,
		EnName:       info.EnName,
		Avatar:       info.Avatar,
		Email:        info.Email,
		JobTitle:     info.JobTitle,
		WorkStation:  info.WorkStation,
		EmployeeNo:   info.EmployeeNo,
		Gender:       info.Gender,
		LeaderUserID: info.LeaderUserID,
		JoinTime:     info.JoinTime,
		FirstSeen:    now,
		LastSeen:     now,
	}
}

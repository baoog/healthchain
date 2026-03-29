package service

import (
	"consentcc/internal/adapter/entity"
	"consentcc/internal/adapter/repository"
	"consentcc/internal/infra/client/jwt"
	"consentcc/internal/model"
	"consentcc/utils"
	"context"
	"github.com/google/uuid"
	"strconv"
)

type (
	UserServiceProvider interface {
		RegisterUser(ctx context.Context, role model.Role, countryCode string, nationalID, name string) (*entity.User, error)
		LoginUser(ctx context.Context, user *entity.ReqLoginUser) (string, error)
		SearchDoctorByNameAndRole(ctx context.Context, name, role, pageSize string) (*[]entity.ViewUserQuery, error)
		GetUserByNationalIDHash(ctx context.Context, nationHash string) (*entity.CitizenIdentity, error)
		SearchHodByListNationalHash(ctx context.Context, hash []string) (*[]entity.ViewUserQuery, error)
	}

	UserService struct {
		userRepo  repository.UserRepoProvider
		jwtClient jwt.JWTClientProvider
	}
)

func NewUserService(userRepo repository.UserRepoProvider, jwtClient jwt.JWTClientProvider) *UserService {
	return &UserService{
		userRepo:  userRepo,
		jwtClient: jwtClient,
	}
}

func (u *UserService) RegisterUser(
	ctx context.Context,
	role model.Role,
	countryCode string,
	nationalID string,
	name string,
) (*entity.User, error) {

	userID := uuid.New()
	nationalHash := utils.HashNationalID(countryCode, nationalID)

	user := &entity.User{
		ID:   userID.String(),
		Role: role,
		Name: name,
	}

	identity := &entity.CitizenIdentity{
		ID:             uuid.New().String(),
		UserID:         userID.String(),
		CountryCode:    countryCode,
		NationalIDHash: nationalHash,
	}

	user, err := u.userRepo.RegisterUser(ctx, user, identity)
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (u *UserService) LoginUser(ctx context.Context, req *entity.ReqLoginUser) (string, error) {
	userIdentify, err := u.userRepo.SearchByNationalIDHash(ctx, req.NationalIDHash)
	if err != nil {
		return "", err
	}

	userToken, err := u.jwtClient.Sign(userIdentify.UserID, string(userIdentify.User.Role), userIdentify.NationalIDHash)
	if err != nil {
		return "", err
	}
	return userToken, nil
}

func (u *UserService) SearchDoctorByNameAndRole(ctx context.Context, name, role, pageSize string) (*[]entity.ViewUserQuery, error) {
	pageSizeInt, err := strconv.Atoi(pageSize)
	if err != nil {
		return nil, err
	}
	return u.userRepo.SearchDoctorByNameAndRole(ctx, name, role, pageSizeInt)
}

func (u *UserService) GetUserByNationalIDHash(ctx context.Context, nationHash string) (*entity.CitizenIdentity, error) {
	return u.userRepo.SearchByNationalIDHash(ctx, nationHash)
}

func (u *UserService) SearchHodByListNationalHash(ctx context.Context, hash []string) (*[]entity.ViewUserQuery, error) {
	return u.userRepo.SearchHodByListNationalHash(ctx, hash)
}

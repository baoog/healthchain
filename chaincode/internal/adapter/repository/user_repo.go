package repository

import (
	"consentcc/internal/adapter/entity"
	"consentcc/internal/infra/client/postgres"
	"consentcc/internal/model"
	"context"
	"database/sql"
	"fmt"
	"gorm.io/gorm"
)

type (
	UserRepo struct {
		postgres *postgres.Client
	}
	UserRepoProvider interface {
		// Define user repository methods here
		RegisterUser(ctx context.Context, user *entity.User, identity *entity.CitizenIdentity) (*entity.User, error)
		SearchByNationalIDHash(ctx context.Context, nationalIDHash string) (*entity.CitizenIdentity, error)
		SearchByUserID(ctx context.Context, userID string) (*entity.CitizenIdentity, error)
		LoadUsersByListNationalHash(ctx context.Context, hashes []string) (*[]entity.CitizenIdentity, map[string]entity.CitizenIdentity, error)
		SearchDoctorByNameAndRole(ctx context.Context, name, role string, pageSize int) (*[]entity.ViewUserQuery, error)
		SearchHodByListNationalHash(ctx context.Context, hash []string) (*[]entity.ViewUserQuery, error)
	}
)

func NewUserRepo(pgClient *postgres.Client) *UserRepo {
	return &UserRepo{
		postgres: pgClient,
	}
}

func (u *UserRepo) RegisterUser(ctx context.Context, user *entity.User, identity *entity.CitizenIdentity) (*entity.User, error) {
	// Implement user registration logic here
	err := u.postgres.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(user).Error; err != nil {
			return err
		}
		if err := tx.Create(identity).Error; err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (u *UserRepo) SearchByNationalIDHash(ctx context.Context, nationalIDHash string) (*entity.CitizenIdentity, error) {
	var identity entity.CitizenIdentity
	fmt.Println("UserRepo", nationalIDHash)
	err := u.postgres.DB.WithContext(ctx).Preload("User").Where("national_id_hash = ?", nationalIDHash).First(&identity).Error
	if err != nil {
		return nil, err
	}
	return &identity, nil
}

func (u *UserRepo) SearchByUserID(ctx context.Context, userID string) (*entity.CitizenIdentity, error) {
	var user entity.CitizenIdentity
	err := u.postgres.DB.WithContext(ctx).Debug().Where("user_id = ?", userID).Preload("User").First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (u *UserRepo) LoadUsersByListNationalHash(ctx context.Context, hashes []string) (*[]entity.CitizenIdentity, map[string]entity.CitizenIdentity, error) {
	var users []entity.CitizenIdentity
	err := u.postgres.DB.WithContext(ctx).Debug().Model(&entity.CitizenIdentity{}).
		Where("national_id_hash in (?)", hashes).
		Preload("User").Find(&users).Error
	if err != nil {
		return nil, nil, err
	}

	if len(users) == 0 {
		return nil, nil, sql.ErrNoRows
	}

	mapCitizenID := make(map[string]entity.CitizenIdentity)

	for _, user := range users {
		if _, ok := mapCitizenID[user.NationalIDHash]; !ok {
			mapCitizenID[user.NationalIDHash] = user
		}
	}

	return &users, mapCitizenID, nil
}

func (u *UserRepo) SearchDoctorByNameAndRole(ctx context.Context, name, role string, pageSize int) (*[]entity.ViewUserQuery, error) {
	var users []entity.ViewUserQuery
	err := u.postgres.DB.WithContext(ctx).Debug().
		Select("ci.user_id as user_id, u.name as name, u.role as role, ci.national_id_hash").
		Table("citizen_identities ci").
		InnerJoins("JOIN users u ON u.id = ci.user_id").
		Where("u.name ilike ? AND u.role = ?", name+"%", role).
		Limit(pageSize).
		Find(&users).Error
	if err != nil {
		return nil, err
	}

	return &users, nil
}

func (u *UserRepo) SearchHodByListNationalHash(ctx context.Context, hash []string) (*[]entity.ViewUserQuery, error) {
	var users []entity.ViewUserQuery
	query := u.postgres.DB.WithContext(ctx).Debug().
		Select("ci.user_id as user_id, u.name as name, u.role as role, ci.national_id_hash").
		Table("citizen_identities ci").
		InnerJoins("JOIN users u ON u.id = ci.user_id").
		Where("u.role = ?", model.RolesConst.HoD)
	if len(hash) > 0 {
		query.Where("ci.national_id_hash in(?) AND u.role = ?", hash)
	}

	err := query.Find(&users).Error
	if err != nil {
		return nil, err
	}

	return &users, nil
}

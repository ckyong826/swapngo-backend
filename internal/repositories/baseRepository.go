package repositories

import (
	"context"
	"errors"
	"swapngo-backend/pkg/database"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type IBaseRepository[T any] interface {
	Create(ctx context.Context, entity *T) (*T, error)
	Update(ctx context.Context, entity *T) (*T, error)
	Delete(ctx context.Context, entity *T) (*T, error)
	
	FindByID(ctx context.Context, id uuid.UUID) (*T, error)
	FindByIDs(ctx context.Context, ids []uuid.UUID) ([]*T, error)
	LockById(ctx context.Context, id uuid.UUID) (*T, error)
	LockByIds(ctx context.Context, ids []uuid.UUID) ([]*T, error)
	FindAll(ctx context.Context) ([]*T, error)
	FindBy(ctx context.Context, query interface{}, args ...interface{}) ([]*T, error)
	FirstBy(ctx context.Context, query interface{}, args ...interface{}) (*T, error)
	Count(ctx context.Context, query interface{}, args ...interface{}) (int64, error)
}

type BaseRepository[T any] struct {
	db *gorm.DB
}

func NewBaseRepository[T any](db *gorm.DB) *BaseRepository[T] {
	return &BaseRepository[T]{db: db}
}

// --- 增删改 (保留原样，传入指针进行操作) ---
func (r *BaseRepository[T]) Create(ctx context.Context, entity *T) (*T, error) {
	err := database.GetDB(ctx, r.db).WithContext(ctx).Create(entity).Error
	if err != nil {
		return nil, err
	}
	return entity, nil 
}

func (r *BaseRepository[T]) Update(ctx context.Context, entity *T) (*T, error) {
	err := database.GetDB(ctx, r.db).WithContext(ctx).Save(entity).Error
	if err != nil {
		return nil, err
	}
	return entity, nil
}

func (r *BaseRepository[T]) Delete(ctx context.Context, entity *T) (*T, error) {
	err := database.GetDB(ctx, r.db).WithContext(ctx).Delete(entity).Error
	if err != nil {
		return nil, err
	}
	return entity, nil
}

// --- 读操作 (全部重构为返回实体) ---

func (r *BaseRepository[T]) FindByID(ctx context.Context, id uuid.UUID) (*T, error) {
	var entity T
	err := database.GetDB(ctx, r.db).WithContext(ctx).First(&entity, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil // 优雅处理 Not Found
		}
		return nil, err
	}
	return &entity, nil
}

func (r *BaseRepository[T]) FindByIDs(ctx context.Context, ids []uuid.UUID) ([]*T, error) {
	var entities []*T
	err := database.GetDB(ctx, r.db).WithContext(ctx).Where("id IN ?", ids).Find(&entities).Error
	return entities, err
}

func (r *BaseRepository[T]) LockById(ctx context.Context, id uuid.UUID) (*T, error) {
	var entity T
	err := database.GetDB(ctx, r.db).WithContext(ctx).Clauses(clause.Locking{Strength: "UPDATE"}).First(&entity, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &entity, nil
}

func (r *BaseRepository[T]) LockByIds(ctx context.Context, ids []uuid.UUID) ([]*T, error) {
	var entities []*T
	err := database.GetDB(ctx, r.db).WithContext(ctx).Clauses(clause.Locking{Strength: "UPDATE"}).Where("id IN ?", ids).Find(&entities).Error
	return entities, err
}

func (r *BaseRepository[T]) FindAll(ctx context.Context) ([]*T, error) {
	var entities []*T
	err := database.GetDB(ctx, r.db).WithContext(ctx).Find(&entities).Error
	return entities, err
}

func (r *BaseRepository[T]) FindBy(ctx context.Context, query interface{}, args ...interface{}) ([]*T, error) {
	var entities []*T
	err := database.GetDB(ctx, r.db).WithContext(ctx).Where(query, args...).Find(&entities).Error
	return entities, err
}

func (r *BaseRepository[T]) FirstBy(ctx context.Context, query interface{}, args ...interface{}) (*T, error) {
	var entity T
	err := database.GetDB(ctx, r.db).WithContext(ctx).Where(query, args...).First(&entity).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &entity, nil
}

func (r *BaseRepository[T]) Count(ctx context.Context, query interface{}, args ...interface{}) (int64, error) {
	var count int64
	err := database.GetDB(ctx, r.db).WithContext(ctx).Model(new(T)).Where(query, args...).Count(&count).Error
	return count, err
}
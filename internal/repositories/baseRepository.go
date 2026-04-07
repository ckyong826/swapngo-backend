package repositories

import (
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)


type IBaseRepository[T any] interface {
	Create(ctx context.Context, entity *T) error
	Update(ctx context.Context, entity *T) error
	Delete(ctx context.Context, entity *T) error
	FindByID(ctx context.Context, id uuid.UUID, entity *T) error
	FindByIDs(ctx context.Context, ids []uuid.UUID, entities *[]T) error
	LockById(ctx context.Context, id uuid.UUID, entity *T) error
	LockByIds(ctx context.Context, ids []uuid.UUID, entities *[]T) error
	FindAll(ctx context.Context, entities *[]T) error
	FindBy(ctx context.Context, query interface{}, args ...interface{}) ([]T, error)
	FirstBy(ctx context.Context, query interface{}, args ...interface{}) (T, error)
	Count(ctx context.Context, query interface{}, args ...interface{}) (int64, error)
}

type BaseRepository[T any] struct {
	db *gorm.DB
}

func NewBaseRepository[T any](db *gorm.DB) *BaseRepository[T] {
	return &BaseRepository[T]{db: db}
}

func (r *BaseRepository[T]) Create(ctx context.Context, entity *T) error {
	return r.db.WithContext(ctx).Create(entity).Error
}

func (r *BaseRepository[T]) Update(ctx context.Context, entity *T) error {
	return r.db.WithContext(ctx).Save(entity).Error
}

func (r *BaseRepository[T]) Delete(ctx context.Context, entity *T) error {
	return r.db.WithContext(ctx).Delete(entity).Error
}

func (r *BaseRepository[T]) FindByID(ctx context.Context, id uuid.UUID, entity *T) error {
	return r.db.WithContext(ctx).First(entity, id).Error
}

func (r *BaseRepository[T]) FindByIDs(ctx context.Context, ids []uuid.UUID, entities *[]T) error {
	return r.db.WithContext(ctx).Where("id IN ?", ids).Find(entities).Error
}

func (r *BaseRepository[T]) LockById(ctx context.Context, id uuid.UUID, entity *T) error {
	return r.db.WithContext(ctx).Clauses(clause.Locking{Strength: "UPDATE"}).First(entity, id).Error
}

func (r *BaseRepository[T]) LockByIds(ctx context.Context, ids []uuid.UUID, entities *[]T) error {
	return r.db.WithContext(ctx).Clauses(clause.Locking{Strength: "UPDATE"}).Where("id IN ?", ids).Find(entities).Error
}

func (r *BaseRepository[T]) FindAll(ctx context.Context, entities *[]T) error {
	return r.db.WithContext(ctx).Find(entities).Error
}

func (r *BaseRepository[T]) FindBy(ctx context.Context, query interface{}, args ...interface{}) ([]T, error) {
	var entities []T
	err := r.db.WithContext(ctx).Where(query, args...).Find(&entities).Error
	return entities, err
}

func (r *BaseRepository[T]) FirstBy(ctx context.Context, query interface{}, args ...interface{}) (T, error) {
	var entity T
	err := r.db.WithContext(ctx).Where(query, args...).First(&entity).Error
	return entity, err
}

func (r *BaseRepository[T]) Count(ctx context.Context, query interface{}, args ...interface{}) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(new(T)).Where(query, args...).Count(&count).Error
	return count, err
}
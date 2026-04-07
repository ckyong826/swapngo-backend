package services

import (
	"context"

	"swapngo-backend/internal/repositories"

	"github.com/google/uuid"
)

type IBaseService[T any] interface {
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

type BaseService[T any] struct {
	repository repositories.IBaseRepository[T]
}

func NewBaseService[T any](repository repositories.IBaseRepository[T]) *BaseService[T] {
	return &BaseService[T]{repository: repository}
}

func (s *BaseService[T]) Create(ctx context.Context, entity *T) error {
	return s.repository.Create(ctx, entity)
}

func (s *BaseService[T]) Update(ctx context.Context, entity *T) error {
	return s.repository.Update(ctx, entity)
}

func (s *BaseService[T]) Delete(ctx context.Context, entity *T) error {
	return s.repository.Delete(ctx, entity)
}

func (s *BaseService[T]) FindByID(ctx context.Context, id uuid.UUID, entity *T) error {
	return s.repository.FindByID(ctx, id, entity)
}

func (s *BaseService[T]) FindByIDs(ctx context.Context, ids []uuid.UUID, entities *[]T) error {
	return s.repository.FindByIDs(ctx, ids, entities)
}

func (s *BaseService[T]) LockById(ctx context.Context, id uuid.UUID, entity *T) error {
	return s.repository.LockById(ctx, id, entity)
}

func (s *BaseService[T]) LockByIds(ctx context.Context, ids []uuid.UUID, entities *[]T) error {
	return s.repository.LockByIds(ctx, ids, entities)
}

func (s *BaseService[T]) FindAll(ctx context.Context, entities *[]T) error {
	return s.repository.FindAll(ctx, entities)
}

func (s *BaseService[T]) FindBy(ctx context.Context, query interface{}, args ...interface{}) ([]T, error) {
	return s.repository.FindBy(ctx, query, args...)
}

func (s *BaseService[T]) FirstBy(ctx context.Context, query interface{}, args ...interface{}) (T, error) {
	return s.repository.FirstBy(ctx, query, args...)
}

func (s *BaseService[T]) Count(ctx context.Context, query interface{}, args ...interface{}) (int64, error) {
	return s.repository.Count(ctx, query, args...)
}

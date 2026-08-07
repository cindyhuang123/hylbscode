package todo

import (
	"context"

	"github.com/cindyhuang123/hylbscode/internal/db"
	"github.com/cindyhuang123/hylbscode/internal/pubsub"
	"github.com/google/uuid"
)

type Todo struct {
	ID         string
	SessionID  string
	Content    string
	Status     string
	Priority   int64
	Position   int64
	CreatedAt  int64
	UpdatedAt  int64
	FinishedAt int64
}

type Service interface {
	pubsub.Suscriber[Todo]
	List(ctx context.Context, sessionID string) ([]Todo, error)
	BulkSet(ctx context.Context, sessionID string, todos []Todo) ([]Todo, error)
	SetStatus(ctx context.Context, id, status string) (Todo, error)
	Delete(ctx context.Context, id string) error
}

type service struct {
	*pubsub.Broker[Todo]
	q db.Querier
}

func (s *service) List(ctx context.Context, sessionID string) ([]Todo, error) {
	dbTodos, err := s.q.ListTodosBySession(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	todos := make([]Todo, len(dbTodos))
	for i, dbTodo := range dbTodos {
		todos[i] = s.fromDBItem(dbTodo)
	}
	return todos, nil
}

func (s *service) BulkSet(ctx context.Context, sessionID string, todos []Todo) ([]Todo, error) {
	if err := s.q.BulkDeleteTodosBySession(ctx, sessionID); err != nil {
		return nil, err
	}
	created := make([]Todo, 0, len(todos))
	for i, t := range todos {
		dbTodo, err := s.q.CreateTodo(ctx, db.CreateTodoParams{
			ID:        uuid.New().String(),
			SessionID: sessionID,
			Content:   t.Content,
			Status:    t.Status,
			Priority:  t.Priority,
			Position:  int64(i),
		})
		if err != nil {
			return nil, err
		}
		created = append(created, s.fromDBItem(dbTodo))
		s.Publish(pubsub.CreatedEvent, s.fromDBItem(dbTodo))
	}
	return created, nil
}

func (s *service) SetStatus(ctx context.Context, id, status string) (Todo, error) {
	dbTodo, err := s.q.UpdateTodoStatus(ctx, db.UpdateTodoStatusParams{
		ID:     id,
		Status: status,
	})
	if err != nil {
		return Todo{}, err
	}
	todo := s.fromDBItem(dbTodo)
	s.Publish(pubsub.UpdatedEvent, todo)
	return todo, nil
}

func (s *service) Delete(ctx context.Context, id string) error {
	dbTodo, err := s.q.GetTodoByID(ctx, id)
	if err != nil {
		return err
	}
	if err := s.q.DeleteTodo(ctx, id); err != nil {
		return err
	}
	s.Publish(pubsub.DeletedEvent, s.fromDBItem(dbTodo))
	return nil
}

func (s service) fromDBItem(item db.Todo) Todo {
	return Todo{
		ID:         item.ID,
		SessionID:  item.SessionID,
		Content:    item.Content,
		Status:     item.Status,
		Priority:   item.Priority,
		Position:   item.Position,
		CreatedAt:  item.CreatedAt,
		UpdatedAt:  item.UpdatedAt,
		FinishedAt: item.FinishedAt.Int64,
	}
}

func NewService(q db.Querier) Service {
	broker := pubsub.NewBroker[Todo]()
	return &service{
		broker,
		q,
	}
}

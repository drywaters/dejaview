package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/drywaters/seenema/internal/model"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PersonRepository handles database operations for persons (read-only)
type PersonRepository struct {
	pool *pgxpool.Pool
}

// NewPersonRepository creates a new PersonRepository
func NewPersonRepository(pool *pgxpool.Pool) *PersonRepository {
	return &PersonRepository{pool: pool}
}

// GetAll retrieves all persons ordered by initial
func (r *PersonRepository) GetAll(ctx context.Context) ([]*model.Person, error) {
	query := `SELECT id, initial, name FROM persons ORDER BY initial`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("get all persons: %w", err)
	}
	defer rows.Close()

	var persons []*model.Person
	for rows.Next() {
		person := &model.Person{}
		if err := rows.Scan(&person.ID, &person.Initial, &person.Name); err != nil {
			return nil, fmt.Errorf("scan person: %w", err)
		}
		persons = append(persons, person)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate persons: %w", err)
	}

	return persons, nil
}

// GetByID retrieves a person by their ID
func (r *PersonRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Person, error) {
	query := `SELECT id, initial, name FROM persons WHERE id = $1`

	person := &model.Person{}
	err := r.pool.QueryRow(ctx, query, id).Scan(&person.ID, &person.Initial, &person.Name)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get person by id: %w", err)
	}

	return person, nil
}

// GetByInitial retrieves a person by their initial
func (r *PersonRepository) GetByInitial(ctx context.Context, initial string) (*model.Person, error) {
	query := `SELECT id, initial, name FROM persons WHERE initial = $1`

	person := &model.Person{}
	err := r.pool.QueryRow(ctx, query, initial).Scan(&person.ID, &person.Initial, &person.Name)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get person by initial: %w", err)
	}

	return person, nil
}

// GetAllAsMap returns all persons as a map keyed by initial
func (r *PersonRepository) GetAllAsMap(ctx context.Context) (map[string]*model.Person, error) {
	persons, err := r.GetAll(ctx)
	if err != nil {
		return nil, err
	}

	personMap := make(map[string]*model.Person, len(persons))
	for _, p := range persons {
		personMap[p.Initial] = p
	}

	return personMap, nil
}


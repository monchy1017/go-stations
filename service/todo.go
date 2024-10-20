package service

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/TechBowl-japan/go-stations/model"
)

// A TODOService implements CRUD of TODO entities.
type TODOService struct {
	DB *sql.DB
}

// NewTODOService returns new TODOService.
func NewTODOService(db *sql.DB) *TODOService {
	return &TODOService{
		DB: db,
	}
}

// CreateTODO creates a TODO on DB.
func (s *TODOService) CreateTODO(ctx context.Context, subject, description string) (*model.TODO, error) {
	const (
		insert  = `INSERT INTO todos(subject, description) VALUES(?, ?)`
		confirm = `SELECT id, subject, description, created_at, updated_at FROM todos WHERE id = ?`
	)
	// TODOの作成
	result, err := s.DB.ExecContext(ctx, insert, subject, description)
	if err != nil {
		return nil, err
	}

	// IDを取得
	id, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}

	// TODOの取得
	row := s.DB.QueryRowContext(ctx, confirm, id)
	todo := &model.TODO{}
	err = row.Scan(&todo.ID, &todo.Subject, &todo.Description, &todo.CreatedAt, &todo.UpdatedAt)
	if err != nil {
		return nil, err
	}

	return todo, nil
}

// ReadTODO reads TODOs on DB.
func (s *TODOService) ReadTODO(ctx context.Context, prevID, size int64) ([]*model.TODO, error) {
	const (
		read       = `SELECT id, subject, description, created_at, updated_at FROM todos ORDER BY id DESC LIMIT ?`
		readWithID = `SELECT id, subject, description, created_at, updated_at FROM todos WHERE id < ? ORDER BY id DESC LIMIT ?`
	)

	if size == 0 {
		return []*model.TODO{}, nil
	}

	var rows *sql.Rows
	var err error

	//PrevIDが指定されているかどうか
	if prevID > 0 {
		rows, err = s.DB.QueryContext(ctx, readWithID, prevID, size)
	} else {
		rows, err = s.DB.QueryContext(ctx, read, size)
	}

	if err != nil {
		return nil, err
	}
	defer func() {
		if cerr := rows.Close(); cerr != nil {
			err = cerr
		}
	}()

	var todos []*model.TODO

	// TODOの取得(繰り返し処理し、スライスに追加)
	for rows.Next() {
		todo := &model.TODO{}
		if err := rows.Scan(&todo.ID, &todo.Subject, &todo.Description, &todo.CreatedAt, &todo.UpdatedAt); err != nil {
			return nil, err
		}
		todos = append(todos, todo)
	}

	// エラー処理
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if len(todos) == 0 {
		return []*model.TODO{}, nil
	}
	if size > 0 && len(todos) > int(size) {
		return todos[:size], nil
	}

	return todos, nil
}

// UpdateTODO updates the TODO on DB.
func (s *TODOService) UpdateTODO(ctx context.Context, id int64, subject, description string) (*model.TODO, error) {

	if id == 0 {
		return nil, &model.ErrNotFound{}
	}

	const (
		update  = `UPDATE todos SET subject = ?, description = ? WHERE id = ?`
		confirm = `SELECT subject, description, created_at, updated_at FROM todos WHERE id = ?`
	)

	// TODOの更新
	result, err := s.DB.ExecContext(ctx, update, subject, description, id)
	if err != nil {
		return nil, err
	}

	//更新されたTODOが存在しない場合
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return nil, err
	}
	if rowsAffected == 0 {
		return nil, &model.ErrNotFound{}
	}

	// 更新されたTODOの取得
	row := s.DB.QueryRowContext(ctx, confirm, id)
	todo := &model.TODO{}
	todo.ID = id

	err = row.Scan(&todo.Subject, &todo.Description, &todo.CreatedAt, &todo.UpdatedAt)
	if err != nil {
		return nil, err
	}

	return todo, nil
}

// DeleteTODO deletes TODOs on DB by ids.
func (s *TODOService) DeleteTODO(ctx context.Context, ids []int64) error {
	const deleteFmt = `DELETE FROM todos WHERE id IN (?%s)`

	if len(ids) == 0 {
		return nil
	}

	// idsの数だけ?を作成
	placeholders := strings.Repeat(",?", len(ids)-1)
	query := fmt.Sprintf(deleteFmt, placeholders)

	// idsをinterface{}のスライスに変換
	args := make([]interface{}, len(ids))
	for i, id := range ids {
		args[i] = id
	}

	result, err := s.DB.ExecContext(ctx, query, args...)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return &model.ErrNotFound{Resource: "TODO"}
	}
	return nil
}

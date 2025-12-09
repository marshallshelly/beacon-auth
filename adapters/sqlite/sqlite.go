package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/marshallshelly/beacon-auth/core"
	_ "modernc.org/sqlite"
)

type SQLiteAdapter struct {
	db *sql.DB
}

type Config struct {
	DataSourceName string
}

func New(ctx context.Context, cfg *Config) (*SQLiteAdapter, error) {
	if cfg.DataSourceName == "" {
		cfg.DataSourceName = "file:beaconauth.db?cache=shared&mode=rwc"
	}

	db, err := sql.Open("sqlite", cfg.DataSourceName)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// SQLite guidelines for concurrency
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(0)

	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &SQLiteAdapter{db: db}, nil
}

func (s *SQLiteAdapter) ID() string {
	return "sqlite"
}

func (s *SQLiteAdapter) Create(ctx context.Context, model string, data map[string]interface{}) (map[string]interface{}, error) {
	return create(ctx, s.db, model, data, s)
}

func (s *SQLiteAdapter) FindOne(ctx context.Context, query *core.Query) (map[string]interface{}, error) {
	return findOne(ctx, s.db, query)
}

func (s *SQLiteAdapter) FindMany(ctx context.Context, query *core.Query) ([]map[string]interface{}, error) {
	return findMany(ctx, s.db, query)
}

func (s *SQLiteAdapter) Update(ctx context.Context, query *core.Query, data map[string]interface{}) (map[string]interface{}, error) {
	return update(ctx, s.db, query, data, s)
}

func (s *SQLiteAdapter) UpdateMany(ctx context.Context, query *core.Query, data map[string]interface{}) (int64, error) {
	return updateMany(ctx, s.db, query, data)
}

func (s *SQLiteAdapter) Delete(ctx context.Context, query *core.Query) error {
	return deleteOne(ctx, s.db, query)
}

func (s *SQLiteAdapter) DeleteMany(ctx context.Context, query *core.Query) (int64, error) {
	return deleteMany(ctx, s.db, query)
}

func (s *SQLiteAdapter) Count(ctx context.Context, query *core.Query) (int64, error) {
	return count(ctx, s.db, query)
}

func (s *SQLiteAdapter) Transaction(ctx context.Context, fn func(core.Adapter) error) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	txAdapter := &sqliteTransaction{
		tx:      tx,
		adapter: s,
	}

	if err := fn(txAdapter); err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			return fmt.Errorf("transaction error: %w, rollback error: %v", err, rbErr)
		}
		return err
	}

	return tx.Commit()
}

func (s *SQLiteAdapter) Ping(ctx context.Context) error {
	return s.db.PingContext(ctx)
}

func (s *SQLiteAdapter) Close() error {
	return s.db.Close()
}

// sqliteTransaction wraps a SQLite transaction
type sqliteTransaction struct {
	tx      *sql.Tx
	adapter *SQLiteAdapter
}

func (t *sqliteTransaction) ID() string { return "sqlite-tx" }

func (t *sqliteTransaction) Create(ctx context.Context, model string, data map[string]interface{}) (map[string]interface{}, error) {
	return create(ctx, t.tx, model, data, t)
}

func (t *sqliteTransaction) FindOne(ctx context.Context, query *core.Query) (map[string]interface{}, error) {
	return findOne(ctx, t.tx, query)
}

func (t *sqliteTransaction) FindMany(ctx context.Context, query *core.Query) ([]map[string]interface{}, error) {
	return findMany(ctx, t.tx, query)
}

func (t *sqliteTransaction) Update(ctx context.Context, query *core.Query, data map[string]interface{}) (map[string]interface{}, error) {
	return update(ctx, t.tx, query, data, t)
}

func (t *sqliteTransaction) UpdateMany(ctx context.Context, query *core.Query, data map[string]interface{}) (int64, error) {
	return updateMany(ctx, t.tx, query, data)
}

func (t *sqliteTransaction) Delete(ctx context.Context, query *core.Query) error {
	return deleteOne(ctx, t.tx, query)
}

func (t *sqliteTransaction) DeleteMany(ctx context.Context, query *core.Query) (int64, error) {
	return deleteMany(ctx, t.tx, query)
}

func (t *sqliteTransaction) Count(ctx context.Context, query *core.Query) (int64, error) {
	return count(ctx, t.tx, query)
}

func (t *sqliteTransaction) Transaction(ctx context.Context, fn func(core.Adapter) error) error {
	return fn(t)
}

func (t *sqliteTransaction) Ping(ctx context.Context) error { return nil }
func (t *sqliteTransaction) Close() error                   { return nil }

type queryExecuter interface {
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row
}

func create(ctx context.Context, db queryExecuter, model string, data map[string]interface{}, finder core.Adapter) (map[string]interface{}, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("no data provided")
	}

	columns := make([]string, 0, len(data))
	placeholders := make([]string, 0, len(data))
	values := make([]interface{}, 0, len(data))

	for col, val := range data {
		columns = append(columns, col)
		placeholders = append(placeholders, "?")
		values = append(values, val)
	}

	query := fmt.Sprintf(
		"INSERT INTO %s (%s) VALUES (%s)",
		model,
		strings.Join(columns, ", "),
		strings.Join(placeholders, ", "),
	)

	_, err := db.ExecContext(ctx, query, values...)
	if err != nil {
		return nil, err
	}

	if id, ok := data["id"]; ok {
		return finder.FindOne(ctx, &core.Query{
			Model: model,
			Where: []core.WhereClause{
				{Field: "id", Operator: core.OpEqual, Value: id},
			},
		})
	}
	return data, nil
}

func findOne(ctx context.Context, db queryExecuter, query *core.Query) (map[string]interface{}, error) {
	sqlStr, args, err := buildSelectQuery(query, true)
	if err != nil {
		return nil, err
	}

	rows, err := db.QueryContext(ctx, sqlStr, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	if !rows.Next() {
		return nil, nil
	}

	return scanRowsDynamic(rows)
}

func findMany(ctx context.Context, db queryExecuter, query *core.Query) ([]map[string]interface{}, error) {
	sqlStr, args, err := buildSelectQuery(query, false)
	if err != nil {
		return nil, err
	}

	rows, err := db.QueryContext(ctx, sqlStr, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	results := make([]map[string]interface{}, 0)
	for rows.Next() {
		result, err := scanRowsDynamic(rows)
		if err != nil {
			return nil, err
		}
		results = append(results, result)
	}

	return results, rows.Err()
}

func update(ctx context.Context, db queryExecuter, query *core.Query, data map[string]interface{}, finder core.Adapter) (map[string]interface{}, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("no data provided")
	}

	setClauses := make([]string, 0, len(data))
	values := make([]interface{}, 0, len(data))

	for col, val := range data {
		setClauses = append(setClauses, fmt.Sprintf("%s = ?", col))
		values = append(values, val)
	}

	whereClause, whereArgs, err := buildWhereClause(query.Where)
	if err != nil {
		return nil, err
	}
	values = append(values, whereArgs...)

	// SQLite doesn't support LIMIT in UPDATE
	sqlStr := fmt.Sprintf(
		"UPDATE %s SET %s%s",
		query.Model,
		strings.Join(setClauses, ", "),
		whereClause,
	)

	_, err = db.ExecContext(ctx, sqlStr, values...)
	if err != nil {
		return nil, err
	}

	return finder.FindOne(ctx, query)
}

func updateMany(ctx context.Context, db queryExecuter, query *core.Query, data map[string]interface{}) (int64, error) {
	if len(data) == 0 {
		return 0, fmt.Errorf("no data provided")
	}

	setClauses := make([]string, 0, len(data))
	values := make([]interface{}, 0, len(data))

	for col, val := range data {
		setClauses = append(setClauses, fmt.Sprintf("%s = ?", col))
		values = append(values, val)
	}

	whereClause, whereArgs, err := buildWhereClause(query.Where)
	if err != nil {
		return 0, err
	}
	values = append(values, whereArgs...)

	sqlStr := fmt.Sprintf(
		"UPDATE %s SET %s%s",
		query.Model,
		strings.Join(setClauses, ", "),
		whereClause,
	)

	result, err := db.ExecContext(ctx, sqlStr, values...)
	if err != nil {
		return 0, err
	}

	return result.RowsAffected()
}

func deleteOne(ctx context.Context, db queryExecuter, query *core.Query) error {
	whereClause, args, err := buildWhereClause(query.Where)
	if err != nil {
		return err
	}

	// SQLite doesn't support LIMIT in DELETE (without compile flag)
	sqlStr := fmt.Sprintf("DELETE FROM %s%s", query.Model, whereClause)
	_, err = db.ExecContext(ctx, sqlStr, args...)
	return err
}

func deleteMany(ctx context.Context, db queryExecuter, query *core.Query) (int64, error) {
	whereClause, args, err := buildWhereClause(query.Where)
	if err != nil {
		return 0, err
	}

	sqlStr := fmt.Sprintf("DELETE FROM %s%s", query.Model, whereClause)
	result, err := db.ExecContext(ctx, sqlStr, args...)
	if err != nil {
		return 0, err
	}

	return result.RowsAffected()
}

func count(ctx context.Context, db queryExecuter, query *core.Query) (int64, error) {
	whereClause, args, err := buildWhereClause(query.Where)
	if err != nil {
		return 0, err
	}

	sqlStr := fmt.Sprintf("SELECT COUNT(*) FROM %s%s", query.Model, whereClause)

	var count int64
	err = db.QueryRowContext(ctx, sqlStr, args...).Scan(&count)
	if err != nil {
		return 0, err
	}

	return count, nil
}

func buildSelectQuery(query *core.Query, limit1 bool) (string, []interface{}, error) {
	whereClause, args, err := buildWhereClause(query.Where)
	if err != nil {
		return "", nil, err
	}

	sqlStr := fmt.Sprintf("SELECT * FROM %s%s", query.Model, whereClause)

	if len(query.OrderBy) > 0 {
		orderClauses := make([]string, 0, len(query.OrderBy))
		for _, order := range query.OrderBy {
			direction := "ASC"
			if order.Desc {
				direction = "DESC"
			}
			orderClauses = append(orderClauses, fmt.Sprintf("%s %s", order.Field, direction))
		}
		sqlStr += " ORDER BY " + strings.Join(orderClauses, ", ")
	}

	if limit1 {
		sqlStr += " LIMIT 1"
	} else if query.Limit > 0 {
		sqlStr += fmt.Sprintf(" LIMIT %d", query.Limit)
	}

	if query.Offset > 0 {
		sqlStr += fmt.Sprintf(" OFFSET %d", query.Offset)
	}

	return sqlStr, args, nil
}

func buildWhereClause(where []core.WhereClause) (string, []interface{}, error) {
	if len(where) == 0 {
		return "", nil, nil
	}

	clauses := make([]string, 0, len(where))
	args := make([]interface{}, 0, len(where))

	for _, clause := range where {
		sql, clauseArgs, err := buildSingleWhereClause(clause)
		if err != nil {
			return "", nil, err
		}
		clauses = append(clauses, sql)
		args = append(args, clauseArgs...)
	}

	return " WHERE " + strings.Join(clauses, " AND "), args, nil
}

func buildSingleWhereClause(clause core.WhereClause) (string, []interface{}, error) {
	switch clause.Operator {
	case core.OpEqual:
		return fmt.Sprintf("%s = ?", clause.Field), []interface{}{clause.Value}, nil
	case core.OpNotEqual:
		return fmt.Sprintf("%s != ?", clause.Field), []interface{}{clause.Value}, nil
	case core.OpGreaterThan:
		return fmt.Sprintf("%s > ?", clause.Field), []interface{}{clause.Value}, nil
	case core.OpGreaterOrEqual:
		return fmt.Sprintf("%s >= ?", clause.Field), []interface{}{clause.Value}, nil
	case core.OpLessThan:
		return fmt.Sprintf("%s < ?", clause.Field), []interface{}{clause.Value}, nil
	case core.OpLessOrEqual:
		return fmt.Sprintf("%s <= ?", clause.Field), []interface{}{clause.Value}, nil
	case core.OpLike:
		return fmt.Sprintf("%s LIKE ?", clause.Field), []interface{}{clause.Value}, nil
	case core.OpIn:
		values, ok := clause.Value.([]interface{})
		if !ok {
			return "", nil, fmt.Errorf("IN operator requires []interface{} value")
		}
		placeholders := make([]string, len(values))
		for i := range values {
			placeholders[i] = "?"
		}
		return fmt.Sprintf("%s IN (%s)", clause.Field, strings.Join(placeholders, ", ")), values, nil
	case core.OpNotIn:
		values, ok := clause.Value.([]interface{})
		if !ok {
			return "", nil, fmt.Errorf("NOT IN operator requires []interface{} value")
		}
		placeholders := make([]string, len(values))
		for i := range values {
			placeholders[i] = "?"
		}
		return fmt.Sprintf("%s NOT IN (%s)", clause.Field, strings.Join(placeholders, ", ")), values, nil
	case core.OpIsNull:
		return fmt.Sprintf("%s IS NULL", clause.Field), nil, nil
	case core.OpIsNotNull:
		return fmt.Sprintf("%s IS NOT NULL", clause.Field), nil, nil
	default:
		return "", nil, fmt.Errorf("unsupported operator: %v", clause.Operator)
	}
}

func scanRowsDynamic(rows *sql.Rows) (map[string]interface{}, error) {
	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	values := make([]interface{}, len(columns))
	valuePtrs := make([]interface{}, len(columns))
	for i := range values {
		valuePtrs[i] = &values[i]
	}

	if err := rows.Scan(valuePtrs...); err != nil {
		return nil, err
	}

	result := make(map[string]interface{})
	for i, col := range columns {
		val := values[i]
		if b, ok := val.([]byte); ok {
			result[col] = string(b)
		} else {
			result[col] = val
		}
	}

	return result, nil
}

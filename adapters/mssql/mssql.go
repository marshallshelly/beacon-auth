package mssql

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/marshallshelly/beacon-auth/core"
	_ "github.com/microsoft/go-mssqldb"
)

type MSSQLAdapter struct {
	db *sql.DB
}

type Config struct {
	Host     string
	Port     int
	Database string
	Username string
	Password string
	Params   map[string]string
	MaxConns int
	MinConns int
}

func New(ctx context.Context, cfg *Config) (*MSSQLAdapter, error) {
	if cfg.Port == 0 {
		cfg.Port = 1433
	}
	if cfg.MaxConns == 0 {
		cfg.MaxConns = 10
	}

	query := url.Values{}
	query.Add("database", cfg.Database)
	for k, v := range cfg.Params {
		query.Add(k, v)
	}

	u := &url.URL{
		Scheme:   "sqlserver",
		User:     url.UserPassword(cfg.Username, cfg.Password),
		Host:     fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		RawQuery: query.Encode(),
	}

	db, err := sql.Open("sqlserver", u.String())
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	db.SetMaxOpenConns(cfg.MaxConns)
	db.SetMaxIdleConns(cfg.MinConns)
	db.SetConnMaxLifetime(time.Hour)

	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &MSSQLAdapter{db: db}, nil
}

func (m *MSSQLAdapter) ID() string {
	return "mssql"
}

func (m *MSSQLAdapter) Create(ctx context.Context, model string, data map[string]interface{}) (map[string]interface{}, error) {
	return create(ctx, m.db, model, data)
}

func (m *MSSQLAdapter) FindOne(ctx context.Context, query *core.Query) (map[string]interface{}, error) {
	return findOne(ctx, m.db, query)
}

func (m *MSSQLAdapter) FindMany(ctx context.Context, query *core.Query) ([]map[string]interface{}, error) {
	return findMany(ctx, m.db, query)
}

func (m *MSSQLAdapter) Update(ctx context.Context, query *core.Query, data map[string]interface{}) (map[string]interface{}, error) {
	return update(ctx, m.db, query, data)
}

func (m *MSSQLAdapter) UpdateMany(ctx context.Context, query *core.Query, data map[string]interface{}) (int64, error) {
	return updateMany(ctx, m.db, query, data)
}

func (m *MSSQLAdapter) Delete(ctx context.Context, query *core.Query) error {
	return deleteOne(ctx, m.db, query)
}

func (m *MSSQLAdapter) DeleteMany(ctx context.Context, query *core.Query) (int64, error) {
	return deleteMany(ctx, m.db, query)
}

func (m *MSSQLAdapter) Count(ctx context.Context, query *core.Query) (int64, error) {
	return count(ctx, m.db, query)
}

func (m *MSSQLAdapter) Transaction(ctx context.Context, fn func(core.Adapter) error) error {
	tx, err := m.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	txAdapter := &mssqlTransaction{
		tx:      tx,
		adapter: m,
	}

	if err := fn(txAdapter); err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			return fmt.Errorf("transaction error: %w, rollback error: %v", err, rbErr)
		}
		return err
	}

	return tx.Commit()
}

func (m *MSSQLAdapter) Ping(ctx context.Context) error {
	return m.db.PingContext(ctx)
}

func (m *MSSQLAdapter) Close() error {
	return m.db.Close()
}

type mssqlTransaction struct {
	tx      *sql.Tx
	adapter *MSSQLAdapter
}

func (t *mssqlTransaction) ID() string { return "mssql-tx" }

func (t *mssqlTransaction) Create(ctx context.Context, model string, data map[string]interface{}) (map[string]interface{}, error) {
	return create(ctx, t.tx, model, data)
}

func (t *mssqlTransaction) FindOne(ctx context.Context, query *core.Query) (map[string]interface{}, error) {
	return findOne(ctx, t.tx, query)
}

func (t *mssqlTransaction) FindMany(ctx context.Context, query *core.Query) ([]map[string]interface{}, error) {
	return findMany(ctx, t.tx, query)
}

func (t *mssqlTransaction) Update(ctx context.Context, query *core.Query, data map[string]interface{}) (map[string]interface{}, error) {
	return update(ctx, t.tx, query, data)
}

func (t *mssqlTransaction) UpdateMany(ctx context.Context, query *core.Query, data map[string]interface{}) (int64, error) {
	return updateMany(ctx, t.tx, query, data)
}

func (t *mssqlTransaction) Delete(ctx context.Context, query *core.Query) error {
	return deleteOne(ctx, t.tx, query)
}

func (t *mssqlTransaction) DeleteMany(ctx context.Context, query *core.Query) (int64, error) {
	return deleteMany(ctx, t.tx, query)
}

func (t *mssqlTransaction) Count(ctx context.Context, query *core.Query) (int64, error) {
	return count(ctx, t.tx, query)
}

func (t *mssqlTransaction) Transaction(ctx context.Context, fn func(core.Adapter) error) error {
	return fn(t)
}

func (t *mssqlTransaction) Ping(ctx context.Context) error { return nil }
func (t *mssqlTransaction) Close() error                   { return nil }

type queryExecuter interface {
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row
}

func create(ctx context.Context, db queryExecuter, model string, data map[string]interface{}) (map[string]interface{}, error) {
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

	// MSSQL uses OUTPUT to return data. It must be before VALUES but after INSERT INTO ...
	// Syntax: INSERT INTO table (col) OUTPUT Inserted.* VALUES (val)
	query := fmt.Sprintf(
		"INSERT INTO %s (%s) OUTPUT Inserted.* VALUES (%s)",
		model,
		strings.Join(columns, ", "),
		strings.Join(placeholders, ", "),
	)

	rows, err := db.QueryContext(ctx, query, values...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	if !rows.Next() {
		return nil, fmt.Errorf("no rows returned from insert")
	}

	return scanRowsDynamic(rows)
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

func update(ctx context.Context, db queryExecuter, query *core.Query, data map[string]interface{}) (map[string]interface{}, error) {
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

	// MSSQL UPDATE with OUTPUT
	// Syntax: UPDATE table SET col=val OUTPUT Inserted.* WHERE ...
	// MSSQL doesn't support LIMIT in UPDATE directly in standard SQL like MySQL, but supports TOP with subqueries or CTEs.
	// But standard UPDATE ... WHERE ... is fine.
	// For "single record" safety, we rely on the Where clause.

	sqlStr := fmt.Sprintf(
		"UPDATE %s SET %s OUTPUT Inserted.*%s",
		query.Model,
		strings.Join(setClauses, ", "),
		whereClause,
	)

	rows, err := db.QueryContext(ctx, sqlStr, values...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	if !rows.Next() {
		return nil, nil
	}

	return scanRowsDynamic(rows)
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
	// MSSQL supports DELETE TOP(1) FROM ...
	whereClause, args, err := buildWhereClause(query.Where)
	if err != nil {
		return err
	}

	sqlStr := fmt.Sprintf("DELETE TOP(1) FROM %s%s", query.Model, whereClause)
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

	// MSSQL Pagination
	// If limit/offset, we need ORDER BY
	hasOffset := query.Offset > 0
	hasLimit := query.Limit > 0
	hasOrder := len(query.OrderBy) > 0

	// Basic SELECT part
	sqlStr := "SELECT "
	if limit1 && !hasOffset {
		sqlStr += "TOP 1 "
	} else if hasLimit && !hasOffset {
		sqlStr += fmt.Sprintf("TOP %d ", query.Limit)
	}
	sqlStr += fmt.Sprintf("* FROM %s%s", query.Model, whereClause)

	// ORDER BY
	if hasOrder {
		orderClauses := make([]string, 0, len(query.OrderBy))
		for _, order := range query.OrderBy {
			direction := "ASC"
			if order.Desc {
				direction = "DESC"
			}
			orderClauses = append(orderClauses, fmt.Sprintf("%s %s", order.Field, direction))
		}
		sqlStr += " ORDER BY " + strings.Join(orderClauses, ", ")
	} else if hasOffset {
		// OFFSET requires ORDER BY in MSSQL
		sqlStr += " ORDER BY (SELECT NULL)"
	}

	// OFFSET / FETCH
	if hasOffset {
		sqlStr += fmt.Sprintf(" OFFSET %d ROWS", query.Offset)
		if hasLimit {
			sqlStr += fmt.Sprintf(" FETCH NEXT %d ROWS ONLY", query.Limit)
		} else if limit1 {
			sqlStr += " FETCH NEXT 1 ROWS ONLY"
		}
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
	// Re-use logic for placeholders (?)
	// MSSQL '?' support depends on driver usage. go-mssqldb supports it.
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

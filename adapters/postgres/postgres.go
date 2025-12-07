package postgres

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/marshallshelly/beaconauth/core"
)

// PostgresAdapter implements the Adapter interface for PostgreSQL
type PostgresAdapter struct {
	pool *pgxpool.Pool
}

// Config holds PostgreSQL configuration
type Config struct {
	Host     string
	Port     int
	Database string
	Username string
	Password string
	SSLMode  string
	MaxConns int32
	MinConns int32
}

// New creates a new PostgreSQL adapter
func New(ctx context.Context, cfg *Config) (*PostgresAdapter, error) {
	if cfg.Port == 0 {
		cfg.Port = 5432
	}
	if cfg.SSLMode == "" {
		cfg.SSLMode = "prefer"
	}
	if cfg.MaxConns == 0 {
		cfg.MaxConns = 10
	}
	if cfg.MinConns == 0 {
		cfg.MinConns = 2
	}

	connString := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s pool_max_conns=%d pool_min_conns=%d",
		cfg.Host,
		cfg.Port,
		cfg.Username,
		cfg.Password,
		cfg.Database,
		cfg.SSLMode,
		cfg.MaxConns,
		cfg.MinConns,
	)

	poolConfig, err := pgxpool.ParseConfig(connString)
	if err != nil {
		return nil, fmt.Errorf("failed to parse connection string: %w", err)
	}

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	return &PostgresAdapter{pool: pool}, nil
}

// ID returns the adapter identifier
func (p *PostgresAdapter) ID() string {
	return "postgres"
}

// Create creates a new record
func (p *PostgresAdapter) Create(ctx context.Context, model string, data map[string]interface{}) (map[string]interface{}, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("no data provided")
	}

	columns := make([]string, 0, len(data))
	placeholders := make([]string, 0, len(data))
	values := make([]interface{}, 0, len(data))
	returning := make([]string, 0, len(data))

	i := 1
	for col, val := range data {
		columns = append(columns, col)
		placeholders = append(placeholders, fmt.Sprintf("$%d", i))
		values = append(values, val)
		returning = append(returning, col)
		i++
	}

	query := fmt.Sprintf(
		"INSERT INTO %s (%s) VALUES (%s) RETURNING %s",
		model,
		strings.Join(columns, ", "),
		strings.Join(placeholders, ", "),
		strings.Join(returning, ", "),
	)

	row := p.pool.QueryRow(ctx, query, values...)
	return p.scanRow(row, returning)
}

// FindOne finds a single record matching the query
func (p *PostgresAdapter) FindOne(ctx context.Context, query *core.Query) (map[string]interface{}, error) {
	sql, args, err := p.buildSelectQuery(query, true)
	if err != nil {
		return nil, err
	}

	rows, err := p.pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	if !rows.Next() {
		return nil, nil
	}

	result, err := p.scanRowsDynamic(rows)
	if err != nil {
		return nil, err
	}

	return result, nil
}

// FindMany finds all records matching the query
func (p *PostgresAdapter) FindMany(ctx context.Context, query *core.Query) ([]map[string]interface{}, error) {
	sql, args, err := p.buildSelectQuery(query, false)
	if err != nil {
		return nil, err
	}

	rows, err := p.pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	results := make([]map[string]interface{}, 0)
	for rows.Next() {
		result, err := p.scanRowsDynamic(rows)
		if err != nil {
			return nil, err
		}
		results = append(results, result)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return results, nil
}

// Update updates a single record matching the query
func (p *PostgresAdapter) Update(ctx context.Context, query *core.Query, data map[string]interface{}) (map[string]interface{}, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("no data provided")
	}

	setClauses := make([]string, 0, len(data))
	values := make([]interface{}, 0, len(data))

	i := 1
	for col, val := range data {
		setClauses = append(setClauses, fmt.Sprintf("%s = $%d", col, i))
		values = append(values, val)
		i++
	}

	whereClause, whereArgs, err := p.buildWhereClause(query.Where, i)
	if err != nil {
		return nil, err
	}
	values = append(values, whereArgs...)

	sql := fmt.Sprintf(
		"UPDATE %s SET %s%s RETURNING *",
		query.Model,
		strings.Join(setClauses, ", "),
		whereClause,
	)

	rows, err := p.pool.Query(ctx, sql, values...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	if !rows.Next() {
		return nil, nil
	}

	result, err := p.scanRowsDynamic(rows)
	if err != nil {
		return nil, err
	}

	return result, nil
}

// UpdateMany updates all records matching the query
func (p *PostgresAdapter) UpdateMany(ctx context.Context, query *core.Query, data map[string]interface{}) (int64, error) {
	if len(data) == 0 {
		return 0, fmt.Errorf("no data provided")
	}

	setClauses := make([]string, 0, len(data))
	values := make([]interface{}, 0, len(data))

	i := 1
	for col, val := range data {
		setClauses = append(setClauses, fmt.Sprintf("%s = $%d", col, i))
		values = append(values, val)
		i++
	}

	whereClause, whereArgs, err := p.buildWhereClause(query.Where, i)
	if err != nil {
		return 0, err
	}
	values = append(values, whereArgs...)

	sql := fmt.Sprintf(
		"UPDATE %s SET %s%s",
		query.Model,
		strings.Join(setClauses, ", "),
		whereClause,
	)

	result, err := p.pool.Exec(ctx, sql, values...)
	if err != nil {
		return 0, err
	}

	return result.RowsAffected(), nil
}

// Delete deletes a single record matching the query
func (p *PostgresAdapter) Delete(ctx context.Context, query *core.Query) error {
	whereClause, args, err := p.buildWhereClause(query.Where, 1)
	if err != nil {
		return err
	}

	sql := fmt.Sprintf("DELETE FROM %s%s", query.Model, whereClause)

	_, err = p.pool.Exec(ctx, sql, args...)
	return err
}

// DeleteMany deletes all records matching the query
func (p *PostgresAdapter) DeleteMany(ctx context.Context, query *core.Query) (int64, error) {
	whereClause, args, err := p.buildWhereClause(query.Where, 1)
	if err != nil {
		return 0, err
	}

	sql := fmt.Sprintf("DELETE FROM %s%s", query.Model, whereClause)

	result, err := p.pool.Exec(ctx, sql, args...)
	if err != nil {
		return 0, err
	}

	return result.RowsAffected(), nil
}

// Count counts records matching the query
func (p *PostgresAdapter) Count(ctx context.Context, query *core.Query) (int64, error) {
	whereClause, args, err := p.buildWhereClause(query.Where, 1)
	if err != nil {
		return 0, err
	}

	sql := fmt.Sprintf("SELECT COUNT(*) FROM %s%s", query.Model, whereClause)

	var count int64
	err = p.pool.QueryRow(ctx, sql, args...).Scan(&count)
	if err != nil {
		return 0, err
	}

	return count, nil
}

// Transaction executes a function in a transaction
func (p *PostgresAdapter) Transaction(ctx context.Context, fn func(core.Adapter) error) error {
	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return err
	}

	txAdapter := &postgresTransaction{
		tx: tx,
	}

	if err := fn(txAdapter); err != nil {
		if rbErr := tx.Rollback(ctx); rbErr != nil {
			return fmt.Errorf("transaction error: %w, rollback error: %v", err, rbErr)
		}
		return err
	}

	return tx.Commit(ctx)
}

// Ping checks the connection
func (p *PostgresAdapter) Ping(ctx context.Context) error {
	return p.pool.Ping(ctx)
}

// Close closes the connection pool
func (p *PostgresAdapter) Close() error {
	p.pool.Close()
	return nil
}

// buildSelectQuery builds a SELECT query from a Query struct
func (p *PostgresAdapter) buildSelectQuery(query *core.Query, limit1 bool) (string, []interface{}, error) {
	whereClause, args, err := p.buildWhereClause(query.Where, 1)
	if err != nil {
		return "", nil, err
	}

	sql := fmt.Sprintf("SELECT * FROM %s%s", query.Model, whereClause)

	// Add ORDER BY
	if len(query.OrderBy) > 0 {
		orderClauses := make([]string, 0, len(query.OrderBy))
		for _, order := range query.OrderBy {
			direction := "ASC"
			if order.Desc {
				direction = "DESC"
			}
			orderClauses = append(orderClauses, fmt.Sprintf("%s %s", order.Field, direction))
		}
		sql += " ORDER BY " + strings.Join(orderClauses, ", ")
	}

	// Add LIMIT
	if limit1 {
		sql += " LIMIT 1"
	} else if query.Limit > 0 {
		sql += fmt.Sprintf(" LIMIT %d", query.Limit)
	}

	// Add OFFSET
	if query.Offset > 0 {
		sql += fmt.Sprintf(" OFFSET %d", query.Offset)
	}

	return sql, args, nil
}

// buildWhereClause builds a WHERE clause from WhereClause slice
func (p *PostgresAdapter) buildWhereClause(where []core.WhereClause, startIndex int) (string, []interface{}, error) {
	if len(where) == 0 {
		return "", nil, nil
	}

	clauses := make([]string, 0, len(where))
	args := make([]interface{}, 0, len(where))
	index := startIndex

	for _, clause := range where {
		sql, clauseArgs, err := p.buildSingleWhereClause(clause, index)
		if err != nil {
			return "", nil, err
		}
		clauses = append(clauses, sql)
		args = append(args, clauseArgs...)
		index += len(clauseArgs)
	}

	return " WHERE " + strings.Join(clauses, " AND "), args, nil
}

// buildSingleWhereClause builds a single WHERE clause
func (p *PostgresAdapter) buildSingleWhereClause(clause core.WhereClause, startIndex int) (string, []interface{}, error) {
	switch clause.Operator {
	case core.OpEqual:
		return fmt.Sprintf("%s = $%d", clause.Field, startIndex), []interface{}{clause.Value}, nil

	case core.OpNotEqual:
		return fmt.Sprintf("%s != $%d", clause.Field, startIndex), []interface{}{clause.Value}, nil

	case core.OpGreaterThan:
		return fmt.Sprintf("%s > $%d", clause.Field, startIndex), []interface{}{clause.Value}, nil

	case core.OpGreaterOrEqual:
		return fmt.Sprintf("%s >= $%d", clause.Field, startIndex), []interface{}{clause.Value}, nil

	case core.OpLessThan:
		return fmt.Sprintf("%s < $%d", clause.Field, startIndex), []interface{}{clause.Value}, nil

	case core.OpLessOrEqual:
		return fmt.Sprintf("%s <= $%d", clause.Field, startIndex), []interface{}{clause.Value}, nil

	case core.OpLike:
		return fmt.Sprintf("%s LIKE $%d", clause.Field, startIndex), []interface{}{clause.Value}, nil

	case core.OpIn:
		values, ok := clause.Value.([]interface{})
		if !ok {
			return "", nil, fmt.Errorf("IN operator requires []interface{} value")
		}
		placeholders := make([]string, len(values))
		for i := range values {
			placeholders[i] = fmt.Sprintf("$%d", startIndex+i)
		}
		return fmt.Sprintf("%s IN (%s)", clause.Field, strings.Join(placeholders, ", ")), values, nil

	case core.OpNotIn:
		values, ok := clause.Value.([]interface{})
		if !ok {
			return "", nil, fmt.Errorf("NOT IN operator requires []interface{} value")
		}
		placeholders := make([]string, len(values))
		for i := range values {
			placeholders[i] = fmt.Sprintf("$%d", startIndex+i)
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

// scanRow scans a row into a map with known columns
func (p *PostgresAdapter) scanRow(row pgx.Row, columns []string) (map[string]interface{}, error) {
	values := make([]interface{}, len(columns))
	valuePtrs := make([]interface{}, len(columns))
	for i := range values {
		valuePtrs[i] = &values[i]
	}

	if err := row.Scan(valuePtrs...); err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	result := make(map[string]interface{})
	for i, col := range columns {
		result[col] = values[i]
	}

	return result, nil
}

// scanRowsDynamic scans multiple rows with unknown columns
func (p *PostgresAdapter) scanRowsDynamic(rows pgx.Rows) (map[string]interface{}, error) {
	values, err := rows.Values()
	if err != nil {
		return nil, err
	}

	fields := rows.FieldDescriptions()
	result := make(map[string]interface{})

	for i, field := range fields {
		if i < len(values) {
			result[field.Name] = values[i]
		}
	}

	return result, nil
}

// postgresTransaction wraps a PostgreSQL transaction
type postgresTransaction struct {
	tx pgx.Tx
}

func (t *postgresTransaction) ID() string {
	return "postgres-tx"
}

func (t *postgresTransaction) Create(ctx context.Context, model string, data map[string]interface{}) (map[string]interface{}, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("no data provided")
	}

	columns := make([]string, 0, len(data))
	placeholders := make([]string, 0, len(data))
	values := make([]interface{}, 0, len(data))
	returning := make([]string, 0, len(data))

	i := 1
	for col, val := range data {
		columns = append(columns, col)
		placeholders = append(placeholders, fmt.Sprintf("$%d", i))
		values = append(values, val)
		returning = append(returning, col)
		i++
	}

	query := fmt.Sprintf(
		"INSERT INTO %s (%s) VALUES (%s) RETURNING %s",
		model,
		strings.Join(columns, ", "),
		strings.Join(placeholders, ", "),
		strings.Join(returning, ", "),
	)

	row := t.tx.QueryRow(ctx, query, values...)
	return scanRowTx(row, returning)
}

func (t *postgresTransaction) FindOne(ctx context.Context, query *core.Query) (map[string]interface{}, error) {
	// Build and execute query using tx instead of pool
	sql, args, err := buildSelectQueryTx(query, true)
	if err != nil {
		return nil, err
	}

	rows, err := t.tx.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	if !rows.Next() {
		return nil, nil
	}

	values, err := rows.Values()
	if err != nil {
		return nil, err
	}

	fields := rows.FieldDescriptions()
	result := make(map[string]interface{})

	for i, field := range fields {
		if i < len(values) {
			result[field.Name] = values[i]
		}
	}

	return result, nil
}

func (t *postgresTransaction) FindMany(ctx context.Context, query *core.Query) ([]map[string]interface{}, error) {
	sql, args, err := buildSelectQueryTx(query, false)
	if err != nil {
		return nil, err
	}

	rows, err := t.tx.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	results := make([]map[string]interface{}, 0)
	for rows.Next() {
		values, err := rows.Values()
		if err != nil {
			return nil, err
		}

		fields := rows.FieldDescriptions()
		result := make(map[string]interface{})

		for i, field := range fields {
			if i < len(values) {
				result[field.Name] = values[i]
			}
		}
		results = append(results, result)
	}

	return results, rows.Err()
}

func (t *postgresTransaction) Update(ctx context.Context, query *core.Query, data map[string]interface{}) (map[string]interface{}, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("no data provided")
	}

	setClauses := make([]string, 0, len(data))
	values := make([]interface{}, 0, len(data))

	i := 1
	for col, val := range data {
		setClauses = append(setClauses, fmt.Sprintf("%s = $%d", col, i))
		values = append(values, val)
		i++
	}

	whereClause, whereArgs, err := buildWhereClauseTx(query.Where, i)
	if err != nil {
		return nil, err
	}
	values = append(values, whereArgs...)

	sql := fmt.Sprintf(
		"UPDATE %s SET %s%s RETURNING *",
		query.Model,
		strings.Join(setClauses, ", "),
		whereClause,
	)

	rows, err := t.tx.Query(ctx, sql, values...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	if !rows.Next() {
		return nil, nil
	}

	rowValues, err := rows.Values()
	if err != nil {
		return nil, err
	}

	fields := rows.FieldDescriptions()
	result := make(map[string]interface{})

	for i, field := range fields {
		if i < len(rowValues) {
			result[field.Name] = rowValues[i]
		}
	}

	return result, nil
}

func (t *postgresTransaction) UpdateMany(ctx context.Context, query *core.Query, data map[string]interface{}) (int64, error) {
	if len(data) == 0 {
		return 0, fmt.Errorf("no data provided")
	}

	setClauses := make([]string, 0, len(data))
	values := make([]interface{}, 0, len(data))

	i := 1
	for col, val := range data {
		setClauses = append(setClauses, fmt.Sprintf("%s = $%d", col, i))
		values = append(values, val)
		i++
	}

	whereClause, whereArgs, err := buildWhereClauseTx(query.Where, i)
	if err != nil {
		return 0, err
	}
	values = append(values, whereArgs...)

	sql := fmt.Sprintf(
		"UPDATE %s SET %s%s",
		query.Model,
		strings.Join(setClauses, ", "),
		whereClause,
	)

	result, err := t.tx.Exec(ctx, sql, values...)
	if err != nil {
		return 0, err
	}

	return result.RowsAffected(), nil
}

func (t *postgresTransaction) Delete(ctx context.Context, query *core.Query) error {
	whereClause, args, err := buildWhereClauseTx(query.Where, 1)
	if err != nil {
		return err
	}

	sql := fmt.Sprintf("DELETE FROM %s%s", query.Model, whereClause)
	_, err = t.tx.Exec(ctx, sql, args...)
	return err
}

func (t *postgresTransaction) DeleteMany(ctx context.Context, query *core.Query) (int64, error) {
	whereClause, args, err := buildWhereClauseTx(query.Where, 1)
	if err != nil {
		return 0, err
	}

	sql := fmt.Sprintf("DELETE FROM %s%s", query.Model, whereClause)
	result, err := t.tx.Exec(ctx, sql, args...)
	if err != nil {
		return 0, err
	}

	return result.RowsAffected(), nil
}

func (t *postgresTransaction) Count(ctx context.Context, query *core.Query) (int64, error) {
	whereClause, args, err := buildWhereClauseTx(query.Where, 1)
	if err != nil {
		return 0, err
	}

	sql := fmt.Sprintf("SELECT COUNT(*) FROM %s%s", query.Model, whereClause)

	var count int64
	err = t.tx.QueryRow(ctx, sql, args...).Scan(&count)
	if err != nil {
		return 0, err
	}

	return count, nil
}

func (t *postgresTransaction) Transaction(ctx context.Context, fn func(core.Adapter) error) error {
	// Nested transactions not supported, just execute the function
	return fn(t)
}

func (t *postgresTransaction) Ping(ctx context.Context) error {
	return nil
}

func (t *postgresTransaction) Close() error {
	return nil
}

// Helper functions for transaction

func buildSelectQueryTx(query *core.Query, limit1 bool) (string, []interface{}, error) {
	whereClause, args, err := buildWhereClauseTx(query.Where, 1)
	if err != nil {
		return "", nil, err
	}

	sql := fmt.Sprintf("SELECT * FROM %s%s", query.Model, whereClause)

	if len(query.OrderBy) > 0 {
		orderClauses := make([]string, 0, len(query.OrderBy))
		for _, order := range query.OrderBy {
			direction := "ASC"
			if order.Desc {
				direction = "DESC"
			}
			orderClauses = append(orderClauses, fmt.Sprintf("%s %s", order.Field, direction))
		}
		sql += " ORDER BY " + strings.Join(orderClauses, ", ")
	}

	if limit1 {
		sql += " LIMIT 1"
	} else if query.Limit > 0 {
		sql += fmt.Sprintf(" LIMIT %d", query.Limit)
	}

	if query.Offset > 0 {
		sql += fmt.Sprintf(" OFFSET %d", query.Offset)
	}

	return sql, args, nil
}

func buildWhereClauseTx(where []core.WhereClause, startIndex int) (string, []interface{}, error) {
	if len(where) == 0 {
		return "", nil, nil
	}

	clauses := make([]string, 0, len(where))
	args := make([]interface{}, 0, len(where))
	index := startIndex

	for _, clause := range where {
		sql, clauseArgs, err := buildSingleWhereClauseTx(clause, index)
		if err != nil {
			return "", nil, err
		}
		clauses = append(clauses, sql)
		args = append(args, clauseArgs...)
		index += len(clauseArgs)
	}

	return " WHERE " + strings.Join(clauses, " AND "), args, nil
}

func buildSingleWhereClauseTx(clause core.WhereClause, startIndex int) (string, []interface{}, error) {
	switch clause.Operator {
	case core.OpEqual:
		return fmt.Sprintf("%s = $%d", clause.Field, startIndex), []interface{}{clause.Value}, nil
	case core.OpNotEqual:
		return fmt.Sprintf("%s != $%d", clause.Field, startIndex), []interface{}{clause.Value}, nil
	case core.OpGreaterThan:
		return fmt.Sprintf("%s > $%d", clause.Field, startIndex), []interface{}{clause.Value}, nil
	case core.OpGreaterOrEqual:
		return fmt.Sprintf("%s >= $%d", clause.Field, startIndex), []interface{}{clause.Value}, nil
	case core.OpLessThan:
		return fmt.Sprintf("%s < $%d", clause.Field, startIndex), []interface{}{clause.Value}, nil
	case core.OpLessOrEqual:
		return fmt.Sprintf("%s <= $%d", clause.Field, startIndex), []interface{}{clause.Value}, nil
	case core.OpLike:
		return fmt.Sprintf("%s LIKE $%d", clause.Field, startIndex), []interface{}{clause.Value}, nil
	case core.OpIn:
		values, ok := clause.Value.([]interface{})
		if !ok {
			return "", nil, fmt.Errorf("IN operator requires []interface{} value")
		}
		placeholders := make([]string, len(values))
		for i := range values {
			placeholders[i] = fmt.Sprintf("$%d", startIndex+i)
		}
		return fmt.Sprintf("%s IN (%s)", clause.Field, strings.Join(placeholders, ", ")), values, nil
	case core.OpNotIn:
		values, ok := clause.Value.([]interface{})
		if !ok {
			return "", nil, fmt.Errorf("NOT IN operator requires []interface{} value")
		}
		placeholders := make([]string, len(values))
		for i := range values {
			placeholders[i] = fmt.Sprintf("$%d", startIndex+i)
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

func scanRowTx(row pgx.Row, columns []string) (map[string]interface{}, error) {
	values := make([]interface{}, len(columns))
	valuePtrs := make([]interface{}, len(columns))
	for i := range values {
		valuePtrs[i] = &values[i]
	}

	if err := row.Scan(valuePtrs...); err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	result := make(map[string]interface{})
	for i, col := range columns {
		result[col] = values[i]
	}

	return result, nil
}


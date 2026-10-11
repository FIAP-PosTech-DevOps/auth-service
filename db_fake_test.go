package main

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"io"
	"sync"
	"testing"
)

// Banco falso para os testes, sem dependência externa: implementa o mínimo
// da interface database/sql/driver para responder ao QueryRow(...).Scan(&id)
// dos handlers. Cada teste decide o que a consulta devolve.

// fakeResult é a resposta de uma consulta: um id, nenhuma linha ou um erro.
type fakeResult struct {
	id     int64
	noRows bool
	err    error
}

type fakeDB struct {
	mu      sync.Mutex
	result  fakeResult
	queries []string
	args    [][]driver.Value
}

// newFakeDB devolve um *sql.DB ligado ao banco falso e o próprio fake, para
// o teste configurar a resposta e conferir o que foi consultado.
func newFakeDB(t *testing.T, r fakeResult) (*sql.DB, *fakeDB) {
	t.Helper()
	f := &fakeDB{result: r}
	db := sql.OpenDB(fakeConnector{f})
	t.Cleanup(func() { _ = db.Close() })
	return db, f
}

type fakeConnector struct{ f *fakeDB }

func (c fakeConnector) Connect(context.Context) (driver.Conn, error) { return &fakeConn{c.f}, nil }
func (c fakeConnector) Driver() driver.Driver                        { return fakeDriver{} }

type fakeDriver struct{}

func (fakeDriver) Open(string) (driver.Conn, error) { return nil, errors.New("use fakeConnector") }

type fakeConn struct{ f *fakeDB }

func (c *fakeConn) Prepare(query string) (driver.Stmt, error) { return &fakeStmt{c.f, query}, nil }
func (c *fakeConn) Close() error                              { return nil }
func (c *fakeConn) Begin() (driver.Tx, error)                 { return nil, errors.New("sem transações") }

type fakeStmt struct {
	f     *fakeDB
	query string
}

func (s *fakeStmt) Close() error  { return nil }
func (s *fakeStmt) NumInput() int { return -1 }
func (s *fakeStmt) Exec([]driver.Value) (driver.Result, error) {
	return nil, errors.New("Exec não usado pelos handlers")
}

func (s *fakeStmt) Query(args []driver.Value) (driver.Rows, error) {
	s.f.mu.Lock()
	defer s.f.mu.Unlock()
	s.f.queries = append(s.f.queries, s.query)
	s.f.args = append(s.f.args, args)
	if s.f.result.err != nil {
		return nil, s.f.result.err
	}
	return &fakeRows{id: s.f.result.id, done: s.f.result.noRows}, nil
}

type fakeRows struct {
	id   int64
	done bool
}

func (r *fakeRows) Columns() []string { return []string{"id"} }
func (r *fakeRows) Close() error      { return nil }
func (r *fakeRows) Next(dest []driver.Value) error {
	if r.done {
		return io.EOF
	}
	dest[0] = r.id
	r.done = true
	return nil
}

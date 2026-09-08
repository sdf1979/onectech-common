package onectechcommon

import (
	"bytes"
	"database/sql"
	"fmt"
	"hash/crc64"
	"os"
	"sort"
	"strconv"
	"strings"
	"unsafe"

	agg "github.com/sdf1979/onectech-common/aggregator"
	_ "modernc.org/sqlite"
)

type Store struct {
	db         *sql.DB
	tx         *sql.Tx
	crc64Table *crc64.Table
	cache      map[int64]string
	batchSize  int64
	countSave  int64
	buf        bytes.Buffer
	node       *Node

	txOpen             bool
	stmtInsertString   *sql.Stmt
	stmtInsertTtimeout *sql.Stmt
	stmtInsertTlock    *sql.Stmt
	stmtInsertSdbl     *sql.Stmt
}

type ttimeout struct {
	ID              int64
	EventTime       int64
	ProcessName     string
	OSThread        int64
	ConnectID       int64
	WaitConnections string
	HashContext     int64
}

type tlock struct {
	ID              int64
	EventTime       int64
	Duration        int64
	ProcessName     string
	OSThread        int64
	ConnectID       int64
	Regions         string
	Locks           string
	WaitConnections string
	HashContext     int64
}

type sdbl struct {
	ID          int64
	EventTime   int64
	ProcessName string
	OSThread    int64
	ConnectID   int64
	Func        string
	HashContext int64
}

type blockedDataRegion struct {
	region   string
	lockType string
	data     []map[string]string
}

func newBlockedDataRegion(data string) *blockedDataRegion {
	b := &blockedDataRegion{
		data: make([]map[string]string, 0),
	}

	idx := strings.Index(data, " ")
	b.region = data[:idx]
	data = data[idx+1:]

	idx = strings.Index(data, " ")
	b.lockType = data[:idx]
	data = data[idx+1:]

	for row := range strings.SplitSeq(data, ", ") {
		rowMap := make(map[string]string)
		for keyValueStr := range strings.SplitSeq(row, " ") {
			keyValue := strings.Split(keyValueStr, "=")
			rowMap[keyValue[0]] = keyValue[1]
		}
		b.data = append(b.data, rowMap)
	}

	return b
}

type blockedData struct {
	data map[string]*blockedDataRegion
}

func newBlockedData(tl tlock) *blockedData {
	bd := &blockedData{
		data: make(map[string]*blockedDataRegion),
	}

	regions := []struct {
		region   string
		idxStart int
		idxEnd   int
	}{}

	// Расчет начального и завершающего индекса в Locks по регионам блокировки
	for reg := range strings.SplitSeq(tl.Regions, ",") {
		regions = append(regions, struct {
			region   string
			idxStart int
			idxEnd   int
		}{reg, strings.Index(tl.Locks, reg), 0})
	}
	sort.Slice(regions, func(i, j int) bool {
		return regions[i].idxStart < regions[j].idxStart
	})
	for i := 0; i < len(regions)-1; i++ {
		regions[i].idxEnd = regions[i+1].idxStart - 1
	}
	regions[len(regions)-1].idxEnd = len(tl.Locks)

	// Обход регионов с получением заблокированных данных
	for _, reg := range regions {
		blockedDataRegion := newBlockedDataRegion(tl.Locks[reg.idxStart:reg.idxEnd])
		bd.data[reg.region] = blockedDataRegion
	}

	return bd
}

func (bdVictim *blockedData) isBlocked(bdSource *blockedData) bool {
	for _, bdrVictim := range bdVictim.data {
		bdrSource, ok := bdSource.data[bdrVictim.region]
		if !ok {
			continue
		}
		if bdrVictim.lockType == SHARED && bdrSource.lockType == SHARED {
			continue
		}

		for _, dVictim := range bdrVictim.data {
			for _, dSource := range bdrSource.data {
				numberMatches := 0
				for keyVictim, valueVictim := range dVictim {
					valueSource, ok := dSource[keyVictim]
					if !ok {
						return true
					}
					if valueVictim == valueSource {
						numberMatches++
					}
				}
				if numberMatches == len(dVictim) {
					return true
				}
			}
		}

		for _, dVictim := range bdrVictim.data {
			for _, dSource := range bdrSource.data {
				numberMatches := 0
				for keyVictim, valueVictim := range dVictim {
					valueSource, ok := dSource[keyVictim]
					if !ok {
						return true
					}
					if valueVictim == valueSource {
						numberMatches++
					}
				}
				if numberMatches == len(dVictim) {
					return true
				}
			}
		}
	}
	return false
}

func NewStore() *Store {
	return &Store{
		crc64Table: crc64.MakeTable(crc64.ECMA),
		cache:      make(map[int64]string),
		batchSize:  100000,
	}
}

func (s *Store) OpenNew() error {
	if err := s.open(); err != nil {
		return fmt.Errorf("error open database: %w", err)
	}
	if err := s.initTables(); err != nil {
		return fmt.Errorf("error init tables: %w", err)
	}
	if err := s.openTran(); err != nil {
		return fmt.Errorf("error open transaction: %w", err)
	}
	s.txOpen = true
	return nil
}

func (s *Store) Open() error {
	if err := s.open(); err != nil {
		return fmt.Errorf("error open database: %w", err)
	}
	return nil
}

func (s *Store) Close() {
	if s.txOpen {
		s.commitTran()
	}
	if err := s.createIndexes(); err != nil {
		defLog.Errf("Error create indexes: %v", err)
		os.Exit(1)
	}
	if err := s.db.Close(); err != nil {
		defLog.Errf("Error close db: %v", err)
	}
}

func (s *Store) SaveEvent(event *EventLog) {
	if s.countSave >= s.batchSize {
		if err := s.commitTran(); err != nil {
			defLog.Errf("Error commit transaction: %v", err)
		}
		if err := s.openTran(); err != nil {
			defLog.Errf("Error open transaction: %v", err)
		}
		s.countSave = 0
	}

	switch event.Name() {
	case TLOCK:
		s.saveTlock(event)
		s.countSave++
	case TTIMEOUT:
		s.saveTtimeout(event)
		s.countSave++
	case SDBL:
		s.saveSsbl(event)
		s.countSave++
	}
}

func (s *Store) TtimeoutAnalysis() error {

	s.node = NewNode(&agg.Counter{})

	ttimeouts, err := s.ttimeouts()
	if err != nil {
		return err
	}

	for _, ttimeoutVictim := range ttimeouts {

		tlVictim, err := s.victimTlock(ttimeoutVictim)
		if err != nil {
			continue
		}

		slSource, err := s.sourceTransactionBlocking(tlVictim)
		if err != nil {
			continue
		}

		tlSource, err := s.sourceTlock(tlVictim, slSource)
		if err != nil {
			continue
		}

		s.investigateTtimeout(tlVictim, tlSource)
	}

	/*s.node.SetDescr([]string{"",
	"Информационная база",
	"Пространство блокировок жертвы",
	"Контекст жертвы",
	"Пространство блокировок источника",
	"Контекст источника"})*/
	fmt.Println(s.node.ToHTML("Инф. база \\ Пространство блокировок жертвы \\ Контекст жертвы \\ Пространство блокировок источника \\ Контекст источника"))

	return nil
}

func (s *Store) investigateTtimeout(tlVictim tlock, tlSource []tlock) {

	posComma := strings.Index(tlVictim.WaitConnections, ",")
	if posComma != -1 {
		panic(fmt.Sprintf("WaitConnections=%s", tlVictim.WaitConnections))
	}

	bdVictim := newBlockedData(tlVictim)
	isAllBlocked := 0
	for _, tl := range tlSource {
		bdSource := newBlockedData(tl)
		isBlocked := bdVictim.isBlocked(bdSource)
		if isBlocked {
			contextVictim, _ := s.digest(tlVictim.HashContext)
			contextSource, _ := s.digest(tl.HashContext)
			s.node.Insert([]string{tlVictim.ProcessName, tlVictim.Regions, contextVictim, tl.Regions, contextSource}, agg.UpdateContext{Count: 1})
			isAllBlocked++
		}
	}
	if isAllBlocked == 0 || isAllBlocked > 1 {
		//TODO подумать что делать в случае, если виновник не найден или два виновника
	}
}

func (s *Store) ttimeouts() ([]ttimeout, error) {
	rows, err := s.db.Query(`
        SELECT id, event_time, p_process_name, os_thread, t_connectid, wait_connections, hash_context
        FROM ttimeout
        ORDER BY event_time
    `)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var timeouts []ttimeout
	for rows.Next() {
		var tt ttimeout
		err := rows.Scan(
			&tt.ID,
			&tt.EventTime,
			&tt.ProcessName,
			&tt.OSThread,
			&tt.ConnectID,
			&tt.WaitConnections,
			&tt.HashContext,
		)
		if err != nil {
			return nil, err
		}
		timeouts = append(timeouts, tt)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}
	return timeouts, nil
}

func (s *Store) victimTlock(tt ttimeout) (tlock, error) {
	var tl tlock
	row := s.db.QueryRow(`
        SELECT id, event_time, duration, p_process_name, os_thread, t_connectid, regions, locks, wait_connections, hash_context
        FROM tlock
        WHERE os_thread = ? AND t_connectid = ? AND p_process_name = ? AND event_time >= ?
		ORDER BY event_time
        LIMIT 1
    `, tt.OSThread, tt.ConnectID, tt.ProcessName, tt.EventTime)

	err := row.Scan(
		&tl.ID,
		&tl.EventTime,
		&tl.Duration,
		&tl.ProcessName,
		&tl.OSThread,
		&tl.ConnectID,
		&tl.Regions,
		&tl.Locks,
		&tl.WaitConnections,
		&tl.HashContext,
	)
	if err != nil {
		return tl, err
	}
	return tl, nil
}

func (s *Store) sourceTransactionBlocking(tl tlock) (sdbl, error) {
	var sl sdbl
	row := s.db.QueryRow(`
        SELECT id, event_time, p_process_name, os_thread, t_connectid, func, hash_context
        FROM sdbl
        WHERE t_connectid = ? AND p_process_name = ? AND func = ? AND event_time <= ?
		ORDER BY event_time DESC
        LIMIT 1
    `, tl.WaitConnections, tl.ProcessName, BEGIN_TRANSACTION, tl.EventTime-tl.Duration)

	err := row.Scan(
		&sl.ID,
		&sl.EventTime,
		&sl.ProcessName,
		&sl.OSThread,
		&sl.ConnectID,
		&sl.Func,
		&sl.HashContext,
	)
	if err != nil {
		return sl, err
	}
	return sl, nil
}

func (s *Store) sourceTlock(tlVictim tlock, slSource sdbl) ([]tlock, error) {
	condition, args := s.buildLikeCondition(tlVictim.Regions)

	query := `SELECT id, event_time, duration, p_process_name, os_thread, t_connectid, regions, locks, wait_connections, hash_context
        FROM tlock
        WHERE os_thread = ? AND t_connectid = ? AND p_process_name = ? AND event_time >= ? AND event_time <= ?
		` + condition + `
		ORDER BY event_time desc`

	allArgs := []any{
		slSource.OSThread, slSource.ConnectID, slSource.ProcessName, slSource.EventTime, tlVictim.EventTime - tlVictim.Duration,
	}

	allArgs = append(allArgs, args...)

	rows, err := s.db.Query(query, allArgs...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tlocks []tlock
	for rows.Next() {
		var tl tlock
		err := rows.Scan(
			&tl.ID,
			&tl.EventTime,
			&tl.Duration,
			&tl.ProcessName,
			&tl.OSThread,
			&tl.ConnectID,
			&tl.Regions,
			&tl.Locks,
			&tl.WaitConnections,
			&tl.HashContext,
		)
		if err != nil {
			return nil, err
		}
		tlocks = append(tlocks, tl)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}
	return tlocks, nil
}

func (s *Store) buildLikeCondition(region string) (string, []any) {
	prefixes := strings.Split(region, ",")

	placeholders := make([]string, len(prefixes))
	args := make([]any, len(prefixes))
	for i, p := range prefixes {
		placeholders[i] = "regions LIKE ?"
		args[i] = "%" + p + "%"
	}

	condition := "AND (" + strings.Join(placeholders, " OR ") + ")"
	return condition, args
}

func (s *Store) open() error {
	db, err := sql.Open("sqlite", "tmp.db")
	if err != nil {
		return err
	}
	s.db = db

	_, err = db.Exec("PRAGMA synchronous = OFF")
	if err != nil {
		return err
	}
	_, err = db.Exec("PRAGMA journal_mode = MEMORY")
	if err != nil {
		return err
	}
	_, err = db.Exec("PRAGMA cache_size = -1048576")
	if err != nil {
		return err
	}
	_, err = db.Exec("PRAGMA temp_store = MEMORY")
	if err != nil {
		return err
	}
	return nil
}

func (s *Store) closeStmt(stmt *sql.Stmt) {
	if stmt != nil {
		if err := stmt.Close(); err != nil {
			defLog.Errf("%v", err)
		}
	}
}

func (s *Store) openTran() error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	s.tx = tx

	queryContext := `INSERT INTO strings (hash, data) VALUES (?, ?)`
	stmtInsertContext, err := s.tx.Prepare(queryContext)
	if err != nil {
		return err
	}
	s.stmtInsertString = stmtInsertContext

	queryTtimeout := `INSERT INTO ttimeout (event_time, p_process_name, os_thread, t_connectid, wait_connections, hash_context) VALUES (?, ?, ?, ?, ?, ?)`
	stmtInserTtimeout, err := s.tx.Prepare(queryTtimeout)
	if err != nil {
		return err
	}
	s.stmtInsertTtimeout = stmtInserTtimeout

	queryTlock := `INSERT INTO tlock (event_time, duration, p_process_name, os_thread, t_connectid, regions, locks, wait_connections, hash_context) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`
	stmtInsertTlock, err := s.tx.Prepare(queryTlock)
	if err != nil {
		return err
	}
	s.stmtInsertTlock = stmtInsertTlock

	querySdbl := `INSERT INTO sdbl (event_time, p_process_name, os_thread, t_connectid, func, hash_context) VALUES (?, ?, ?, ?, ?, ?)`
	stmtInsertSdbl, err := s.tx.Prepare(querySdbl)
	if err != nil {
		return err
	}
	s.stmtInsertSdbl = stmtInsertSdbl

	return nil
}

func (s *Store) commitTran() error {
	s.closeStmt(s.stmtInsertString)
	s.closeStmt(s.stmtInsertTtimeout)
	s.closeStmt(s.stmtInsertTlock)
	s.closeStmt(s.stmtInsertSdbl)

	if err := s.tx.Commit(); err != nil {
		return err
	}

	return nil
}

func (s *Store) saveTtimeout(event *EventLog) error {
	hash, err := s.hashString(event.Value(CONTEXT))
	if err != nil {
		return err
	}
	osThread, err := strconv.ParseInt(event.Value(OS_THREAD), 10, 64)
	if err != nil {
		return err
	}
	tConnectId, err := strconv.ParseInt(event.Value(T_CONNECT_ID), 10, 64)
	if err != nil {
		return err
	}

	_, err = s.stmtInsertTtimeout.Exec(
		event.Time().UnixMicro(),
		event.Value(P_PROCESS_NAME),
		osThread,
		tConnectId,
		event.Value(WAIT_CONNECTION),
		int64(hash))
	return err
}

func (s *Store) saveTlock(event *EventLog) error {
	hash, err := s.hashString(event.Value(CONTEXT))
	if err != nil {
		return err
	}
	osThread, err := strconv.ParseInt(event.Value(OS_THREAD), 10, 64)
	if err != nil {
		return err
	}
	tConnectId, err := strconv.ParseInt(event.Value(T_CONNECT_ID), 10, 64)
	if err != nil {
		return err
	}

	_, err = s.stmtInsertTlock.Exec(
		event.Time().UnixMicro(),
		event.Duration(),
		event.Value(P_PROCESS_NAME),
		osThread,
		tConnectId,
		TrimQuotedString(event.Value(REGIONS)),
		TrimQuotedString(event.Value(LOCKS)),
		event.Value(WAIT_CONNECTION),
		int64(hash))
	return err
}

func (s *Store) saveSsbl(event *EventLog) error {

	if event.Value(FUNC) != BEGIN_TRANSACTION {
		return nil
	}

	hash, err := s.hashString(event.Value(CONTEXT))
	if err != nil {
		return err
	}
	osThread, err := strconv.ParseInt(event.Value(OS_THREAD), 10, 64)
	if err != nil {
		return err
	}
	tConnectId, err := strconv.ParseInt(event.Value(T_CONNECT_ID), 10, 64)
	if err != nil {
		return err
	}

	_, err = s.stmtInsertSdbl.Exec(
		event.Time().UnixMicro(),
		event.Value(P_PROCESS_NAME),
		osThread,
		tConnectId,
		event.Value(FUNC),
		int64(hash))
	return err
}

func (s *Store) hashString(context string) (int64, error) {
	contextTrim := TrimQuotedString(context)
	data := unsafe.Slice(unsafe.StringData(contextTrim), len(contextTrim))
	hash := int64(crc64.Checksum(data, s.crc64Table))
	if _, ok := s.cache[hash]; !ok {
		if err := s.insertString(hash, contextTrim); err != nil {
			return 0, fmt.Errorf("error writing string to the database:%w", err)
		}
		s.cache[hash] = contextTrim
	}
	return hash, nil
}

func (s *Store) digest(hash int64) (string, error) {
	if data, ok := s.cache[hash]; !ok {
		str, err := s.selectString(hash)
		if err != nil {
			return data, err
		}
		s.cache[hash] = str
		return str, nil
	} else {
		return data, nil
	}
}

func (s *Store) insertString(hash int64, context string) error {
	_, err := s.stmtInsertString.Exec(hash, context)
	return err
}

func (s *Store) selectString(hash int64) (string, error) {
	var data string
	row := s.db.QueryRow(`SELECT data FROM strings WHERE hash = ?`, hash)

	err := row.Scan(
		&data,
	)
	if err != nil {
		return data, err
	}
	return data, nil
}

func (s *Store) initTables() error {
	query := `
    DROP TABLE IF EXISTS strings;
	DROP TABLE IF EXISTS ttimeout;
	DROP TABLE IF EXISTS tlock;
	DROP TABLE IF EXISTS sdbl;
	CREATE TABLE IF NOT EXISTS strings (hash INTEGER PRIMARY KEY, data TEXT NOT NULL) WITHOUT ROWID;
	CREATE TABLE IF NOT EXISTS ttimeout (
		id INTEGER PRIMARY KEY,
        event_time INTEGER NOT NULL,
		p_process_name TEXT NOT NULL,
		os_thread INTEGER NOT NULL,
		t_connectid INTEGER NOT NULL,
		wait_connections TEXT NOT NULL,
        hash_context INTEGER NOT NULL
    );
	CREATE TABLE IF NOT EXISTS tlock (
		id INTEGER PRIMARY KEY,
        event_time INTEGER NOT NULL,
		duration INTEGER NOT NULL,
		p_process_name TEXT NOT NULL,
		os_thread INTEGER NOT NULL,
		t_connectid INTEGER NOT NULL,
		regions INTEGER NOT NULL,
		locks INTEGER NOT NULL,
		wait_connections TEXT NOT NULL,
        hash_context INTEGER NOT NULL
    );
	CREATE TABLE IF NOT EXISTS sdbl (
		id INTEGER PRIMARY KEY,
        event_time INTEGER NOT NULL,
		p_process_name TEXT NOT NULL,
		os_thread INTEGER NOT NULL,
		t_connectid INTEGER NOT NULL,
		func INTEGER NOT NULL,
		hash_context INTEGER NOT NULL
    );
    `
	_, err := s.db.Exec(query)
	if err != nil {
		return err
	}

	return nil
}

func (s *Store) createIndexes() error {
	queries := []string{
		`CREATE INDEX IF NOT EXISTS idx_ttimeout ON ttimeout (t_connectid, os_thread, p_process_name, event_time);`,
		`CREATE INDEX IF NOT EXISTS idx_tlock ON tlock (t_connectid, os_thread, p_process_name, event_time);`,
		`CREATE INDEX IF NOT EXISTS idx_sdbl ON sdbl (t_connectid, p_process_name, func, event_time);`,
	}
	for _, q := range queries {
		if _, err := s.db.Exec(q); err != nil {
			return err
		}
	}
	return nil
}

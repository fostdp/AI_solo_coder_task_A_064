package repository

import (
	"database/sql"
	"fmt"
	"power-twin-backend/internal/model"

	_ "github.com/mattn/go-sqlite3"
)

type SQLiteRepo struct {
	DB *sql.DB
}

func NewSQLiteRepo(dbPath string) (*SQLiteRepo, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}
	repo := &SQLiteRepo{DB: db}
	err = repo.InitSchema()
	if err != nil {
		return nil, err
	}
	err = repo.SeedTopologyData()
	if err != nil {
		return nil, err
	}
	return repo, nil
}

func (r *SQLiteRepo) InitSchema() error {
	statements := []string{
		`CREATE TABLE IF NOT EXISTS lines (id TEXT PRIMARY KEY, name TEXT, substation_count INTEGER)`,
		`CREATE TABLE IF NOT EXISTS substations (id TEXT PRIMARY KEY, line_id TEXT, name TEXT, pos_x REAL, pos_y REAL, pos_z REAL)`,
		`CREATE TABLE IF NOT EXISTS rectifiers (id TEXT PRIMARY KEY, substation_id TEXT, name TEXT, rated_power REAL)`,
		`CREATE TABLE IF NOT EXISTS dc_switchgears (id TEXT PRIMARY KEY, substation_id TEXT, name TEXT, rated_current REAL)`,
		`CREATE TABLE IF NOT EXISTS feeders (id TEXT PRIMARY KEY, source_id TEXT, target_id TEXT, impedance_r REAL, impedance_x REAL, rated_current REAL)`,
		`CREATE TABLE IF NOT EXISTS alarms (id TEXT PRIMARY KEY, level INTEGER, type TEXT, device_id TEXT, message TEXT, timestamp DATETIME, acknowledged INTEGER DEFAULT 0)`,
		`CREATE TABLE IF NOT EXISTS operation_history (id INTEGER PRIMARY KEY AUTOINCREMENT, device_id TEXT, operation TEXT, operator TEXT, result TEXT, timestamp DATETIME DEFAULT CURRENT_TIMESTAMP)`,
	}
	for _, s := range statements {
		_, err := r.DB.Exec(s)
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *SQLiteRepo) SeedTopologyData() error {
	var count int
	r.DB.QueryRow("SELECT COUNT(*) FROM lines").Scan(&count)
	if count > 0 {
		return nil
	}

	tx, err := r.DB.Begin()
	if err != nil {
		return err
	}

	lines := []model.Line{
		{ID: "line1", Name: "1号线", SubstationCount: 20},
		{ID: "line2", Name: "2号线", SubstationCount: 20},
		{ID: "line3", Name: "3号线", SubstationCount: 20},
	}
	for _, l := range lines {
		_, err := tx.Exec("INSERT INTO lines (id, name, substation_count) VALUES (?, ?, ?)", l.ID, l.Name, l.SubstationCount)
		if err != nil {
			tx.Rollback()
			return err
		}
	}

	zValues := map[string]float64{"line1": 0, "line2": 8, "line3": 16}
	rectifierCount := 0
	switchgearCount := 0
	substationIDs := make(map[string][]string)

	for li, line := range lines {
		var subIDs []string
		for si := 0; si < 20; si++ {
			subID := fmt.Sprintf("sub_%d_%d", li+1, si+1)
			subName := fmt.Sprintf("%s牵引变电所%02d", line.Name, si+1)
			posX := float64(si) * 1.5
			posY := float64(li) * 2.0
			posZ := zValues[line.ID]
			_, err := tx.Exec("INSERT INTO substations (id, line_id, name, pos_x, pos_y, pos_z) VALUES (?, ?, ?, ?, ?, ?)",
				subID, line.ID, subName, posX, posY, posZ)
			if err != nil {
				tx.Rollback()
				return err
			}
			subIDs = append(subIDs, subID)

			numRect := 3
			globalIdx := li*20 + si
			if globalIdx < 20 {
				numRect = 4
			}
			if rectifierCount+numRect > 200 {
				numRect = 200 - rectifierCount
			}
			if numRect < 0 {
				numRect = 0
			}
			for ri := 0; ri < numRect; ri++ {
				rectID := fmt.Sprintf("rect_%d", rectifierCount+1)
				rectName := fmt.Sprintf("%s整流器%d", subName, ri+1)
				_, err := tx.Exec("INSERT INTO rectifiers (id, substation_id, name, rated_power) VALUES (?, ?, ?, ?)",
					rectID, subID, rectName, 3000.0)
				if err != nil {
					tx.Rollback()
					return err
				}
				rectifierCount++
			}

			numSw := 7
			if globalIdx >= 40 {
				numSw = 6
			}
			if switchgearCount+numSw > 400 {
				numSw = 400 - switchgearCount
			}
			if numSw < 0 {
				numSw = 0
			}
			for swi := 0; swi < numSw; swi++ {
				swID := fmt.Sprintf("sw_%d", switchgearCount+1)
				swName := fmt.Sprintf("%s直流开关%d", subName, swi+1)
				_, err := tx.Exec("INSERT INTO dc_switchgears (id, substation_id, name, rated_current) VALUES (?, ?, ?, ?)",
					swID, subID, swName, 4000.0)
				if err != nil {
					tx.Rollback()
					return err
				}
				switchgearCount++
			}
		}
		substationIDs[line.ID] = subIDs
	}

	feederCount := 0
	for _, line := range lines {
		subs := substationIDs[line.ID]
		for i := 0; i < len(subs)-1; i++ {
			feederID := fmt.Sprintf("feeder_%d", feederCount+1)
			impedanceR := 0.02 + float64(i%5)*0.005
			impedanceX := 0.01 + float64(i%3)*0.003
			_, err := tx.Exec("INSERT INTO feeders (id, source_id, target_id, impedance_r, impedance_x, rated_current) VALUES (?, ?, ?, ?, ?, ?)",
				feederID, subs[i], subs[i+1], impedanceR, impedanceX, 4000.0)
			if err != nil {
				tx.Rollback()
				return err
			}
			feederCount++
		}
	}

	return tx.Commit()
}

func (r *SQLiteRepo) GetTopology() ([]model.TopologyNode, []model.TopologyEdge, error) {
	var nodes []model.TopologyNode
	var edges []model.TopologyEdge

	rows, err := r.DB.Query("SELECT id, line_id, name, pos_x, pos_y, pos_z FROM substations")
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var s model.Substation
		err := rows.Scan(&s.ID, &s.LineID, &s.Name, &s.PosX, &s.PosY, &s.PosZ)
		if err != nil {
			return nil, nil, err
		}
		node := model.TopologyNode{
			ID:       s.ID,
			Name:     s.Name,
			Type:     "substation",
			LineID:   s.LineID,
			Position: [3]float64{s.PosX, s.PosY, s.PosZ},
			LoadRate: 0,
			Status:   "normal",
		}
		nodes = append(nodes, node)
	}

	rows2, err := r.DB.Query("SELECT id, source_id, target_id, impedance_r, impedance_x, rated_current FROM feeders")
	if err != nil {
		return nil, nil, err
	}
	defer rows2.Close()
	for rows2.Next() {
		var f model.Feeder
		err := rows2.Scan(&f.ID, &f.SourceID, &f.TargetID, &f.ImpedanceR, &f.ImpedanceX, &f.RatedCurrent)
		if err != nil {
			return nil, nil, err
		}
		edge := model.TopologyEdge{
			ID:       f.ID,
			Source:   f.SourceID,
			Target:   f.TargetID,
			Type:     "feeder",
			LoadRate: 0,
			Status:   "normal",
		}
		edges = append(edges, edge)
	}

	return nodes, edges, nil
}

func (r *SQLiteRepo) InsertAlarm(alarm model.Alarm) error {
	_, err := r.DB.Exec("INSERT INTO alarms (id, level, type, device_id, message, timestamp, acknowledged) VALUES (?, ?, ?, ?, ?, ?, ?)",
		alarm.ID, alarm.Level, alarm.Type, alarm.DeviceID, alarm.Message, alarm.Timestamp, alarm.Acknowledged)
	return err
}

func (r *SQLiteRepo) GetAlarms(acknowledged bool) ([]model.Alarm, error) {
	var alarms []model.Alarm
	var rows *sql.Rows
	var err error
	if acknowledged {
		rows, err = r.DB.Query("SELECT id, level, type, device_id, message, timestamp, acknowledged FROM alarms WHERE acknowledged = 1 ORDER BY timestamp DESC")
	} else {
		rows, err = r.DB.Query("SELECT id, level, type, device_id, message, timestamp, acknowledged FROM alarms WHERE acknowledged = 0 ORDER BY timestamp DESC")
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var a model.Alarm
		var ack int
		err := rows.Scan(&a.ID, &a.Level, &a.Type, &a.DeviceID, &a.Message, &a.Timestamp, &ack)
		if err != nil {
			return nil, err
		}
		a.Acknowledged = ack == 1
		alarms = append(alarms, a)
	}
	return alarms, nil
}

func (r *SQLiteRepo) AcknowledgeAlarm(id string) error {
	_, err := r.DB.Exec("UPDATE alarms SET acknowledged = 1 WHERE id = ?", id)
	return err
}

func (r *SQLiteRepo) InsertOperationHistory(deviceID, operation, operator, result string) error {
	_, err := r.DB.Exec("INSERT INTO operation_history (device_id, operation, operator, result) VALUES (?, ?, ?, ?)",
		deviceID, operation, operator, result)
	return err
}

func (r *SQLiteRepo) GetOperationHistory(deviceID string) ([]map[string]interface{}, error) {
	rows, err := r.DB.Query("SELECT id, device_id, operation, operator, result, timestamp FROM operation_history WHERE device_id = ? ORDER BY timestamp DESC", deviceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []map[string]interface{}
	for rows.Next() {
		var id int
		var devID, op, operator, res, ts string
		err := rows.Scan(&id, &devID, &op, &operator, &res, &ts)
		if err != nil {
			return nil, err
		}
		m := map[string]interface{}{
			"id":        id,
			"device_id": devID,
			"operation": op,
			"operator":  operator,
			"result":    res,
			"timestamp": ts,
		}
		result = append(result, m)
	}
	return result, nil
}

func (r *SQLiteRepo) GetFeeders() ([]model.Feeder, error) {
	rows, err := r.DB.Query("SELECT id, source_id, target_id, impedance_r, impedance_x, rated_current FROM feeders")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var feeders []model.Feeder
	for rows.Next() {
		var f model.Feeder
		err := rows.Scan(&f.ID, &f.SourceID, &f.TargetID, &f.ImpedanceR, &f.ImpedanceX, &f.RatedCurrent)
		if err != nil {
			return nil, err
		}
		feeders = append(feeders, f)
	}
	return feeders, nil
}

func (r *SQLiteRepo) GetSubstations() ([]model.Substation, error) {
	rows, err := r.DB.Query("SELECT id, line_id, name, pos_x, pos_y, pos_z FROM substations")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var subs []model.Substation
	for rows.Next() {
		var s model.Substation
		err := rows.Scan(&s.ID, &s.LineID, &s.Name, &s.PosX, &s.PosY, &s.PosZ)
		if err != nil {
			return nil, err
		}
		subs = append(subs, s)
	}
	return subs, nil
}

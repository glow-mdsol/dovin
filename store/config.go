package store

func (s *Store) ConfigGet(key string) (string, error) {
	var val string
	err := s.db.QueryRow(`SELECT value FROM config WHERE key = ?`, key).Scan(&val)
	return val, err
}

func (s *Store) ConfigSet(key, value string) error {
	_, err := s.db.Exec(
		`INSERT INTO config(key,value) VALUES(?,?) ON CONFLICT(key) DO UPDATE SET value=excluded.value`,
		key, value,
	)
	return err
}

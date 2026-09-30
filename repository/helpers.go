package repository

import "gorm.io/gorm"

// requireRowsAffected returns the statement's error, or gorm.ErrRecordNotFound
// when it matched no rows.
func requireRowsAffected(res *gorm.DB) error {
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

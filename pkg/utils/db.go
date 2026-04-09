package utils

import (
	"errors"

	"gorm.io/gorm"
)

// FirstOrNil can accept any GORM query and return nil if the record is not found.
func FirstOrNil[T any](query *gorm.DB) (*T, error) {
	var dest T
	
	err := query.First(&dest).Error
	
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err 
	}
	
	return &dest, nil
}	

// FirstorFail can throw custom errors.
func FirstOrFail[T any](query *gorm.DB, customErr error) (*T, error) {
	var dest T
	
	err := query.First(&dest).Error
	
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, customErr
		}
		return nil, err
	}
	
	return &dest, nil
}